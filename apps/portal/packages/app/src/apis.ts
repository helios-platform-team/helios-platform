import {
  ScmIntegrationsApi,
  scmIntegrationsApiRef,
  ScmAuth,
} from '@backstage/integration-react';
import {
  AnyApiFactory,
  configApiRef,
  createApiFactory,
  alertApiRef,
} from '@backstage/core-plugin-api';
import { toastApiRef } from '@backstage/frontend-plugin-api';

export const apis: AnyApiFactory[] = [
  createApiFactory({
    api: scmIntegrationsApiRef,
    deps: { configApi: configApiRef },
    factory: ({ configApi }) => ScmIntegrationsApi.fromConfig(configApi),
  }),
  createApiFactory({
    api: toastApiRef,
    deps: { alertApi: alertApiRef },
    factory: ({ alertApi }) => ({
      post: toast => {
        alertApi.post({
          message: String(toast.title),
          severity: ((): 'error' | 'warning' | 'info' | 'success' => {
            switch (toast.status) {
              case 'danger':
                return 'error';
              case 'success':
                return 'success';
              case 'warning':
                return 'warning';
              case 'info':
                return 'info';
              default:
                return 'info';
            }
          })(),
          display: toast.timeout ? 'transient' : 'permanent',
        });
        return { close: () => {} };
      },
    }),
  }),
  ScmAuth.createDefaultApiFactory(),
];
