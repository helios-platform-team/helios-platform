import { Router } from 'express';
import { KubeConfig, CoreV1Api } from '@kubernetes/client-node';

function createDatabaseRouter(): Router {
  const router = Router();

  router.get('/api/helios/database/info/:componentName', async (req, res) => {
    try {
      const { componentName } = req.params;
      
      // Initialize Kubernetes client
      const kc = new KubeConfig();
      kc.loadFromDefault();
      const k8sApi = kc.makeApiClient(CoreV1Api);

      // Get the secret
      const secretName = `${componentName}-backend-db-secret`;
      const secret = await k8sApi.readNamespacedSecret(secretName, 'default');

      if (!secret.body?.data) {
        return res.status(404).json({ error: `Secret ${secretName} not found or has no data` });
      }

      const data = secret.body.data;
      
      // Decode base64 data
      const decode = (val: string | Buffer) => {
        const buf = typeof val === 'string' ? Buffer.from(val, 'base64') : val;
        return buf.toString('utf8');
      };

      const databaseInfo = {
        host: data.host ? decode(data.host) : undefined,
        port: data.port ? parseInt(decode(data.port), 10) : undefined,
        user: data.user ? decode(data.user) : undefined,
        password: data.password ? decode(data.password) : undefined,
        database: data.database ? decode(data.database) : undefined,
      };

      // Try to get pod status
      let status = 'Unknown';
      try {
        const podLabel = `app=${componentName}-backend-db`;
        const pods = await k8sApi.listNamespacedPod('default', undefined, undefined, undefined, undefined, podLabel);
        
        if (pods.body?.items && pods.body.items.length > 0) {
          const pod = pods.body.items[0];
          const phase = pod.status?.phase;
          status = phase === 'Running' ? 'Running' : phase || 'Unknown';
        }
      } catch (podError) {
        // Pod might not exist, that's okay
        status = 'Unknown';
      }

      res.json({
        ...databaseInfo,
        status,
      });
    } catch (error: any) {
      console.error('Error fetching database info:', error);
      res.status(500).json({
        error: error?.message || 'Failed to fetch database information',
      });
    }
  });

  router.get('/api/helios/database/health', (req, res) => {
    res.json({ status: 'ok', service: 'database-api' });
  });

  return router;
}

export default createDatabaseRouter;
