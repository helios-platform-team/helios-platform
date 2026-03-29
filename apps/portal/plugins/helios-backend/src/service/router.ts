import { Router } from 'express';
import expressPromiseRouter from 'express-promise-router';
import * as k8s from '@kubernetes/client-node';
import { Config } from '@backstage/config';
import { LoggerService } from '@backstage/backend-plugin-api';

export interface RouterOptions {
  logger: LoggerService;
  config: Config;
}

/** Matches Helm operator: GetDatabaseSecretName / GetDatabaseHost (traits.go). */
const databaseSecretName = (componentName: string) => `${componentName}-db-secret`;
const databasePodLabel = (componentName: string) => `app=${componentName}-db`;

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
function resolveDbNameFromHeliosApp(body: unknown, componentName: string): string {
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

export async function createRouter(
  options: RouterOptions,
): Promise<Router> {
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
    const secretName = databaseSecretName(componentName);

    let dbName = `${componentName}-db`;
    try {
      const heliosApp: unknown = await customApi.getNamespacedCustomObject({
        group: 'app.helios.io',
        version: 'v1alpha1',
        namespace,
        plural: 'heliosapps',
        name: componentName,
      });
      const heliosAppBody =
        (heliosApp as { body?: unknown }).body ?? heliosApp;
      dbName = resolveDbNameFromHeliosApp(heliosAppBody, componentName);
    } catch {
      // HeliosApp CR may be absent; keep default db name.
    }

    let secret: k8s.V1Secret | undefined;
    try {
      const read = await k8sApi.readNamespacedSecret({
        name: secretName,
        namespace,
      });
      secret = (read as { body?: k8s.V1Secret }).body ?? (read as k8s.V1Secret);
    } catch (err: unknown) {
      const code =
        (err as { statusCode?: number; response?: { statusCode?: number } })
          ?.statusCode ??
        (err as { response?: { statusCode?: number } })?.response?.statusCode;
      if (code === 404) {
        return res.status(404).json({
          error: `Secret ${secretName} not found in namespace ${namespace}`,
        });
      }
      logger.error(
        `Failed to read database secret ${secretName}: ${String(err)}`,
      );
      return res.status(500).json({
        error: err instanceof Error ? err.message : String(err),
      });
    }

    const data = secret?.data ?? {};
    const decode = (val?: string) =>
      val ? Buffer.from(val, 'base64').toString() : '';

    let status: 'Running' | 'Failed' | 'Unknown' = 'Unknown';
    try {
      const pods: unknown = await k8sApi.listNamespacedPod({
        namespace,
        labelSelector: databasePodLabel(componentName),
      });
      const podList = (pods as { body?: k8s.V1PodList }).body ?? pods;
      const items = (podList as k8s.V1PodList)?.items;
      if (items && items.length > 0) {
        const phase = items[0].status?.phase;
        if (phase === 'Running') status = 'Running';
        else if (phase === 'Failed') status = 'Failed';
      }
    } catch {
      // Pod listing is best-effort.
    }

    const portStr = decode(data.DB_PORT);
    const port = portStr ? parseInt(portStr, 10) : 5432;

    return res.json({
      host: decode(data.DB_HOST),
      port,
      user: decode(data.DB_USER),
      password: decode(data.DB_PASS),
      database: decode(data.DB_NAME) || dbName,
      status,
    });
  });

  return router;
}
