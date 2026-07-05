import {
  createPlugin,
  createRoutableExtension,
  createApiFactory,
  discoveryApiRef,
  fetchApiRef,
} from '@backstage/core-plugin-api';

import { rootRouteRef } from './routes';
import { k8sSecretManagerApiRef, K8sSecretManagerApiClient } from './api';

export const k8SSecretManagerPlugin = createPlugin({
  id: 'k8s-secret-manager',
  routes: {
    root: rootRouteRef,
  },
  apis: [
    createApiFactory({
      api: k8sSecretManagerApiRef,
      deps: { discoveryApi: discoveryApiRef, fetchApi: fetchApiRef },
      factory: ({ discoveryApi, fetchApi }) =>
        new K8sSecretManagerApiClient({ discoveryApi, fetchApi }),
    }),
  ],
});

export const K8SSecretManagerPage = k8SSecretManagerPlugin.provide(
  createRoutableExtension({
    name: 'K8SSecretManagerPage',
    component: () =>
      import('./components/SecretsManagementPage').then(
        m => m.SecretsManagementPage,
      ),
    mountPoint: rootRouteRef,
  }),
);
