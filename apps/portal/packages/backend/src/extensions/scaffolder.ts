import { createBackendModule } from '@backstage/backend-plugin-api';
import { scaffolderActionsExtensionPoint } from '@backstage/plugin-scaffolder-node';
import { coreServices } from '@backstage/backend-plugin-api';
import { createKubernetesApplyAction } from '../actions/kubernetes-apply';
import { createGitCredentialsSecretAction } from '../actions/create-git-credentials-secret';
import { createGiteaWebhookAction } from '../actions/create-gitea-webhook';

export const scaffolderModuleCustomActions = createBackendModule({
  pluginId: 'scaffolder',
  moduleId: 'custom-actions',
  register(env) {
    env.registerInit({
      deps: {
        scaffolder: scaffolderActionsExtensionPoint,
        config: coreServices.rootConfig,
      },
      async init({ scaffolder, config }) {
        scaffolder.addActions(createKubernetesApplyAction() as any);
        scaffolder.addActions(createGitCredentialsSecretAction() as any);
        scaffolder.addActions(createGiteaWebhookAction({ config }) as any);
      },
    });
  },
});
