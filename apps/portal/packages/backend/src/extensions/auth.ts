import { createBackendModule } from '@backstage/backend-plugin-api';
import { githubAuthenticator } from '@backstage/plugin-auth-backend-module-github-provider';
import {
  authProvidersExtensionPoint,
  createOAuthProviderFactory,
} from '@backstage/plugin-auth-node';
import { stringifyEntityRef } from '@backstage/catalog-model';

export const customAuth = createBackendModule({
  pluginId: 'auth',
  moduleId: 'custom-auth-provider',
  register(reg) {
    reg.registerInit({
      deps: { providers: authProvidersExtensionPoint },
      async init({ providers }) {
        providers.registerProvider({
          providerId: 'github',
          factory: createOAuthProviderFactory({
            authenticator: githubAuthenticator,
            async signInResolver(info, ctx) {
              console.log('GitHub Auth Info:', JSON.stringify(info, null, 2));
              // GitHub returned profile might map username to loging in fullProfile
              const username = info.result.fullProfile.username;

              if (!username) {
                throw new Error(`GitHub user profile contained no username. Full Profile: ${JSON.stringify(info.result.fullProfile)}`);
              }

              // Map the GitHub username to a Backstage User entity
              const userEntity = stringifyEntityRef({
                kind: 'User',
                name: username,
                namespace: 'default',
              });

              try {
                return await ctx.signInWithCatalogUser({
                  entityRef: userEntity,
                });
              } catch (error) {
                console.log(`User ${username} not found in catalog, falling back to guest user.`);
                return ctx.signInWithCatalogUser({
                  entityRef: { name: 'guest', kind: 'User', namespace: 'default' },
                });
              }
            },
          }),
        });
      },
    });
  },
});
