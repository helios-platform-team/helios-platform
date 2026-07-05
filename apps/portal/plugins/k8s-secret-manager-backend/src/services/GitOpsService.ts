import { LoggerService } from '@backstage/backend-plugin-api';
import * as k8s from '@kubernetes/client-node';
import { randomUUID } from 'crypto';
import { promises as fs } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';
import { promisify } from 'util';
import { execFile as execFileCb } from 'child_process';
import * as yaml from 'js-yaml';
import { HeliosAppResource } from './K8sSecretService';

const execFile = promisify(execFileCb);

export class GitOpsService {
  constructor(
    private readonly logger: LoggerService,
    private readonly k8sCoreApi: k8s.CoreV1Api,
  ) {}

  /**
   * Clones the repository to a temporary directory and returns the working path
   * along with the parsed HeliosApp resource from the YAML file.
   */
  async checkoutAndFetchHeliosApp(
    namespace: string,
    serviceName: string,
    gitopsRepo: string,
    gitopsPath: string,
    gitopsSecretRef?: string,
  ): Promise<{
    workDir: string;
    targetFile: string;
    heliosApp: HeliosAppResource | null;
  }> {
    const fileName =
      process.env.HELIOS_GITOPS_HELIOSAPP_FILE || 'helios-app.yaml';
    const cleanRepoPath = gitopsPath.replace(/\/+$/, '');
    const targetFile = `${cleanRepoPath}/${fileName}`;

    const { username, token } = await this.#resolveGitCredentials(
      namespace,
      gitopsSecretRef,
    );
    if (!token) {
      throw new Error(`GitOps token missing for service ${serviceName}`);
    }

    const authRepoUrl = this.#buildAuthRepoUrl(gitopsRepo, username, token);
    const workDir = join(tmpdir(), `k8s-secret-gitops-${randomUUID()}`);
    await fs.mkdir(workDir, { recursive: true });

    try {
      await execFile('git', ['clone', '--depth', '1', authRepoUrl, workDir]);
      const fullFilePath = join(workDir, targetFile);

      let heliosApp: HeliosAppResource | null = null;
      try {
        const fileContent = await fs.readFile(fullFilePath, 'utf8');
        heliosApp = yaml.load(fileContent) as HeliosAppResource;
      } catch (fileErr: any) {
        if (fileErr.code !== 'ENOENT') throw fileErr;
        this.logger.info(
          `File ${targetFile} does not exist yet. It will be created.`,
        );
      }

      return { workDir, targetFile, heliosApp };
    } catch (err: any) {
      await fs.rm(workDir, { recursive: true, force: true });
      throw err;
    }
  }

  /**
   * Commits and pushes changes from an active local workspace directory
   */
  async commitAndPushSecretChange(
    workDir: string,
    targetFile: string,
    serviceName: string,
    heliosAppSpec: any,
    namespace: string,
  ): Promise<void> {
    try {
      const manifest = {
        apiVersion: 'app.helios.io/v1alpha1',
        kind: 'HeliosApp',
        metadata: {
          name: serviceName,
          namespace,
        },
        spec: heliosAppSpec,
      };

      const yamlString = yaml.dump(manifest, {
        indent: 2,
        noRefs: true,
        noArrayIndent: true,
      });

      const fullFilePath = join(workDir, targetFile);
      await fs.mkdir(join(fullFilePath, '..'), { recursive: true });
      await fs.writeFile(fullFilePath, yamlString, 'utf8');

      await execFile('git', ['-C', workDir, 'add', targetFile]);
      const status = await execFile('git', [
        '-C',
        workDir,
        'status',
        '--porcelain',
      ]);

      if (!status.stdout.trim()) {
        this.logger.info(
          `GitOps HeliosApp unchanged for ${serviceName}, skipping commit`,
        );
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
      this.logger.info(
        `Synced ${targetFile} to GitOps repository for ${serviceName}`,
      );
    } finally {
      await fs.rm(workDir, { recursive: true, force: true });
    }
  }

  async #resolveGitCredentials(
    namespace: string,
    secretRef?: string,
  ): Promise<{ username: string; token: string }> {
    let token = process.env.GITEA_TOKEN ?? '';
    let username = process.env.GITEA_USER ?? 'git';

    if (!secretRef) return { username, token };

    try {
      const secret = await this.k8sCoreApi.readNamespacedSecret({
        namespace,
        name: secretRef,
      });
      const data = secret.data ?? {};
      const decode = (v?: string) =>
        v ? Buffer.from(v, 'base64').toString('utf8') : '';

      token = decode(data.token) || decode(data.password) || token;
      username = decode(data.username) || username;
    } catch (err: any) {
      this.logger.warn(
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
}
