import { createBackendModule } from '@backstage/backend-plugin-api';
import { scaffolderActionsExtensionPoint } from '@backstage/plugin-scaffolder-node';
import { createRunKubectlAction } from '../actions/runKubectl'; // Import file ở Bước 1

export const scaffolderCustomActionsModule = createBackendModule({
  pluginId: 'scaffolder', // Module này thuộc về plugin Scaffolder
  moduleId: 'custom-actions',
  register(env) {
    env.registerInit({
      deps: { scaffolder: scaffolderActionsExtensionPoint },
      async init({ scaffolder }) {
        // Đăng ký hành động 'kubernetes:apply' vào hệ thống
        scaffolder.addActions(createRunKubectlAction());
      },
    });
  },
});