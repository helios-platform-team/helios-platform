import {
  BackstageCredentials,
  BackstageUserPrincipal,
} from '@backstage/backend-plugin-api';

export const HELIOS_CRD = {
  group: 'app.helios.io',
  version: 'v1alpha1',
  plural: 'heliosapps',
};

export interface SecretResponse {
  name: string;
  namespace: string;
  createdAt?: string;
}

export interface K8sSecretService {
  listSecrets(
    serviceName: string,
    namespace: string,
  ): Promise<SecretResponse[]>;
  createSecret(
    input: {
      serviceName: string;
      namespace: string;
      secretName: string;
      secretData: Record<string, string>;
      entityRef?: string;
    },
    options: { credentials: BackstageCredentials<BackstageUserPrincipal> },
  ): Promise<SecretResponse>;
  deleteSecret(
    serviceName: string,
    name: string,
    namespace: string,
  ): Promise<void>;
}

export interface HeliosTrait {
  type: string;
  properties?: Record<string, unknown>;
}

export interface HeliosComponent {
  name: string;
  traits?: HeliosTrait[];
}

export interface HeliosAppSpec {
  gitopsRepo?: string;
  gitopsPath?: string;
  gitopsSecretRef?: string;
  components?: HeliosComponent[];
}

export interface HeliosAppResource {
  metadata?: {
    resourceVersion?: string;
  };
  spec?: HeliosAppSpec;
}
