import * as k8s from '@kubernetes/client-node';
import { K8sSecretServiceImpl } from './K8sSecretService';

const mockCoreV1Api = {
  listNamespacedSecret: jest.fn(),
  readNamespacedSecret: jest.fn(),
  replaceNamespacedSecret: jest.fn(),
  createNamespacedSecret: jest.fn(),
  deleteNamespacedSecret: jest.fn(),
};

const mockCustomObjectsApi = {
  getNamespacedCustomObject: jest.fn(),
  replaceNamespacedCustomObject: jest.fn(),
};

jest.mock('@kubernetes/client-node', () => {
  const actual = jest.requireActual('@kubernetes/client-node');
  return {
    ...actual,
    KubeConfig: class {
      loadFromDefault() {}
      makeApiClient(Client: unknown) {
        if (Client === actual.CoreV1Api) {
          return mockCoreV1Api;
        }
        if (Client === actual.CustomObjectsApi) {
          return mockCustomObjectsApi;
        }
        throw new Error('unexpected api client type');
      }
    },
  };
});

describe('K8sSecretServiceImpl', () => {
  const logger = {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
  } as any;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('creates secret and patches external-secret-reference trait on target component', async () => {
    const svc = new K8sSecretServiceImpl(logger);
    mockCoreV1Api.readNamespacedSecret.mockRejectedValue(
      new k8s.ApiException(
        404,
        'Not Found',
        JSON.stringify({ code: 404 }),
        {},
      ),
    );
    mockCoreV1Api.createNamespacedSecret.mockResolvedValue({
      metadata: {
        name: 'my-service-db',
        namespace: 'default',
        creationTimestamp: new Date('2020-01-01'),
      },
    });
    mockCustomObjectsApi.getNamespacedCustomObject.mockResolvedValue({
      spec: {
        components: [
          { name: 'web', traits: [] },
          {
            name: 'my-service',
            traits: [
              { type: 'external-secret-reference', properties: { secretName: 'old' } },
            ],
          },
        ],
      },
    });
    mockCustomObjectsApi.replaceNamespacedCustomObject.mockResolvedValue({});

    const result = await svc.createSecret(
      {
        serviceName: 'my-service',
        secretName: 'db',
        namespace: 'default',
        secretData: { DB_USER: 'u' },
      },
      {
        credentials: {
          principal: { userEntityRef: 'user:default/alice' },
        } as any,
      },
    );

    expect(result.name).toBe('my-service-db');
    expect(mockCoreV1Api.createNamespacedSecret).toHaveBeenCalledTimes(1);
    expect(mockCustomObjectsApi.replaceNamespacedCustomObject).toHaveBeenCalledTimes(
      1,
    );
    const [{ body }] = mockCustomObjectsApi.replaceNamespacedCustomObject.mock.calls[0];
    const webComp = body.spec.components.find((c: any) => c.name === 'web');
    const serviceComp = body.spec.components.find(
      (c: any) => c.name === 'my-service',
    );
    expect(webComp.traits).toEqual([]);
    expect(serviceComp.traits).toEqual([
      {
        type: 'external-secret-reference',
        properties: { secretName: 'my-service-db' },
      },
    ]);
    expect(logger.info).toHaveBeenCalledWith(
      expect.stringContaining('Operator reconcile will render with CUE'),
    );
  });

  it('skips HeliosApp update when service component does not exist', async () => {
    const svc = new K8sSecretServiceImpl(logger);
    mockCoreV1Api.readNamespacedSecret.mockRejectedValue(
      new k8s.ApiException(
        404,
        'Not Found',
        JSON.stringify({ code: 404 }),
        {},
      ),
    );
    mockCoreV1Api.createNamespacedSecret.mockResolvedValue({
      metadata: {
        name: 'my-service-db',
        namespace: 'default',
      },
    });
    mockCustomObjectsApi.getNamespacedCustomObject.mockResolvedValue({
      spec: { components: [{ name: 'api', traits: [] }] },
    });

    await svc.createSecret(
      {
        serviceName: 'my-service',
        secretName: 'db',
        namespace: 'default',
        secretData: { DB_USER: 'u' },
      },
      {
        credentials: {
          principal: { userEntityRef: 'user:default/alice' },
        } as any,
      },
    );

    expect(mockCustomObjectsApi.replaceNamespacedCustomObject).not.toHaveBeenCalled();
    expect(logger.warn).toHaveBeenCalledWith(
      'Skipping HeliosApp trait update for my-service: component my-service not found',
    );
  });
});
