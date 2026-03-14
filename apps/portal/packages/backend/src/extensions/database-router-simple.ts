import {
  coreServices,
  createBackendModule,
} from '@backstage/backend-plugin-api';
import { Router } from 'express';
import { execSync } from 'child_process';

export default createBackendModule({
  pluginId: 'app',
  moduleId: 'database-info',
  register(reg) {
    reg.registerInit({
      deps: {
        middleware: coreServices.middleware,
      },
      async init({ middleware }) {
        const router = Router();

        router.get('/api/helios/database/info/:componentName', async (req, res) => {
          try {
            const { componentName } = req.params;
            const secretName = `${componentName}-backend-db-secret`;

            // Use kubectl to fetch secret
            const secretCmd = `kubectl get secret ${secretName} -n default -o json`;
            const secretJson = JSON.parse(execSync(secretCmd, { encoding: 'utf-8' }));

            if (!secretJson.data) {
              return res.status(404).json({ error: `Secret ${secretName} not found or has no data` });
            }

            const data = secretJson.data;
            
            // Decode base64
            const decode = (val: string | undefined) => {
              if (!val) return undefined;
              return Buffer.from(val, 'base64').toString('utf-8');
            };

            // Extract values (support both DB_* and lowercase keys)
            const host = decode(data['DB_HOST'] || data['host']);
            const user = decode(data['DB_USER'] || data['user']);
            const password = decode(data['DB_PASS'] || data['password']);
            const portStr = decode(data['DB_PORT'] || data['port']);
            const port = portStr ? parseInt(portStr, 10) : 5432;
            const database = decode(data['DB_NAME'] || data['database']) || `${componentName}-db`;

            // Get pod status
            let status = 'Unknown';
            try {
              const podLabel = `app=${componentName}-backend-db`;
              const podCmd = `kubectl get pods -l ${podLabel} -n default -o json`;
              const podsJson = JSON.parse(execSync(podCmd, { encoding: 'utf-8' }));
              
              if (podsJson.items && podsJson.items.length > 0) {
                const phase = podsJson.items[0].status?.phase;
                status = phase === 'Running' ? 'Running' : phase || 'Unknown';
              }
            } catch (podError) {
              status = 'Unknown';
            }

            res.json({
              host,
              port,
              user,
              password,
              database,
              status,
            });
          } catch (error: any) {
            console.error('Error fetching database info:', error);
            res.status(500).json({
              error: error?.message || 'Failed to fetch database information',
            });
          }
        });

        middleware.use(router);
      },
    });
  },
});
