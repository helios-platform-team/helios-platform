import { k8SSecretManagerPlugin } from './plugin';

describe('k8s-secret-manager', () => {
  it('should export plugin', () => {
    expect(k8SSecretManagerPlugin).toBeDefined();
  });
});
