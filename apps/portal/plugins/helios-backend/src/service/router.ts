import { Router } from 'express';
import expressPromiseRouter from 'express-promise-router';
import * as k8s from '@kubernetes/client-node';
import { Config } from '@backstage/config';
import { LoggerService } from '@backstage/backend-plugin-api';

export interface RouterOptions {
  logger: LoggerService;
  config: Config;
}

function isNotFoundError(err: unknown): boolean {
  const statusCode =
    (err as { statusCode?: number; response?: { statusCode?: number } })
      ?.statusCode ??
    (err as { response?: { statusCode?: number } })?.response?.statusCode ??
    (err as { body?: { code?: number } })?.body?.code;

  if (statusCode === 404) {
    return true;
  }

  const msg = String(err).toLowerCase();
  return (
    msg.includes('not found') ||
    (msg.includes('status code') && msg.includes('404'))
  );
}

/** Matches Helm operator: GetDatabaseSecretName / GetDatabaseHost (traits.go). */
const databaseSecretNames = (componentName: string) => [
  `${componentName}-db-secret`,
  `${componentName}-backend-db-secret`,
];
const databasePodLabels = (componentName: string) => [
  `app=${componentName}-db`,
  `app=${componentName}-backend-db`,
];

function parseTraitProperties(raw: unknown): Record<string, unknown> | null {
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    return raw as Record<string, unknown>;
  }
  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw) as Record<string, unknown>;
    } catch {
      return null;
    }
  }
  return null;
}

/** Reads dbName from database trait if set; else operator default `${component}-db`. */
function resolveDbNameFromHeliosApp(
  body: unknown,
  componentName: string,
): string {
  const fallback = `${componentName}-db`;
  const spec = (body as { spec?: { components?: unknown[] } })?.spec;
  const components = spec?.components;
  if (!Array.isArray(components)) {
    return fallback;
  }
  for (const comp of components) {
    const traits = (comp as { traits?: unknown[] })?.traits;
    if (!Array.isArray(traits)) continue;
    for (const trait of traits) {
      const t = trait as { type?: string; properties?: unknown };
      if (String(t?.type).toLowerCase() !== 'database') continue;
      const props = parseTraitProperties(t.properties);
      const dbName = props?.dbName;
      if (typeof dbName === 'string' && dbName.length > 0) {
        return dbName;
      }
    }
  }
  return fallback;
}

