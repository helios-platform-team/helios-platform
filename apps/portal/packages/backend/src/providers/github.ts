import { ScmIntegrations } from '@backstage/integration';
import { InputError } from '@backstage/errors';
import { Config } from '@backstage/config';
import { LoggerService } from '@backstage/backend-plugin-api';
import { WebhookProvider } from './types';

export class GitHubWebhookProvider implements WebhookProvider {
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
    const githubIntegration = integrations.github.byHost(host);

    if (!githubIntegration) {
      throw new InputError(
        `No GitHub integration found for host ${host}. Check your app-config.yaml integrations.github`,
      );
    }

    const token = githubIntegration.config.token;

    if (!token) {
      throw new InputError(
        `GitHub integration for ${host} has no token configured`,
      );
    }

    const apiUrl =
      host === 'github.com'
        ? `https://api.github.com/repos/${owner}/${repo}/hooks`
        : `${githubIntegration.config.apiBaseUrl}/repos/${owner}/${repo}/hooks`;

    logger.info(`Creating webhook on ${owner}/${repo} -> ${webhookUrl}`);

    const response = await fetch(apiUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `token ${token}`,
        Accept: 'application/vnd.github+json',
      },
      body: JSON.stringify({
        name: 'web',
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
        `Failed to create GitHub webhook: ${response.status} ${response.statusText} - ${body}`,
      );
    }

    logger.info('GitHub webhook created successfully');
  }
}
