import {
  createTemplateAction,
  executeShellCommand,
} from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

function getGitToken(): string {
  return (
    process.env.GIT_TOKEN ||
    process.env.GITEA_TOKEN ||
    process.env.GITHUB_TOKEN ||
    ''
  );
}

/**
 * Custom action to create Git credentials secret in Kubernetes.
 * Reads the git token from GIT_TOKEN env var (or provider-specific fallbacks).
 */
export const createGitCredentialsSecretAction = () => {
  return createTemplateAction({
    id: 'kubernetes:create-git-credentials-secret',
    description:
      'Creates a Git credentials secret in Kubernetes using the server-side git token',
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

      const token = getGitToken();

      if (!token) {
        throw new InputError(
          'Git token is not set. Configure GIT_TOKEN, GITEA_TOKEN, or GITHUB_TOKEN on the Backstage server',
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
