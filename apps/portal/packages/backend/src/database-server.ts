import express from 'express';
import { CoreV1Api, KubeConfig } from '@kubernetes/client-node';

const app = express();
const port = 3002;

app.get('/api/helios/database/info/:componentName', async (req, res) => {
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
    const decode = (val: string | Buffer | undefined) => {
      if (!val) return undefined;
      const buf = typeof val === 'string' ? Buffer.from(val, 'base64') : val;
      return buf.toString('utf8');
    };

    // Extract values from secret (keys might be DB_HOST, DB_USER, DB_PASS or host, user, password)
    const host = decode(data['DB_HOST'] || data['host']);
    const user = decode(data['DB_USER'] || data['user']);
    const password = decode(data['DB_PASS'] || data['password']);
    const portStr = decode(data['DB_PORT'] || data['port']);
    const port = portStr ? parseInt(portStr, 10) : 5432;
    const database = decode(data['DB_NAME'] || data['database']) || `${componentName}-db`;

    const databaseInfo = {
      host,
      port,
      user,
      password,
      database,
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

app.listen(port, () => {
  console.log(`Database info server running on port ${port}`);
});
