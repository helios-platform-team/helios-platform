import { Config } from '@backstage/config';
import { WebhookProvider } from './types';
import { GiteaWebhookProvider } from './gitea';
import { GitHubWebhookProvider } from './github';

export function createWebhookProvider(config: Config): WebhookProvider {
  const providerType = process.env.GIT_PROVIDER_TYPE || 'gitea';

  switch (providerType) {
    case 'github':
      return new GitHubWebhookProvider(config);
    case 'gitea':
    default:
      return new GiteaWebhookProvider(config);
  }
}
