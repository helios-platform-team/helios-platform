import {
  createTemplateAction,
  executeShellCommand,
} from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

export const createKubernetesApplyAction = () => {
  return createTemplateAction({
    id: 'kubernetes:apply',
    description: 'Applies a Kubernetes manifest file using kubectl',
    schema: {
      input: z =>
        z
          .object({
            manifestPath: z
              .string()
              .optional()
              .describe(
                'Path to the manifest file to apply, relative to the workspace',
              ),
          resource: z
            .string()
            .optional()
            .describe(
              'Alias for manifestPath (kept for backwards compatibility with older templates)',
            ),
          namespace: z
            .string()
            .optional()
            .describe('Kubernetes namespace to apply resources to'),
          namespaced: z
            .boolean()
            .optional()
            .describe('Whether the resources are namespaced'),
          values: z
            .record(z.any())
            .optional()
            .describe(
              'Optional values (ignored by this action; use fetch:template to render manifests)',
            ),
          })
          .refine(v => Boolean(v.manifestPath || v.resource), {
            message: 'manifestPath (or resource) is required',
          }),
    },
    async handler(ctx) {
      const { manifestPath, resource, namespace, values } = ctx.input;

      const effectiveManifestPath = manifestPath ?? resource;

      if (!effectiveManifestPath) {
        throw new InputError('manifestPath (or resource) is required');
      }

      if (values && Object.keys(values).length > 0) {
        ctx.logger.warn(
          'kubernetes:apply received input.values but does not perform templating; ensure manifests are rendered by fetch:template before applying',
        );
      }

      const args = ['apply', '--server-side', '-f', effectiveManifestPath];
      if (namespace) {
        args.push('-n', namespace);
      }

      ctx.logger.info(`Running kubectl ${args.join(' ')}`);

      await executeShellCommand({
        command: 'kubectl',
        args: args,
        logger: ctx.logger,
        options: {
          cwd: ctx.workspacePath,
        },
      });

      ctx.logger.info(
        `Successfully applied manifest: ${effectiveManifestPath}`,
      );
    },
  });
};

