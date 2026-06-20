import { createTemplateAction } from '@backstage/plugin-scaffolder-node';
import { Config } from '@backstage/config';
import { createWebhookProvider } from '../providers/factory';

export const createGitWebhookAction = (options: { config: Config }) => {
  const { config } = options;

  return createTemplateAction({
    id: 'git:create-webhook',
    description: 'Creates a webhook on the configured git provider repository',
    schema: {
      input: z =>
        z.object({
          repoUrl: z
            .string()
            .describe(
              'Repository URL in Backstage format (host?owner=x&repo=y)',
            ),
          webhookUrl: z.string().describe('Target URL for the webhook'),
          webhookSecret: z
            .string()
            .optional()
            .describe('Secret for webhook payload verification'),
          events: z
            .array(z.string())
            .optional()
            .describe('Events to trigger the webhook (defaults to ["push"])'),
        }),
    },
    async handler(ctx) {
      const {
        repoUrl,
        webhookUrl,
        webhookSecret = '',
        events = ['push'],
      } = ctx.input;

      const provider = createWebhookProvider(config);
      await provider.createWebhook(
        repoUrl,
        webhookUrl,
        webhookSecret,
        events,
        ctx.logger,
      );
    },
  });
};
