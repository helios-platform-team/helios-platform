import {
  BackstageCredentials,
  BackstageUserPrincipal,
  LoggerService,
} from '@backstage/backend-plugin-api';
import * as k8s from '@kubernetes/client-node';
import { K8sSecretService, PaginatedSecretResponse, SecretDto } from './types';
import { GitOpsService } from '../GitOpsService';
import { HeliosAppManager } from '../HeliosAppManager';

export class K8sSecretServiceImpl implements K8sSecretService {
  readonly #logger: LoggerService;
  readonly #k8sCoreApi: k8s.CoreV1Api;
  readonly #gitOpsService: GitOpsService;
  readonly #heliosAppManager: HeliosAppManager;

  constructor(logger: LoggerService) {
    this.#logger = logger;

    const kc = new k8s.KubeConfig();
    kc.loadFromDefault();
    this.#k8sCoreApi = kc.makeApiClient(k8s.CoreV1Api);
    const k8sCustomApi = kc.makeApiClient(k8s.CustomObjectsApi);

    // Initialize Domain Services
    this.#gitOpsService = new GitOpsService(logger, this.#k8sCoreApi);
    this.#heliosAppManager = new HeliosAppManager(logger, k8sCustomApi);
  }

  async listSecrets(
    serviceName: string,
    namespace: string,
    limit: number = 10,
    continueToken?: string,
  ): Promise<PaginatedSecretResponse> {
    const prefix = `${serviceName}-`;

    const startIndex = continueToken ? parseInt(continueToken, 10) : 0;
    if (isNaN(startIndex) || startIndex < 0) {
      throw new Error(`Invalid continueToken: ${continueToken}`);
    }

    let k8sContinue: string | undefined = undefined;
    let matchingFoundSoFar = 0;
    const paginatedItems: SecretDto[] = [];
    let hasMore = false;

    do {
      const response = await this.#k8sCoreApi.listNamespacedSecret({
        namespace,
        limit: 100, // Fetch in chunks to reduce memory footprint
        _continue: k8sContinue,
      });

      const secrets = response.items || [];

      for (const s of secrets) {
        if (s.metadata?.name?.startsWith(prefix)) {
          matchingFoundSoFar++;
          if (matchingFoundSoFar > startIndex) {
            if (paginatedItems.length < limit) {
              paginatedItems.push(this.#mapSecretResponse(s, namespace));
            } else {
              hasMore = true;
              break;
            }
          }
        }
      }

      if (hasMore) {
        break;
      }

      k8sContinue = response.metadata?._continue;
    } while (k8sContinue);

    const nextPageToken = hasMore ? String(startIndex + limit) : undefined;

