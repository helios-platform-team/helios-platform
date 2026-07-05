import { k8SSecretManagerBackendPlugin } from './plugin';

describe('k8SSecretManagerBackendPlugin', () => {
  it('exports a backend plugin instance', () => {
    expect(k8SSecretManagerBackendPlugin).toBeDefined();
  });
});
