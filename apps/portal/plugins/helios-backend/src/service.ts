import { KubeConfig, CoreV1Api } from '@kubernetes/client-node';

export interface DatabaseInfo {
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
  status: 'Running' | 'Failed' | 'Unknown';
}

export class DatabaseService {
  private kubeConfig: KubeConfig;
  private coreApi: CoreV1Api;

  constructor() {
    try {
      this.kubeConfig = new KubeConfig();
      this.kubeConfig.loadFromDefault();
      this.coreApi = this.kubeConfig.makeApiClient(CoreV1Api);
    } catch (error) {
      throw new Error(
        `Failed to initialize Kubernetes client: ${error}. Make sure you are running inside a Kubernetes cluster.`,
      );
    }
  }

  /**
   * Fetch database information from Kubernetes secrets
   */
  async getDatabaseInfo(componentName: string): Promise<DatabaseInfo> {
    try {
      // Get the secret
      const secretName = `${componentName}-backend-db-secret`;
      const namespace = 'default'; // This can be made configurable

      const secretResponse = await this.coreApi.readNamespacedSecret({
        name: secretName,
        namespace,
      } as any);
      const secret = secretResponse;

      if (!secret || !secret.data) {
        throw new Error(`Secret ${secretName} not found`);
      }

      // Decode base64 data
      const data = secret.data;
      const decodeBase64 = (str: string): string => {
        return Buffer.from(str, 'base64').toString('utf-8');
      };

      // Get pod status
      const status = await this.getPodStatus(componentName);

      return {
        host: decodeBase64(data.host || ''),
        port: parseInt(decodeBase64(data.port || '5432'), 10),
        user: decodeBase64(data.user || ''),
        password: decodeBase64(data.password || ''),
        database: decodeBase64(data.database || ''),
        status,
      };
    } catch (error) {
      throw new Error(`Failed to fetch database info: ${error}`);
    }
  }

  /**
   * Get database pod status
   */
  private async getPodStatus(
    componentName: string,
  ): Promise<'Running' | 'Failed' | 'Unknown'> {
    try {
      const labelSelector = `app=${componentName}-backend-db`;
      const namespace = 'default';

      const podsResponse = await this.coreApi.listNamespacedPod({
        namespace,
        labelSelector,
      } as any);
      const pods = podsResponse;

      if (!pods || !pods.items || pods.items.length === 0) {
        return 'Unknown';
      }

      const pod = pods.items[0];
      const phase = pod.status?.phase;

      if (phase === 'Running') {
        return 'Running';
      } else if (phase === 'Failed') {
        return 'Failed';
      }
      return 'Unknown';
    } catch {
      return 'Unknown';
    }
  }
}