    return {
      items: paginatedItems,
      nextPageToken,
    };
  }

  async createSecret(
    input: {
      serviceName: string;
      namespace: string;
      secretName: string;
      entityRef?: string;
    },
    options: { credentials: BackstageCredentials<BackstageUserPrincipal> },
  ): Promise<SecretDto> {
    const { serviceName, secretName, namespace, entityRef } = input;
    const fullName = this.#getPrefixedName(serviceName, secretName);
    const user = options.credentials.principal.userEntityRef;

    this.#logger.info(
      `Creating/Updating secret ${fullName} in ${namespace} for ${user}`,
    );

    const manifest: k8s.V1Secret = {
      apiVersion: 'v1',
      kind: 'Secret',
      metadata: {
        name: fullName,
        namespace,
        annotations: entityRef
          ? { 'backstage.io/managed-by-entity': entityRef }
          : {},
      },
    };

    const responseBody = await this.#upsertSecret(
      fullName,
      namespace,
      manifest,
    );

    // Discover Git repo routing configuration from live K8s cluster (Read-only lookup)
    const gitConfig = await this.#heliosAppManager.getGitOpsConfigFromCluster(
      serviceName,
      namespace,
    );

    // Clone file baseline from Git repository
    const { workDir, targetFile, heliosApp } =
      await this.#gitOpsService.checkoutAndFetchHeliosApp(
        namespace,
        serviceName,
        gitConfig.gitopsRepo,
        gitConfig.gitopsPath,
        gitConfig.gitopsSecretRef,
      );

    const currentResource = heliosApp ?? {
      apiVersion: 'app.helios.io/v1alpha1',
      kind: 'HeliosApp',
      spec: { ...gitConfig, components: [{ name: serviceName, traits: [] }] },
    };

    // Update memory mapping layout details
    const updatedHeliosApp = this.#heliosAppManager.applyExternalSecretTrait(
      currentResource,
      serviceName,
      fullName,
    );

    // Commit changes straight out to GitOps repo (Operator takes over app syncing)
    await this.#gitOpsService.commitAndPushSecretChange(
      workDir,
      targetFile,
      serviceName,
      updatedHeliosApp.spec,
      namespace,
    );

    return this.#mapSecretResponse(responseBody, namespace);
  }

  async deleteSecret(
    serviceName: string,
    name: string,
    namespace: string,
  ): Promise<void> {
    const fullName = this.#getPrefixedName(serviceName, name);
    this.#logger.info(`Deleting secret ${fullName} from ${namespace}`);

    await this.#k8sCoreApi.deleteNamespacedSecret({
      name: fullName,
      namespace,
    });

    // Discover Git configurations from live CR lookup
    const gitConfig = await this.#heliosAppManager.getGitOpsConfigFromCluster(
      serviceName,
      namespace,
    );

    // Fetch current application manifest tracking from Git repository
    const { workDir, targetFile, heliosApp } =
      await this.#gitOpsService.checkoutAndFetchHeliosApp(
        namespace,
        serviceName,
        gitConfig.gitopsRepo,
        gitConfig.gitopsPath,
        gitConfig.gitopsSecretRef,
      );

    if (!heliosApp) {
      this.#logger.warn(
        `No manifest file found at path ${targetFile}. Skipping GitOps deletion step.`,
      );
      return;
    }

    // Strip out specific trait references in memory
    const updatedHeliosApp = this.#heliosAppManager.removeExternalSecretTrait(
      heliosApp,
      serviceName,
      fullName,
    );

    // Push clean file blueprint downstream
    await this.#gitOpsService.commitAndPushSecretChange(
      workDir,
      targetFile,
      serviceName,
      updatedHeliosApp.spec,
      namespace,
    );
  }

  async getSecretEntries(
    serviceName: string,
    secretName: string,
    namespace: string,
  ): Promise<Record<string, string>> {
    const fullName = this.#getPrefixedName(serviceName, secretName);
    const secret = await this.#k8sCoreApi.readNamespacedSecret({
      name: fullName,
      namespace,
    });

    const entries: Record<string, string> = {};
    const data = secret.data ?? {};

    for (const [key, base64Value] of Object.entries(data)) {
      entries[key] = Buffer.from(base64Value, 'base64').toString('utf8');
    }

    return entries;
  }

  async upsertSecretEntry(input: {
    serviceName: string;
    namespace: string;
    secretName: string;
    key: string;
    value: string;
  }): Promise<void> {
    const { serviceName, namespace, secretName, key, value } = input;
    const fullName = this.#getPrefixedName(serviceName, secretName);

    this.#logger.info(
      `Upserting entry '${key}' for secret ${fullName} in ${namespace}`,
    );

    const secret = await this.#k8sCoreApi.readNamespacedSecret({
      name: fullName,
      namespace,
    });

    secret.data = secret.data ?? {};
    secret.data[key] = Buffer.from(value).toString('base64');

    await this.#upsertSecret(fullName, namespace, secret);
  }

  async deleteSecretEntry(input: {
    serviceName: string;
    namespace: string;
    secretName: string;
    key: string;
  }): Promise<void> {
    const { serviceName, namespace, secretName, key } = input;
    const fullName = this.#getPrefixedName(serviceName, secretName);

    this.#logger.info(
      `Deleting entry '${key}' from secret ${fullName} in ${namespace}`,
    );

    const secret = await this.#k8sCoreApi.readNamespacedSecret({
      name: fullName,
      namespace,
    });

    if (secret.data && secret.data[key]) {
      delete secret.data[key];
      await this.#upsertSecret(fullName, namespace, secret);
    }
  }

  // --- Private Helpers ---

  #getPrefixedName(serviceName: string, secretName: string): string {
    return secretName.startsWith(`${serviceName}-`)
      ? secretName
      : `${serviceName}-${secretName}`;
  }

  #mapSecretResponse(
    secret: k8s.V1Secret,
    fallbackNamespace: string,
  ): SecretDto {
    return {
      name: secret.metadata?.name ?? '',
      namespace: secret.metadata?.namespace ?? fallbackNamespace,
      createdAt: secret.metadata?.creationTimestamp?.toISOString(),
    };
  }

  async #upsertSecret(
    name: string,
    namespace: string,
    body: k8s.V1Secret,
  ): Promise<k8s.V1Secret> {
    try {
      await this.#k8sCoreApi.readNamespacedSecret({ name, namespace });
      const res = await this.#k8sCoreApi.replaceNamespacedSecret({
        name,
        namespace,
        body,
      });
      this.#logger.info(`Replaced secret: ${name}`);
      return res;
    } catch (err: any) {
      if (err instanceof k8s.ApiException) {
        try {
          const k8sStatus = JSON.parse(err.body);
          if (k8sStatus?.code === 404) {
            const res = await this.#k8sCoreApi.createNamespacedSecret({
              namespace,
              body,
            });
            this.#logger.info(`Created new secret: ${name}`);
            return res;
          }
        } catch (parseErr) {
          console.error('Failed to parse body:', parseErr);
        }
      }
      throw err;
    }
  }
}
