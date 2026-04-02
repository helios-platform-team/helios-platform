import {
  createTemplateAction,
  executeShellCommand,
} from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

/**
 * Custom action to create Git credentials secret in Kubernetes.
 * Reads GITEA_TOKEN from environment variable to avoid template substitution issues.
 */
export const createGitCredentialsSecretAction = () => {
  return createTemplateAction({
    id: 'kubernetes:create-git-credentials-secret',
    description:
      'Creates a Git credentials secret in Kubernetes using the server-side GITEA_TOKEN',
    schema: {
      input: z =>
        z.object({
          name: z
            .string()
            .describe('Name of the application (used for secret naming)'),
          namespace: z
            .string()
            .optional()
            .describe('Kubernetes namespace (defaults to "default")'),
          username: z.string().describe('Git username'),
          webhookSecret: z.string().optional().describe('Webhook secret token'),
        }),
    },
    async handler(ctx) {
      const {
        name,
        namespace = 'default',
        username,
        webhookSecret = '',
      } = ctx.input;

      const token = process.env.GITEA_TOKEN;

      if (!token) {
        throw new InputError(
          'GITEA_TOKEN environment variable is not set on the Backstage server',
        );
      }

      const secretName = `git-credentials-${name}`;

      ctx.logger.info(
        `Creating secret ${secretName} in namespace ${namespace}`,
      );

      try {
        await executeShellCommand({
          command: 'kubectl',
          args: [
            'delete',
            'secret',
            secretName,
            '-n',
            namespace,
            '--ignore-not-found',
          ],
          logger: ctx.logger,
        });
      } catch (e) {
        // Ignore deletion errors
      }

      await executeShellCommand({
        command: 'kubectl',
        args: [
          'create',
          'secret',
          'generic',
          secretName,
          '-n',
          namespace,
          `--from-literal=token=${token}`,
          `--from-literal=password=${token}`,
          `--from-literal=username=${username}`,
          // Tekton interceptor expects the webhook secret under key `secret`.
          `--from-literal=secret=${webhookSecret}`,
          // Keep `secretToken` for backwards compatibility.
          `--from-literal=secretToken=${webhookSecret}`,
        ],
        logger: ctx.logger,
      });

      ctx.logger.info(`Successfully created secret: ${secretName}`);
    },
  });
};
