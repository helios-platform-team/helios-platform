import {
  coreServices,
  createBackendModule,
} from '@backstage/backend-plugin-api';
import createDatabaseRouter from './database-router';

/**
 * Database router feature for Helios platform
 * Provides API endpoints for fetching database information from Kubernetes
 */
export const databaseFeature = createBackendModule({
  pluginId: 'app',
  moduleId: 'database-feature-legacy',
  register(env) {
    env.registerInit({
      deps: {
        httpRouter: coreServices.httpRouter,
      },
      async init({ httpRouter }) {
        httpRouter.use(createDatabaseRouter() as any);
      },
    });
  },
});
