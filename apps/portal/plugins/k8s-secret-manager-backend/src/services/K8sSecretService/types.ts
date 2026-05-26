export const HELIOS_CRD = {
  group: 'app.helios.io',
  version: 'v1alpha1',
  plural: 'heliosapps',
};

export interface SecretDto {
  name: string;
  namespace: string;
  createdAt?: string;
}

export interface PaginatedSecretResponse {
  items: SecretDto[];
  nextPageToken?: string;
}

export interface K8sSecretService {
  listSecrets(
    serviceName: string,
    namespace: string,
    limit: number,
    continueToken?: string,
  ): Promise<PaginatedSecretResponse>;
  createSecret(input: any, options: any): Promise<SecretDto>;
  deleteSecret(
    serviceName: string,
    name: string,
    namespace: string,
  ): Promise<void>;
  getSecretEntries(
    serviceName: string,
    secretName: string,
    namespace: string,
  ): Promise<Record<string, string>>;
  upsertSecretEntry(input: {
    serviceName: string;
    namespace: string;
    secretName: string;
    key: string;
    value: string;
  }): Promise<void>;
  deleteSecretEntry(input: {
    serviceName: string;
    namespace: string;
    secretName: string;
    key: string;
  }): Promise<void>;
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
