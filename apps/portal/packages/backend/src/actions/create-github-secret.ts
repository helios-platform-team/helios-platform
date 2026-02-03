import { createTemplateAction, executeShellCommand } from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

/**
 * Custom action to create GitHub credentials secret in Kubernetes.
 * Reads GITHUB_TOKEN from environment variable to avoid template substitution issues.
 */
export const createGithubCredentialsSecretAction = () => {
    return createTemplateAction({
        id: 'kubernetes:create-github-secret',
        description: 'Creates a GitHub credentials secret in Kubernetes using the server-side GITHUB_TOKEN',
        schema: {
            input: z => z.object({
                name: z.string().describe('Name of the application (used for secret naming)'),
                namespace: z.string().optional().describe('Kubernetes namespace (defaults to "default")'),
                username: z.string().describe('GitHub username'),
                webhookSecret: z.string().optional().describe('Webhook secret token'),
            }),
        },
        async handler(ctx) {
            const { name, namespace = 'default', username, webhookSecret = '' } = ctx.input;

            // Get token from environment - this runs server-side so has access to env vars
            const token = process.env.GITHUB_TOKEN;

            if (!token) {
                throw new InputError('GITHUB_TOKEN environment variable is not set on the Backstage server');
            }

            const secretName = `github-credentials-${name}`;

            ctx.logger.info(`Creating secret ${secretName} in namespace ${namespace}`);

            // Delete existing secret if it exists (ignore errors)
            try {
                await executeShellCommand({
                    command: 'kubectl',
                    args: ['delete', 'secret', secretName, '-n', namespace, '--ignore-not-found'],
                    logger: ctx.logger,
                });
            } catch (e) {
                // Ignore deletion errors
            }

            // Create the secret
            await executeShellCommand({
                command: 'kubectl',
                args: [
                    'create', 'secret', 'generic', secretName,
                    '-n', namespace,
                    `--from-literal=token=${token}`,
                    `--from-literal=password=${token}`,
                    `--from-literal=username=${username}`,
                    `--from-literal=secretToken=${webhookSecret}`,
                ],
                logger: ctx.logger,
            });

            ctx.logger.info(`Successfully created secret: ${secretName}`);
        },
    });
};
