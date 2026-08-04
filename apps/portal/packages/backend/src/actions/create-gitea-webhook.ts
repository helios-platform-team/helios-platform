import { createTemplateAction } from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';
import { ScmIntegrations } from '@backstage/integration';
import { Config } from '@backstage/config';

export const createGiteaWebhookAction = (options: { config: Config }) => {
  const { config } = options;

  return createTemplateAction({
    id: 'gitea:create-webhook',
    description: 'Creates a webhook on a Gitea repository',
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

      const url = new URL(`https://${repoUrl}`);
      const host = url.host;
      const owner = url.searchParams.get('owner');
      const repo = url.searchParams.get('repo');

      if (!owner || !repo) {
        throw new InputError(
          `Invalid repoUrl: could not extract owner and repo from ${repoUrl}`,
        );
      }

      const integrations = ScmIntegrations.fromConfig(config);
      const giteaIntegration = integrations.gitea.byHost(host);

      if (!giteaIntegration) {
        throw new InputError(
          `No Gitea integration found for host ${host}. Check your app-config.yaml integrations.gitea`,
        );
      }

      const { password, username } = giteaIntegration.config as any;

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

      ctx.logger.info(`Creating webhook on ${owner}/${repo} -> ${webhookUrl}`);

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

      ctx.logger.info('Gitea webhook created successfully');
    },
  });
};
