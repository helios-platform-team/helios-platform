import {
  mockCredentials,
  mockErrorHandler,
  mockServices,
} from '@backstage/backend-test-utils';
import express from 'express';
import request from 'supertest';

import { createRouter } from './router';
import { k8sSecretServiceRef } from './services/K8sSecretService';

const mockSecretResponse = {
  name: 'my-service-db-creds',
  namespace: 'default',
  createdAt: new Date().toISOString(),
};

const mockEntryResponse = [
  { key: 'username', value: 'admin' },
  { key: 'password', value: 'secret' },
];

describe('createRouter', () => {
  let app: express.Express;
  let secretService: jest.Mocked<typeof k8sSecretServiceRef.T>;

  beforeEach(async () => {
    secretService = {
      listSecrets: jest.fn(),
      createSecret: jest.fn(),
      deleteSecret: jest.fn(),
      getSecretEntries: jest.fn(),
      upsertSecretEntry: jest.fn(),
      deleteSecretEntry: jest.fn(),
    } as any;

    const router = await createRouter({
      httpAuth: mockServices.httpAuth(),
      secretService,
    });

    app = express();
    app.use(router);
    app.use(mockErrorHandler());
  });

  describe('GET /secrets', () => {
    it('should return a list of secrets with default query parameters', async () => {
      secretService.listSecrets.mockResolvedValue([mockSecretResponse] as any);

      const response = await request(app).get(
        '/secrets?namespace=default&serviceName=my-service',
      );

      expect(response.status).toBe(200);
      expect(response.body).toEqual([mockSecretResponse]);
      expect(secretService.listSecrets).toHaveBeenCalledWith(
        'my-service',
        'default',
        10,
        undefined,
      );
    });

    it('should pass pagination and token parameters to the service', async () => {
      secretService.listSecrets.mockResolvedValue([] as any);

      await request(app).get(
        '/secrets?namespace=prod&serviceName=my-service&limit=25&continueToken=token123',
      );

      expect(secretService.listSecrets).toHaveBeenCalledWith(
        'my-service',
        'prod',
        25,
        'token123',
      );
    });

    it('should return 400 if serviceName is omitted', async () => {
      const response = await request(app).get('/secrets?namespace=default');

      expect(response.status).toBe(400);
      expect(secretService.listSecrets).not.toHaveBeenCalled();
    });
  });

  describe('POST /secrets', () => {
    it('should create a secret successfully', async () => {
      secretService.createSecret.mockResolvedValue(mockSecretResponse as any);

      const payload = {
        serviceName: 'my-service',
        secretName: 'db-creds',
        namespace: 'default',
        secretData: { username: 'admin' },
      };

      const response = await request(app).post('/secrets').send(payload);

      expect(response.status).toBe(201);
      expect(response.body).toEqual(mockSecretResponse);
      expect(secretService.createSecret).toHaveBeenCalledWith(
        payload,
        expect.any(Object),
      );
    });

    it('should fail validation if a required field is missing', async () => {
      // secretData is optional in schema, so omitting serviceName to trigger validation error
      const response = await request(app).post('/secrets').send({
        secretName: 'db-creds',
        namespace: 'default',
      });

      expect(response.status).toBe(400);
      expect(secretService.createSecret).not.toHaveBeenCalled();
    });

    it('should not allow unauthenticated requests', async () => {
      const response = await request(app)
        .post('/secrets')
        .set('Authorization', mockCredentials.none.header())
        .send({
          serviceName: 'my-service',
          secretName: 'db-creds',
          namespace: 'default',
          secretData: { key: 'val' },
        });

      expect(response.status).toBe(401);
    });
  });

  describe('DELETE /secrets/:namespace/:serviceName/:name', () => {
    it('should delete a secret', async () => {
      secretService.deleteSecret.mockResolvedValue(undefined as any);

      const response = await request(app).delete(
        '/secrets/default/my-service/db-creds',
      );

      expect(response.status).toBe(204);
      expect(secretService.deleteSecret).toHaveBeenCalledWith(
        'my-service',
        'db-creds',
        'default',
      );
    });
  });

  describe('GET /secrets/:namespace/:serviceName/:secretName/entries', () => {
    it('should return all key-value entries of a specific secret', async () => {
      secretService.getSecretEntries.mockResolvedValue(
        mockEntryResponse as any,
      );

      const response = await request(app).get(
        '/secrets/default/my-service/db-creds/entries',
      );

      expect(response.status).toBe(200);
      expect(response.body).toEqual(mockEntryResponse);
      expect(secretService.getSecretEntries).toHaveBeenCalledWith(
        'my-service',
        'db-creds',
        'default',
      );
    });
  });

  describe('PUT /secrets/:namespace/:serviceName/:secretName/entries/:key', () => {
    it('should upsert a single secret entry', async () => {
      secretService.upsertSecretEntry.mockResolvedValue(undefined as any);

      const response = await request(app)
        .put('/secrets/default/my-service/db-creds/entries/db-password')
        .send({ value: 'new-secure-password' });

      expect(response.status).toBe(200);
      expect(secretService.upsertSecretEntry).toHaveBeenCalledWith({
        serviceName: 'my-service',
        namespace: 'default',
        secretName: 'db-creds',
        key: 'db-password',
        value: 'new-secure-password',
      });
    });

    it('should fail validation if body does not match entry schema', async () => {
      const response = await request(app)
        .put('/secrets/default/my-service/db-creds/entries/db-password')
        .send({ wrongKey: 'some-value' });

      expect(response.status).toBe(400);
      expect(secretService.upsertSecretEntry).not.toHaveBeenCalled();
    });
  });

  describe('DELETE /secrets/:namespace/:serviceName/:secretName/entries/:key', () => {
    it('should remove a single entry from a secret', async () => {
      secretService.deleteSecretEntry.mockResolvedValue(undefined as any);

      const response = await request(app).delete(
        '/secrets/default/my-service/db-creds/entries/db-username',
      );

      expect(response.status).toBe(204);
      expect(secretService.deleteSecretEntry).toHaveBeenCalledWith({
        serviceName: 'my-service',
        namespace: 'default',
        secretName: 'db-creds',
        key: 'db-username',
      });
    });
  });
});
