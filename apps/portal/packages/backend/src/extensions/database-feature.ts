import {
  coreServices,
  createBackendModule,
} from '@backstage/backend-plugin-api';
import { Router } from 'express';

export default createBackendModule({
  pluginId: 'app',
  moduleId: 'database',
  register(reg) {
    reg.registerInit({
      deps: {
        rootHttpRouter: coreServices.rootHttpRouter,
      },
      async init({ rootHttpRouter }) {
        const router = Router();

        router.get('/info/:componentName', (req, res) => {
          const { componentName } = req.params;
          res.status(501).json({
            error: `Deprecated database feature endpoint for component=${componentName}`,
          });
        });

        rootHttpRouter.use('/api/helios/database', router as any);
      },
    });
  },
});
