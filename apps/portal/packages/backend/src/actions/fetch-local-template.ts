import { createTemplateAction } from '@backstage/plugin-scaffolder-node';
import fs from 'fs-extra';
import path from 'path';
import nunjucks from 'nunjucks';

// Helper function to recursively read files in a directory
async function getFiles(dir: string): Promise<string[]> {
  const dirents = await fs.readdir(dir, { withFileTypes: true });
  const files = await Promise.all(
    dirents.map((dirent: fs.Dirent) => {
      const res = path.resolve(dir, dirent.name);
      return dirent.isDirectory() ? getFiles(res) : Promise.resolve([res]);
    })
  );
  return Array.prototype.concat(...files);
}

export const createFetchLocalTemplateAction = () => {
  return createTemplateAction({
    id: 'fetch:local-template',
    schema: {
      input: z =>
        z.object({
          templateName: z
            .string()
            .describe('The name of the template folder under examples/'),
          targetPath: z
            .string()
            .describe('The target directory path in the workspace'),
          values: z
            .record(z.any())
            .describe('Values to render the template with'),
          subDir: z
            .string()
            .optional()
            .describe('Subdirectory under content/ (defaults to "source")'),
        }),
    },
    async handler(ctx) {
      const { templateName, targetPath, values, subDir = 'source' } = ctx.input as {
        templateName: string;
        targetPath: string;
        values: Record<string, any>;
        subDir?: string;
      };

      // Backstage backend is executed from apps/portal/packages/backend.
      // Sibling examples folder is at ../../examples/
      const examplesDir = path.resolve(process.cwd(), '../../examples');
      const sourceDir = path.resolve(examplesDir, templateName, 'content', subDir);

      if (!(await fs.pathExists(sourceDir))) {
        ctx.logger.error(`Local template directory does not exist: ${sourceDir}`);
        throw new Error(`Local template directory does not exist: ${sourceDir}`);
      }

      const destDir = path.resolve(ctx.workspacePath, targetPath);
      await fs.ensureDir(destDir);

      ctx.logger.info(`Fetching local template from: ${sourceDir} to: ${destDir}`);

      const env = new nunjucks.Environment(null, {
        autoescape: false,
        tags: {
          variableStart: '${{',
          variableEnd: '}}',
        },
      });

      const files = await getFiles(sourceDir);

      for (const srcFile of files) {
        const relativeFilePath = path.relative(sourceDir, srcFile);
        const destFile = path.resolve(destDir, relativeFilePath);

        await fs.ensureDir(path.dirname(destFile));

        const isBinary =
          srcFile.endsWith('.png') ||
          srcFile.endsWith('.jpg') ||
          srcFile.endsWith('.jpeg') ||
          srcFile.endsWith('.gif') ||
          srcFile.endsWith('.ico') ||
          srcFile.endsWith('.pdf') ||
          srcFile.endsWith('.zip') ||
          srcFile.endsWith('.gz') ||
          srcFile.endsWith('.tar') ||
          srcFile.endsWith('.jar') ||
          srcFile.endsWith('.war') ||
          srcFile.endsWith('.class');

        if (isBinary) {
          // Copy binary files directly without templating
          await fs.copy(srcFile, destFile);
        } else {
          // Read, render, and write text files
          const content = await fs.readFile(srcFile, 'utf8');
          try {
            const rendered = env.renderString(content, { values });
            await fs.writeFile(destFile, rendered, 'utf8');
          } catch (err: any) {
            ctx.logger.error(`Failed to template file ${relativeFilePath}: ${err.message}`);
            throw err;
          }
        }
      }

      ctx.logger.info(`Successfully fetched and templated local template: ${templateName}/${subDir}`);
    },
  });
};
