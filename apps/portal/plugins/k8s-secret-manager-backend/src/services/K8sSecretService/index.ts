import {
  coreServices,
  createServiceFactory,
  createServiceRef,
} from '@backstage/backend-plugin-api';
import { K8sSecretServiceImpl } from './K8sSecretService';
import { K8sSecretService } from './types';

export * from './types';

export const k8sSecretServiceRef = createServiceRef<K8sSecretService>({
  id: 'k8s.secret.service',
  defaultFactory: async service =>
    createServiceFactory({
      service,
      deps: { logger: coreServices.logger },
      async factory({ logger }) {
        return new K8sSecretServiceImpl(logger);
      },
    }),
});
