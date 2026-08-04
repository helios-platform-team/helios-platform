import {
  ScmIntegrationsApi,
  scmIntegrationsApiRef,
  ScmAuth,
} from '@backstage/integration-react';
import {
  AnyApiFactory,
  configApiRef,
  createApiFactory,
  createApiRef,
  discoveryApiRef,
  oauthRequestApiRef,
  OAuthApi,
  OpenIdConnectApi,
  ProfileInfoApi,
  BackstageIdentityApi,
  SessionApi,
} from '@backstage/core-plugin-api';
import { OAuth2 } from '@backstage/core-app-api';

export const giteaAuthApiRef = createApiRef<
  OAuthApi &
    OpenIdConnectApi &
    ProfileInfoApi &
    BackstageIdentityApi &
    SessionApi
>({
  id: 'internal.auth.gitea',
});
export const apis: AnyApiFactory[] = [
  createApiFactory({
    api: scmIntegrationsApiRef,
    deps: { configApi: configApiRef },
    factory: ({ configApi }) => ScmIntegrationsApi.fromConfig(configApi),
  }),
  createApiFactory({
    api: giteaAuthApiRef,
    deps: {
      discoveryApi: discoveryApiRef,
      oauthRequestApi: oauthRequestApiRef,
      configApi: configApiRef,
    },
    factory: ({ discoveryApi, oauthRequestApi, configApi }) => {
      const api = OAuth2.create({
        configApi,
        discoveryApi,
        oauthRequestApi,
        provider: {
          id: 'oauth2',
          title: 'Gitea',
          icon: () => null,
        },
        environment: configApi.getOptionalString('auth.environment'),
      });

      const originalSignOut = api.signOut.bind(api);

      const giteaIntegrations =
        configApi.getOptionalConfigArray('integrations.gitea');
      const giteaBaseUrl =
        giteaIntegrations && giteaIntegrations.length > 0
          ? (giteaIntegrations[0].getOptionalString('baseUrl') ??
            'http://localhost:3030')
          : 'http://localhost:3030';

      const appBaseUrl =
        configApi.getOptionalString('app.baseUrl') ?? window.location.origin;

      api.signOut = async () => {
        try {
          await originalSignOut();
        } finally {
          localStorage.removeItem('@backstage/core-app-api:auth');
          sessionStorage.removeItem('@backstage/core-app-api:auth');
          window.location.assign(
            `${giteaBaseUrl}/user/logout?redirect_to=${encodeURIComponent(appBaseUrl)}`,
          );
        }
      };

      return api;
    },
  }),
  ScmAuth.createDefaultApiFactory(),
];
