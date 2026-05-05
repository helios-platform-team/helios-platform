/*
 * Copyright 2025 The Backstage Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {
  LoggerService,
  BackstageCredentials,
  BackstageUserPrincipal,
} from '@backstage/backend-plugin-api';
import * as k8s from '@kubernetes/client-node';
import { randomUUID } from 'crypto';
import { promises as fs } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { promisify } from 'util';
import { execFile as execFileCb } from 'child_process';
import {
  HELIOS_CRD,
  HeliosAppResource,
  HeliosComponent,
  HeliosTrait,
  K8sSecretService,
  SecretResponse,
} from './types';

const externalSecretReferenceTraitType = 'external-secret-reference';
const execFile = promisify(execFileCb);

// --- Implementation ---
export class K8sSecretServiceImpl implements K8sSecretService {
  readonly #logger: LoggerService;
  readonly #k8sCoreApi: k8s.CoreV1Api;
  readonly #k8sCustomApi: k8s.CustomObjectsApi;

  constructor(logger: LoggerService) {
    this.#logger = logger;

    const kc = new k8s.KubeConfig();
    kc.loadFromDefault();
    this.#k8sCoreApi = kc.makeApiClient(k8s.CoreV1Api);
    this.#k8sCustomApi = kc.makeApiClient(k8s.CustomObjectsApi);
  }

  async listSecrets(
    serviceName: string,
    namespace: string,
  ): Promise<SecretResponse[]> {
    const response = await this.#k8sCoreApi.listNamespacedSecret({ namespace });
    const prefix = `${serviceName}-`;

    return response.items
      .filter(s => s.metadata?.name?.startsWith(prefix))
      .map(s => this.#mapSecretResponse(s, namespace));
  }

  async createSecret(
    input: {
      serviceName: string;
      namespace: string;
      secretName: string;
      secretData: Record<string, string>;
      entityRef?: string;
    },
    options: { credentials: BackstageCredentials<BackstageUserPrincipal> },
  ): Promise<SecretResponse> {
    const { serviceName, secretName, namespace, secretData, entityRef } = input;
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
      stringData: secretData,
    };

    const responseBody = await this.#upsertSecret(
      fullName,
      namespace,
      manifest,
    );

    // Update HeliosApp desired state (trait) so operator+CUE handles GitOps rendering and push.
    await this.#upsertExternalSecretReferenceTrait(
      serviceName,
      namespace,
      fullName,
    );

    this.#logger.info(
      `Secret ${fullName} applied and HeliosApp trait updated for ${serviceName}. ` +
        `Operator reconcile will render with CUE and push manifest changes to GitOps.`,
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
  ): SecretResponse {
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
          const k8sCode = k8sStatus?.code;

          if (k8sCode === 404) {
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

  async #upsertExternalSecretReferenceTrait(
    serviceName: string,
    namespace: string,
    secretName: string,
  ): Promise<void> {
    try {
      const heliosApp = (await this.#k8sCustomApi.getNamespacedCustomObject({
        ...HELIOS_CRD,
        namespace,
        name: serviceName,
      })) as HeliosAppResource;

      const components = heliosApp.spec?.components;
      if (!Array.isArray(components)) {
        this.#logger.warn(
          `Skipping HeliosApp trait update for ${serviceName}: components missing in spec`,
        );
        return;
      }

      this.#logger.info(`Components: ${JSON.stringify(components, null, 2)}`);

      const targetIndex = components.findIndex(c => c?.name === serviceName);
      if (targetIndex < 0) {
        this.#logger.warn(
          `Skipping HeliosApp trait update for ${serviceName}: component ${serviceName} not found`,
        );
        return;
      }

      const updatedComponents = [...components];
      updatedComponents[targetIndex] = this.#upsertTraitOnComponent(
        components[targetIndex],
        secretName,
      );

      const body: HeliosAppResource = {
        ...heliosApp,
        spec: {
          ...heliosApp.spec,
          components: updatedComponents,
        },
      };

      await this.#k8sCustomApi.replaceNamespacedCustomObject({
        ...HELIOS_CRD,
        namespace,
        name: serviceName,
        body,
      });

      await this.#syncHeliosAppToGitOps(
        namespace,
        serviceName,
        body,
      );
    } catch (err: any) {
      this.#logger.error(
        `HeliosApp trait update failed for ${serviceName}: ${err.message}`,
      );
    }
  }

  async #syncHeliosAppToGitOps(
    namespace: string,
    serviceName: string,
    heliosApp: HeliosAppResource,
  ): Promise<void> {
    const repoUrl = heliosApp.spec?.gitopsRepo;
    const repoPath = heliosApp.spec?.gitopsPath;
    if (!repoUrl || !repoPath) {
      this.#logger.warn(
        `Skipping GitOps HeliosApp sync for ${serviceName}: gitopsRepo/gitopsPath missing`,
      );
      return;
    }

    const fileName = process.env.HELIOS_GITOPS_HELIOSAPP_FILE || 'helios-app.yaml';
    const targetFile = `${repoPath.replace(/\/+$/, '')}/${fileName}`;
    const { username, token } = await this.#resolveGitCredentials(namespace, heliosApp);
    if (!token) {
      this.#logger.warn(
        `Skipping GitOps HeliosApp sync for ${serviceName}: token missing (gitopsSecretRef/env)`,
      );
      return;
    }

    const authRepoUrl = this.#buildAuthRepoUrl(repoUrl, username, token);
    const workDir = join(tmpdir(), `k8s-secret-gitops-${randomUUID()}`);
    await fs.mkdir(workDir, { recursive: true });

    try {
      await execFile('git', ['clone', '--depth', '1', authRepoUrl, workDir]);

      const manifest = {
        apiVersion: 'app.helios.io/v1alpha1',
        kind: 'HeliosApp',
        metadata: {
          name: serviceName,
          namespace,
        },
        spec: heliosApp.spec,
      };
      await fs.mkdir(join(workDir, repoPath), { recursive: true });
      await fs.writeFile(
        join(workDir, targetFile),
        `${JSON.stringify(manifest, null, 2)}\n`,
        'utf8',
      );

      await execFile('git', ['-C', workDir, 'add', targetFile]);
      const status = await execFile('git', ['-C', workDir, 'status', '--porcelain']);
      if (!status.stdout.trim()) {
        this.#logger.info(`GitOps HeliosApp unchanged for ${serviceName}, skipping commit`);
        return;
      }

      await execFile('git', [
        '-C',
        workDir,
        'commit',
        '-m',
        `Update HeliosApp external-secret-reference for ${serviceName}`,
      ]);
      await execFile('git', ['-C', workDir, 'push']);
      this.#logger.info(`Synced ${targetFile} to GitOps for ${serviceName}`);
    } catch (err: any) {
      this.#logger.error(
        `GitOps HeliosApp sync failed for ${serviceName}: ${err.message}`,
      );
    } finally {
      await fs.rm(workDir, { recursive: true, force: true });
    }
  }

  async #resolveGitCredentials(
    namespace: string,
    heliosApp: HeliosAppResource,
  ): Promise<{ username: string; token: string }> {
    let token = process.env.GITEA_TOKEN ?? '';
    let username = process.env.GITEA_USER ?? 'git';
    const secretRef = heliosApp.spec?.gitopsSecretRef;

    if (!secretRef) {
      return { username, token };
    }

    try {
      const secret = await this.#k8sCoreApi.readNamespacedSecret({
        namespace,
        name: secretRef,
      });
      const data = secret.data ?? {};
      const decode = (v?: string) => (v ? Buffer.from(v, 'base64').toString('utf8') : '');

      token = decode(data.token) || decode(data.password) || token;
      username = decode(data.username) || username;
    } catch (err: any) {
      this.#logger.warn(
        `Failed to read git credentials secret ${secretRef} for ${namespace}: ${err.message}`,
      );
    }

    return { username, token };
  }

  #buildAuthRepoUrl(repoUrl: string, username: string, token: string): string {
    const url = new URL(repoUrl);
    url.username = username;
    url.password = token;
    return url.toString();
  }

  #upsertTraitOnComponent(
    component: HeliosComponent,
    secretName: string,
  ): HeliosComponent {
    const traits = Array.isArray(component.traits) ? [...component.traits] : [];
    const existingTraitIdx = traits.findIndex(
      t => t?.type === externalSecretReferenceTraitType,
    );

    const externalSecretTrait: HeliosTrait = {
      type: externalSecretReferenceTraitType,
      properties: { secretName },
    };

    if (existingTraitIdx >= 0) {
      traits[existingTraitIdx] = externalSecretTrait;
    } else {
      traits.push(externalSecretTrait);
    }

    return {
      ...component,
      traits,
    };
  }
}
