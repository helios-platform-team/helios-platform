import {
  createBackendPlugin,
  coreServices,
} from '@backstage/backend-plugin-api';
import { k8sSecretServiceRef } from './services/K8sSecretService';
import { createRouter } from './router';

export const k8SSecretManagerBackendPlugin = createBackendPlugin({
  pluginId: 'k8s-secret-manager',
  register(env) {
    env.registerInit({
      deps: {
        httpRouter: coreServices.httpRouter,
        httpAuth: coreServices.httpAuth,
        secretService: k8sSecretServiceRef,
      },
      async init({ httpRouter, httpAuth, secretService }) {
        httpRouter.use(
          await createRouter({
            httpAuth,
            secretService,
          }),
        );
      },
    });
  },
});
