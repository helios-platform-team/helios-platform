/*
 * Hi!
 *
 * Note that this is an EXAMPLE Backstage backend. Please check the README.
 *
 * Happy hacking!
 */

import * as dotenv from 'dotenv';
import { createBackend } from '@backstage/backend-defaults';
import { scaffolderModuleCustomActions } from './extensions/scaffolder';
import { createBackendModule } from '@backstage/backend-plugin-api';
import { authProvidersExtensionPoint, createOAuthProviderFactory } from '@backstage/plugin-auth-node';
import { oauth2Authenticator } from '@backstage/plugin-auth-backend-module-oauth2-provider';
import { AuthorizeResult, isPermission } from '@backstage/plugin-permission-common';
import {
  PermissionPolicy,
  PolicyQuery,
  PolicyQueryUser,
} from '@backstage/plugin-permission-node';
import { policyExtensionPoint } from '@backstage/plugin-permission-node/alpha';
import { catalogEntityReadPermission } from '@backstage/plugin-catalog-common/alpha';
import {
  catalogConditions,
  createCatalogConditionalDecision,
} from '@backstage/plugin-catalog-backend/alpha';

const customAuth = createBackendModule({
  pluginId: 'auth',
  moduleId: 'custom-auth-provider',
  register(reg) {
    reg.registerInit({
      deps: { providers: authProvidersExtensionPoint },
      init: async ({ providers }) => {
        providers.registerProvider({
          providerId: 'oauth2',
          factory: createOAuthProviderFactory({
            authenticator: oauth2Authenticator,
            async profileTransform(result) {
              const res = result as any;
              const accessToken = res.session?.accessToken || res.accessToken;
              if (accessToken) {
                const giteaUrl = process.env.GITEA_URL || process.env.GITEA_INTERNAL_URL || 'http://localhost:3030';
                try {
                  const response = await fetch(`${giteaUrl}/api/v1/user`, {
                    headers: { Authorization: `Bearer ${accessToken}` }
                  });
                  if (response.ok) {
                    const data = await response.json();
                    return {
                      profile: {
                        email: data.email || `${data.login}@helios.local`,
                        displayName: data.full_name || data.login,
                        picture: data.avatar_url,
                      }
                    };
                  }
                } catch (e) {
                  console.error('Failed to fetch Gitea profile in profileTransform', e);
                }
              }
              return { profile: { displayName: 'User' } };
            },
            async signInResolver(info, ctx) {
              const { result } = info as any;
              const accessToken = result.session?.accessToken || result.accessToken;

              if (accessToken) {
                const giteaUrl = process.env.GITEA_URL || process.env.GITEA_INTERNAL_URL || 'http://localhost:3030';
                
                try {
                  const response = await fetch(`${giteaUrl}/api/v1/user`, {
                    headers: { Authorization: `Bearer ${accessToken}` }
                  });
                  
                  if (response.ok) {
                    const data = await response.json();
                    const username = data.login;
                    if (username) {
                      const userEntityRef = `user:default/${username}`;
                      const userGroupRef = `group:default/${username}`;
                      return ctx.issueToken({
                        claims: {
                          sub: userEntityRef,
                          ent: [userEntityRef, userGroupRef, 'user:default/guest'],
                        },
                      });
                    }
                  }
                } catch (e) {
                  console.error(`Failed to fetch Gitea profile:`, e);
                }
              }
              
              throw new Error('Could not fetch user profile from Gitea or profile was empty');
            },
          }),
        });
      },
    });
  },
});

class CatalogOwnershipPermissionPolicy implements PermissionPolicy {
  async handle(request: PolicyQuery, user?: PolicyQueryUser) {
    if (isPermission(request.permission, catalogEntityReadPermission)) {
      const claims = user?.info.ownershipEntityRefs;
      
      console.log('Permission policy check for:', request.permission.name);
      console.log('User claims:', claims);
      
      // Allow internal backend-to-backend calls (which have no user token because backend.auth.keys is disabled)
      if (!user) {
        return { result: AuthorizeResult.ALLOW };
      }

      if (!claims || claims.length === 0) {
        return { result: AuthorizeResult.DENY };
      }

      return createCatalogConditionalDecision(
        request.permission,
        {
          anyOf: [
            catalogConditions.isEntityKind({
              kinds: ['Template', 'Location', 'User', 'Group', 'System', 'Domain', 'API', 'Resource'],
            }),
            catalogConditions.isEntityOwner({
              claims,
            }),
          ],
        },
      );
    }

    return { result: AuthorizeResult.ALLOW };
  }
}

const permissionModuleCatalogOwnerPolicy = createBackendModule({
  pluginId: 'permission',
  moduleId: 'catalog-owner-policy',
  register(reg) {
    reg.registerInit({
      deps: { policy: policyExtensionPoint },
      async init({ policy }) {
        policy.setPolicy(new CatalogOwnershipPermissionPolicy());
      },
    });
  },
});

// Load env vars from root .env
dotenv.config({ debug: true });

const backend = createBackend();

backend.add(import('@backstage/plugin-app-backend'));
backend.add(import('@backstage/plugin-proxy-backend'));

// scaffolder plugin
backend.add(import('@backstage/plugin-scaffolder-backend'));
backend.add(import('@backstage/plugin-scaffolder-backend-module-gitea'));
backend.add(scaffolderModuleCustomActions);

backend.add(
  import('@backstage/plugin-scaffolder-backend-module-notifications'),
);

// auth plugin
backend.add(import('@backstage/plugin-auth-backend'));
backend.add(customAuth);
// Optional: uncomment to enable GitHub OAuth login (also uncomment in app-config.yaml)
// backend.add(customAuth);

// catalog plugin
backend.add(import('@backstage/plugin-catalog-backend'));
backend.add(
  import('@backstage/plugin-catalog-backend-module-scaffolder-entity-model'),
);

// Gitea catalog discovery
backend.add(import('@backstage/plugin-catalog-backend-module-gitea'));

// See https://backstage.io/docs/features/software-catalog/configuration#subscribing-to-catalog-errors
backend.add(import('@backstage/plugin-catalog-backend-module-logs'));

// permission plugin
backend.add(import('@backstage/plugin-permission-backend'));
backend.add(permissionModuleCatalogOwnerPolicy);

// search plugin
backend.add(import('@backstage/plugin-search-backend'));

// search engine
// See https://backstage.io/docs/features/search/search-engines
backend.add(import('@backstage/plugin-search-backend-module-pg'));

// search collators
backend.add(import('@backstage/plugin-search-backend-module-catalog'));
backend.add(import('@backstage/plugin-search-backend-module-techdocs'));

// kubernetes plugin
backend.add(import('@backstage/plugin-kubernetes-backend'));

// helios plugin
backend.add(import('@helios/plugin-database-backend'));

// notifications and signals plugins
backend.add(import('@backstage/plugin-events-backend'));
backend.add(import('@backstage/plugin-notifications-backend'));
backend.add(import('@backstage/plugin-signals-backend'));

// database info extension
backend.add(import('./extensions/database-router-simple'));

// secrets management (env variables)
backend.add(import('@internal/backstage-plugin-k8s-secret-manager-backend'));

backend.start();
