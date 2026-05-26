import { HttpAuthService } from '@backstage/backend-plugin-api';
import { InputError } from '@backstage/errors';
import { z } from 'zod';
import express from 'express';
import Router from 'express-promise-router';
import { k8sSecretServiceRef } from './services/K8sSecretService';

export async function createRouter({
  httpAuth,
  secretService,
}: {
  httpAuth: HttpAuthService;
  secretService: typeof k8sSecretServiceRef.T;
}): Promise<express.Router> {
  const router = Router();
  router.use(express.json());

  const secretSchema = z.object({
    serviceName: z.string().min(1),
    secretName: z.string().min(1),
    namespace: z.string().min(1),
    secretData: z.record(z.string(), z.string()).optional(), // Key-value pairs
    entityRef: z.string().optional(),
  });

  const entrySchema = z.object({
    value: z.string(),
  });

  // GET: List secrets with filtering
  router.get('/secrets', async (req, res) => {
    const namespace = String(req.query.namespace || 'default');
    const serviceName = String(req.query.serviceName);
    const limit = req.query.limit ? parseInt(String(req.query.limit), 10) : 10;

    if (!req.query.serviceName) {
      throw new InputError('serviceName is required');
    }

    const continueToken = req.query.continueToken
      ? String(req.query.continueToken)
      : undefined;

    const secrets = await secretService.listSecrets(
      serviceName,
      namespace,
      limit,
      continueToken,
    );
    res.json(secrets);
  });

  // POST: Create or update a secret (with prefixing logic in service)
  router.post('/secrets', async (req, res) => {
    const parsed = secretSchema.safeParse(req.body);

    if (!parsed.success) {
      throw new InputError(parsed.error.toString());
    }

    const credentials = await httpAuth.credentials(req, { allow: ['user'] });

    const result = await secretService.createSecret(parsed.data, {
      credentials,
    });

    res.status(201).json(result);
  });

  // DELETE: Remove a secret
  router.delete('/secrets/:namespace/:serviceName/:name', async (req, res) => {
    const { namespace, serviceName, name } = req.params;

    await secretService.deleteSecret(serviceName, name, namespace);
    res.status(204).end();
  });

  // --- Entry Level Endpoints ---

  // GET: List all key-value entries of a specific secret
  router.get(
    '/secrets/:namespace/:serviceName/:secretName/entries',
    async (req, res) => {
      const { namespace, serviceName, secretName } = req.params;

      const entries = await secretService.getSecretEntries(
        serviceName,
        secretName,
        namespace,
      );

      res.json(entries);
    },
  );

  // PUT: Upsert a single entry (key/value pair) inside a secret
  router.put(
    '/secrets/:namespace/:serviceName/:secretName/entries/:key',
    async (req, res) => {
      const { namespace, serviceName, secretName, key } = req.params;
      const parsed = entrySchema.safeParse(req.body);

      if (!parsed.success) {
        throw new InputError(parsed.error.toString());
      }

      await secretService.upsertSecretEntry({
        serviceName,
        namespace,
        secretName,
        key,
        value: parsed.data.value,
      });

      res.status(200).end();
    },
  );

  // DELETE: Remove a single entry from a secret
  router.delete(
    '/secrets/:namespace/:serviceName/:secretName/entries/:key',
    async (req, res) => {
      const { namespace, serviceName, secretName, key } = req.params;

      await secretService.deleteSecretEntry({
        serviceName,
        namespace,
        secretName,
        key,
      });

      res.status(204).end();
    },
  );

  return router;
}
