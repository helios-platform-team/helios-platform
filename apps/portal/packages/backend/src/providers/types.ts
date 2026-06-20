import { LoggerService } from '@backstage/backend-plugin-api';

export interface WebhookProvider {
  createWebhook(
    repoUrl: string,
    webhookUrl: string,
    webhookSecret: string,
    events: string[],
    logger: LoggerService,
  ): Promise<void>;
}