export async function createRouter(options: RouterOptions): Promise<Router> {
  const { logger } = options;
  const router = expressPromiseRouter();

  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CoreV1Api);
  const customApi = kc.makeApiClient(k8s.CustomObjectsApi);

  router.get('/health', (_, response) => {
    response.json({ status: 'ok' });
  });

  router.get('/info/:componentName', async (req, res) => {
    const { componentName } = req.params;
    const namespace = 'default';
    const secretCandidates = databaseSecretNames(componentName);

    let dbName = `${componentName}-db`;
    try {
      const heliosApp: unknown = await customApi.getNamespacedCustomObject({
        group: 'app.helios.io',
        version: 'v1alpha1',
        namespace,
        plural: 'heliosapps',
        name: componentName,
      });
      const heliosAppBody = (heliosApp as { body?: unknown }).body ?? heliosApp;
      dbName = resolveDbNameFromHeliosApp(heliosAppBody, componentName);
    } catch {
      // HeliosApp CR may be absent; keep default db name.
    }

    let secret: k8s.V1Secret | undefined;
    let resolvedSecretName = '';
    for (const secretName of secretCandidates) {
      try {
        const read = await k8sApi.readNamespacedSecret({
          name: secretName,
          namespace,
        });
        secret =
          (read as { body?: k8s.V1Secret }).body ?? (read as k8s.V1Secret);
        resolvedSecretName = secretName;
        break;
      } catch (err: unknown) {
        if (!isNotFoundError(err)) {
          logger.error(
            `Failed to read database secret ${secretName}: ${String(err)}`,
          );
          return res.status(500).json({
            error: err instanceof Error ? err.message : String(err),
          });
        }
      }
    }

    if (!secret) {
      return res.status(404).json({
        error: `No database secret found in namespace ${namespace}. Tried: ${secretCandidates.join(
          ', ',
        )}`,
      });
    }

    const data = secret?.data ?? {};
    const decode = (val?: string) =>
      val ? Buffer.from(val, 'base64').toString() : '';

    let status: 'Running' | 'Failed' | 'Unknown' = 'Unknown';
    try {
      for (const labelSelector of databasePodLabels(componentName)) {
        const pods: unknown = await k8sApi.listNamespacedPod({
          namespace,
          labelSelector,
        });
        const podList = (pods as { body?: k8s.V1PodList }).body ?? pods;
        const items = (podList as k8s.V1PodList)?.items;
        if (!items || items.length === 0) {
          continue;
        }

        const phase = items[0].status?.phase;
        if (phase === 'Running') status = 'Running';
        else if (phase === 'Failed') status = 'Failed';
        break;
      }
    } catch {
      // Pod listing is best-effort.
    }

    const portStr = decode(data.DB_PORT);
    const port = portStr ? parseInt(portStr, 10) : 5432;

    return res.json({
      secretName: resolvedSecretName,
      host: decode(data.DB_HOST),
      port,
      user: decode(data.DB_USER),
      password: decode(data.DB_PASS),
      database: decode(data.DB_NAME) || dbName,
      status,
    });
  });

  // ----------------------------------------------------------------
  // GET /status/:componentName — HeliosApp CRD real-time status
  // ----------------------------------------------------------------
  router.get('/status/:componentName', async (req, res) => {
    const { componentName } = req.params;
    const namespace =
      typeof req.query.namespace === 'string' ? req.query.namespace : 'default';

    try {
      const raw: unknown = await customApi.getNamespacedCustomObject({
        group: 'app.helios.io',
        version: 'v1alpha1',
        namespace,
        plural: 'heliosapps',
        name: componentName,
      });

      // The client may return { body: … } or the object directly.
      const obj = (raw as { body?: Record<string, unknown> }).body ?? raw;
      const body = obj as Record<string, unknown>;

      const metadata = (body.metadata ?? {}) as Record<string, unknown>;
      const spec = (body.spec ?? {}) as Record<string, unknown>;
      const status = (body.status ?? {}) as Record<string, unknown>;

      // Extract conditions ([]metav1.Condition)
      const rawConditions = Array.isArray(status.conditions)
        ? status.conditions
        : [];
      const conditions = rawConditions.map((c: Record<string, unknown>) => ({
        type: String(c.type ?? ''),
        status: String(c.status ?? ''),
        reason: String(c.reason ?? ''),
        message: String(c.message ?? ''),
        lastTransitionTime: String(c.lastTransitionTime ?? ''),
      }));

      // Extract resourcesCreated ([]ResourceRef)
      const rawResources = Array.isArray(status.resourcesCreated)
        ? status.resourcesCreated
        : [];
      const resourcesCreated = rawResources.map(
        (r: Record<string, unknown>) => ({
          apiVersion: String(r.apiVersion ?? ''),
          kind: String(r.kind ?? ''),
          name: String(r.name ?? ''),
          namespace: r.namespace ? String(r.namespace) : undefined,
        }),
      );

      // Extract components summary from spec
      const rawComponents = Array.isArray(spec.components)
        ? spec.components
        : [];
      const components = rawComponents.map((c: Record<string, unknown>) => ({
        name: String(c.name ?? ''),
        type: String(c.type ?? ''),
      }));

      return res.json({
        name: String(metadata.name ?? componentName),
        namespace: String(metadata.namespace ?? namespace),
        phase: String(status.phase ?? 'Unknown'),
        message: String(status.message ?? ''),
        conditions,
        resourcesCreated,
        initialBuildTriggered: Boolean(status.initialBuildTriggered),
        // Spec context
        owner: String(spec.owner ?? ''),
        gitRepo: String(spec.gitRepo ?? ''),
        imageRepo: String(spec.imageRepo ?? ''),
        replicas: Number(spec.replicas ?? 0),
        components,
        createdAt: String(metadata.creationTimestamp ?? ''),
      });
    } catch (err: unknown) {
      if (isNotFoundError(err)) {
        return res.status(404).json({
          name: componentName,
          namespace,
          phase: 'Unknown',
          message: 'HeliosApp CR not found in the cluster',
          conditions: [],
          resourcesCreated: [],
          initialBuildTriggered: false,
          owner: '',
          gitRepo: '',
          imageRepo: '',
          replicas: 0,
          components: [],
          createdAt: '',
        });
      }

      logger.error(
        `Failed to fetch HeliosApp status for ${componentName}: ${String(err)}`,
      );
      return res.status(500).json({
        error: err instanceof Error ? err.message : String(err),
      });
    }
  });

  return router;
}
