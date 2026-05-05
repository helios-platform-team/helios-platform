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

describe('createRouter', () => {
  let app: express.Express;
  let secretService: jest.Mocked<typeof k8sSecretServiceRef.T>;

  beforeEach(async () => {
    secretService = {
      listSecrets: jest.fn(),
      createSecret: jest.fn(),
      deleteSecret: jest.fn(),
    } as any; // Cast as any because we don't need to mock private properties

    const router = await createRouter({
      httpAuth: mockServices.httpAuth(),
      secretService,
    });

    app = express();
    app.use(router);
    app.use(mockErrorHandler());
  });

  describe('GET /secrets', () => {
    it('should return a list of secrets', async () => {
      secretService.listSecrets.mockResolvedValue([mockSecretResponse]);

      const response = await request(app).get(
        '/secrets?namespace=default&serviceName=my-service',
      );

      expect(response.status).toBe(200);
      expect(response.body).toEqual([mockSecretResponse]);
      expect(secretService.listSecrets).toHaveBeenCalledWith(
        'my-service',
        'default',
      );
    });
  });

  describe('POST /secrets', () => {
    it('should create a secret successfully', async () => {
      secretService.createSecret.mockResolvedValue(mockSecretResponse);

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

    it('should fail validation if data is missing', async () => {
      const response = await request(app).post('/secrets').send({
        serviceName: 'my-service',
        secretName: 'db-creds',
        namespace: 'default',
        // missing secretData
      });

      expect(response.status).toBe(400); // InputError yields 400
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
      secretService.deleteSecret.mockResolvedValue();

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
});
