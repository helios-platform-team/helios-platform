import { createTemplateAction, executeShellCommand } from '@backstage/plugin-scaffolder-node';
import { z } from 'zod';
import { Writable } from 'stream';

export const createRunKubectlAction = () => {
  return createTemplateAction({
    id: 'kubernetes:apply',
    description: 'Runs kubectl apply on a file',
    schema: {
      // FIX LỖI ZOD: Thêm 'as any' để bỏ qua lỗi kiểm tra kiểu TypeScript
      input: z.object({
        manifestPath: z.string().describe('The path to the yaml file to apply'),
        namespaced: z.boolean().default(true).describe('Whether to use namespace'),
      }) as any, 
    },
    async handler(ctx) {
      ctx.logger.info(`Attempting to apply manifest: ${ctx.input.manifestPath}`);

      // FIX LỖI LOGSTREAM: Tự tạo stream để hứng log từ kubectl và đẩy vào logger của Backstage
      const myLogStream = new Writable({
        write(chunk, _encoding, next) {
          const msg = chunk.toString().trim();
          // Đẩy log vào giao diện Backstage
          if (msg) ctx.logger.info(msg);
          next();
        },
      });

      try {
        await executeShellCommand({
          command: 'kubectl',
          args: ['apply', '-f', ctx.input.manifestPath],
          logStream: myLogStream, // Dùng stream tự tạo, không dùng ctx.logStream cũ
          options: {
            cwd: ctx.workspacePath,
            // Copy biến môi trường để kubectl tìm thấy config của Minikube
            env: { ...process.env },
          },
        });
        ctx.logger.info(`Successfully applied manifest to Kubernetes!`);
      } catch (error) {
        ctx.logger.error(`Failed to run kubectl command.`);
        throw error;
      }
    },
  });
};