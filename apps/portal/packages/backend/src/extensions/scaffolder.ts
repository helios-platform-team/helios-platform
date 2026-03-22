import { createBackendModule } from '@backstage/backend-plugin-api';
import { scaffolderActionsExtensionPoint } from '@backstage/plugin-scaffolder-node';
import { createKubernetesApplyAction } from '../actions/kubernetes-apply';
import { createGithubCredentialsSecretAction } from '../actions/create-github-secret';

export const scaffolderModuleCustomActions = createBackendModule({
  pluginId: 'scaffolder',
  moduleId: 'custom-actions',
  register(env) {
    env.registerInit({
      deps: {
        scaffolder: scaffolderActionsExtensionPoint,
      },
      async init({ scaffolder }) {
        scaffolder.addActions(createKubernetesApplyAction() as any);
        scaffolder.addActions(createGithubCredentialsSecretAction() as any);
      },
    });
  },
});
