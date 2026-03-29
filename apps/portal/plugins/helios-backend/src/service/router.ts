import { Router } from 'express';
import expressPromiseRouter from 'express-promise-router';
import * as k8s from '@kubernetes/client-node';
import { Config } from '@backstage/config';
import { LoggerService } from '@backstage/backend-plugin-api';

export interface RouterOptions {
  logger: LoggerService;
  config: Config;
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

    try {
      // 1. Fetch HeliosApp to get DB name
      const heliosApp: any = await customApi.getNamespacedCustomObject({
        group: 'app.helios.io',
        version: 'v1alpha1',
        namespace,
        plural: 'heliosapps',
        name: componentName,
      });
      const heliosAppBody = heliosApp.body || heliosApp;

      let dbName = componentName.replace(/-/g, '_');
      const dbComponent = heliosAppBody.spec?.components?.find((c: any) => 
        c.traits?.some((t: any) => t.type === 'database') || c.type === 'database'
      );
      
      if (dbComponent) {
          // If it's the new template structure
          dbName = dbComponent.properties?.dbName || dbName;
      }

      // 2. Fetch Secret
      const secrets: any = await k8sApi.listNamespacedSecret({
        namespace,
        labelSelector: `helios.io/secret-type=database-credentials,app=${componentName}-backend`
      });

      const secretList = secrets.body || secrets;
      if (!secretList.items || secretList.items.length === 0) {
        return res.status(404).json({ error: 'Database secret not found' });
      }

      const secret = secretList.items[0];
      const data = secret.data || {};

      const decode = (val?: string) => val ? Buffer.from(val, 'base64').toString() : '';

      // 3. Fetch Pod Status
      const pods: any = await k8sApi.listNamespacedPod({
        namespace,
        labelSelector: `app=${componentName}-backend-db`
      });

      const podList = pods.body || pods;
      let status: 'Running' | 'Failed' | 'Unknown' = 'Unknown';
      if (podList.items && podList.items.length > 0) {
        const podStatus = podList.items[0].status?.phase;
        if (podStatus === 'Running') status = 'Running';
        else if (podStatus === 'Failed') status = 'Failed';
      }

      return res.json({
        host: decode(data.DB_HOST),
        port: parseInt(decode(data.DB_PORT) || '5432', 10),
        user: decode(data.DB_USER),
        password: decode(data.DB_PASS),
        database: dbName,
        status,
      });

    } catch (error: any) {
      logger.error(`Failed to fetch database info for ${componentName}: ${error.message}`);
      return res.status(500).json({ error: error.message });
    }
  });

  return router;
}
