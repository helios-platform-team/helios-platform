import { ScmIntegrations } from '@backstage/integration';
import { InputError } from '@backstage/errors';
import { Config } from '@backstage/config';
import { LoggerService } from '@backstage/backend-plugin-api';
import { WebhookProvider } from './types';

export class GiteaWebhookProvider implements WebhookProvider {
  private readonly config: Config;

  constructor(config: Config) {
    this.config = config;
  }

  async createWebhook(
    repoUrl: string,
    webhookUrl: string,
    webhookSecret: string,
    events: string[],
    logger: LoggerService,
  ): Promise<void> {
    const url = new URL(`https://${repoUrl}`);
    const host = url.host;
    const owner = url.searchParams.get('owner');
    const repo = url.searchParams.get('repo');

    if (!owner || !repo) {
      throw new InputError(
        `Invalid repoUrl: could not extract owner and repo from ${repoUrl}`,
      );
    }

    const integrations = ScmIntegrations.fromConfig(this.config);
    const giteaIntegration = integrations.gitea.byHost(host);

    if (!giteaIntegration) {
      throw new InputError(
        `No Gitea integration found for host ${host}. Check your app-config.yaml integrations.gitea`,
      );
    }

    const { password, username } = giteaIntegration.config;

    if (!password) {
      throw new InputError(
        `Gitea integration for ${host} has no password/token configured`,
      );
    }

    const baseUrl = giteaIntegration.config.baseUrl ?? `http://${host}`;
    const apiUrl = `${baseUrl}/api/v1/repos/${owner}/${repo}/hooks`;

    const authHeader = username
      ? `Basic ${Buffer.from(`${username}:${password}`).toString('base64')}`
      : `token ${password}`;

    logger.info(`Creating webhook on ${owner}/${repo} -> ${webhookUrl}`);

    const response = await fetch(apiUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: authHeader,
      },
      body: JSON.stringify({
        type: 'gitea',
        active: true,
        events,
        config: {
          url: webhookUrl,
          content_type: 'json',
          secret: webhookSecret,
        },
      }),
    });

    if (!response.ok) {
      const body = await response.text();
      throw new Error(
        `Failed to create Gitea webhook: ${response.status} ${response.statusText} - ${body}`,
      );
    }

    logger.info('Gitea webhook created successfully');
  }
}
