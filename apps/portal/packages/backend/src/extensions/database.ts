import { createBackendFeature } from '@backstage/backend-plugin-api';
import { createDatabaseRouter } from '@helios/plugin-database-backend';

/**
 * Database router feature for Helios platform
 * Provides API endpoints for fetching database information from Kubernetes
 */
export const databaseFeature = createBackendFeature({
  pluginId: 'database',
  register(reg) {
    reg.registerInit({
      deps: {
        logger: 'logger' as unknown as any,
      },
      init: async () => {
        const databaseRouter = createDatabaseRouter();
        return {
          router: databaseRouter,
        };
      },
    });
  },
});

// Export router for direct use
export { createDatabaseRouter };
