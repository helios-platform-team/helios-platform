import { createDevApp } from '@backstage/dev-utils';
import { k8SSecretManagerPlugin, K8SSecretManagerPage } from '../src/plugin';

createDevApp()
  .registerPlugin(k8SSecretManagerPlugin)
  .addPage({
    element: <K8SSecretManagerPage />,
    title: 'Root Page',
    path: '/k8s-secret-manager',
  })
  .render();
