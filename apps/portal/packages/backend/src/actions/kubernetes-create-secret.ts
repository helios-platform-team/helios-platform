import {
  createTemplateAction,
  executeShellCommand,
} from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

export const createKubernetesCreateSecretAction = () => {
  return createTemplateAction({
    id: 'kubernetes:create-secret',
    description: 'Creates (or replaces) a Kubernetes Secret using kubectl',
    schema: {
      input: z =>
        z.object({
          namespace: z
            .string()
            .optional()
            .describe('Kubernetes namespace (defaults to "default")'),
          secretName: z.string().describe('Name of the Secret to create'),
          type: z
            .string()
            .optional()
            .describe('Optional Secret type (defaults to Opaque)'),
          data: z
            .record(z.string())
            .describe(
              'Key/value pairs to populate the Secret (values are treated as stringData)',
            ),
        }),
    },
    async handler(ctx) {
      const { namespace = 'default', secretName, type, data } = ctx.input;

      if (!secretName) {
        throw new InputError('secretName is required');
      }
      if (!data || Object.keys(data).length === 0) {
        throw new InputError('data must contain at least one key/value pair');
      }

      ctx.logger.info(
        `Creating secret ${secretName} in namespace ${namespace} with ${Object.keys(data).length} keys`,
      );

      // Delete existing secret if it exists (ignore errors)
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
      } catch {
        // ignore
      }

      const args = ['create', 'secret', 'generic', secretName, '-n', namespace];

      if (type) {
        args.push(`--type=${type}`);
      }

      for (const [key, value] of Object.entries(data)) {
        if (!key) {
          throw new InputError('Secret data keys must be non-empty');
        }
        args.push(`--from-literal=${key}=${value}`);
      }

      await executeShellCommand({
        command: 'kubectl',
        args,
        logger: ctx.logger,
      });

      ctx.logger.info(`Successfully created secret: ${secretName}`);
    },
  });
};
