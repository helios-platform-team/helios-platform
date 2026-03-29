import {
  coreServices,
  createBackendModule,
} from '@backstage/backend-plugin-api';
import createDatabaseRouter from './database-router';

export default createBackendModule({
  pluginId: 'http',
  moduleId: 'helios-database',
  register(env) {
    env.registerInit({
      deps: {
        http: coreServices.httpRouter,
      },
      async init({ http }) {
        http.use(createDatabaseRouter());
      },
    });
  },
});
