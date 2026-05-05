import {
  createApiRef,
  DiscoveryApi,
  FetchApi,
} from '@backstage/core-plugin-api';

// 1. Define the types based on your backend responses
export interface Secret {
  name: string;
  namespace: string;
  createdAt?: string;
}

export interface CreateSecretInput {
  serviceName: string;
  secretName: string;
  namespace: string;
  secretData: Record<string, string>;
  entityRef?: string;
}

// 2. Define the API Interface
export interface K8sSecretManagerApi {
  listSecrets(namespace: string, serviceName: string): Promise<Secret[]>;
  createSecret(input: CreateSecretInput): Promise<Secret>;
  deleteSecret(
    namespace: string,
    serviceName: string,
    name: string,
  ): Promise<void>;
}

// 3. Create the ApiRef for dependency injection
export const k8sSecretManagerApiRef = createApiRef<K8sSecretManagerApi>({
  id: 'plugin.k8s-secret-manager.service',
});

// 4. Implement the API Client
export class K8sSecretManagerApiClient implements K8sSecretManagerApi {
  private readonly discoveryApi: DiscoveryApi;
  private readonly fetchApi: FetchApi;

  constructor(options: { discoveryApi: DiscoveryApi; fetchApi: FetchApi }) {
    this.discoveryApi = options.discoveryApi;
    this.fetchApi = options.fetchApi;
  }

  private async getBaseUrl(): Promise<string> {
    // This dynamically finds the backend URL (e.g., http://localhost:7007/api/k8s-secret-manager)
    return await this.discoveryApi.getBaseUrl('k8s-secret-manager');
  }

  async listSecrets(namespace: string, serviceName: string): Promise<Secret[]> {
    const baseUrl = await this.getBaseUrl();
    const query = new URLSearchParams({
      namespace,
      serviceName,
    });
    const response = await this.fetchApi.fetch(
      `${baseUrl}/secrets?${query.toString()}`,
    );

    if (!response.ok) throw new Error(await response.text());
    return await response.json();
  }

  async createSecret(input: CreateSecretInput): Promise<Secret> {
    const baseUrl = await this.getBaseUrl();
    const response = await this.fetchApi.fetch(`${baseUrl}/secrets`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    });

    if (!response.ok) throw new Error(await response.text());
    return await response.json();
  }

  async deleteSecret(
    namespace: string,
    serviceName: string,
    name: string,
  ): Promise<void> {
    const baseUrl = await this.getBaseUrl();
    const encodedNamespace = encodeURIComponent(namespace);
    const encodedServiceName = encodeURIComponent(serviceName);
    const encodedName = encodeURIComponent(name);
    const response = await this.fetchApi.fetch(
      `${baseUrl}/secrets/${encodedNamespace}/${encodedServiceName}/${encodedName}`,
      {
        method: 'DELETE',
      },
    );

    if (!response.ok) throw new Error(await response.text());
  }
}
