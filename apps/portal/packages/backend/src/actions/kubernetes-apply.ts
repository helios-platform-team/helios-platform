import { createTemplateAction, executeShellCommand } from '@backstage/plugin-scaffolder-node';
import { InputError } from '@backstage/errors';

export const createKubernetesApplyAction = () => {
    return createTemplateAction({
        id: 'kubernetes:apply',
        description: 'Applies a Kubernetes manifest file using kubectl',
        schema: {
            input: z => z.object({
                manifestPath: z.string().describe('Path to the manifest file to apply, relative to the workspace'),
                namespace: z.string().optional().describe('Kubernetes namespace to apply resources to'),
                namespaced: z.boolean().optional().describe('Whether the resources are namespaced'),
            }),
        },
        async handler(ctx) {
            const { manifestPath, namespace } = ctx.input;

            if (!manifestPath) {
                throw new InputError('manifestPath is required');
            }

            const args = ['apply', '-f', manifestPath];
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

            ctx.logger.info(`Successfully applied manifest: ${manifestPath}`);
        },
    });
};
