import { useState } from 'react';
import {
  Header,
  Box,
  Button,
  Dialog,
  DialogTrigger,
  DialogHeader,
  DialogBody,
  DialogFooter,
  Flex,
  TextField,
} from '@backstage/ui';
import { useApi } from '@backstage/core-plugin-api';
import useAsyncRetry from 'react-use/lib/useAsyncRetry';
import { k8sSecretManagerApiRef } from '../../api';
import { SecretTable } from './SecretsTable';
import { useEntity } from '@backstage/plugin-catalog-react';

export const SecretsManagementPage = () => {
  const { entity } = useEntity();
  const api = useApi(k8sSecretManagerApiRef);
  const [secretName, setSecretName] = useState('');
  const [secretKey, setSecretKey] = useState('value');
  const [secretValue, setSecretValue] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  // For a standalone page, you might hardcode or provide an input for the namespace/service.
  // If this is used on a Catalog Entity page later, you'd use the useEntity() hook instead.
  const namespace = 'default';
  const serviceName = entity.metadata.name ?? 'my-service';

  const {
    value: secrets,
    loading,
    retry,
  } = useAsyncRetry(async () => {
    return await api.listSecrets(namespace, serviceName);
  }, [api, namespace, serviceName]);

  const handleCreateSecret = async () => {
    const trimmedSecretName = secretName.trim();
    const trimmedSecretKey = secretKey.trim();

    setIsCreating(true);
    try {
      await api.createSecret({
        serviceName,
        namespace,
        secretName: trimmedSecretName,
        secretData: { [trimmedSecretKey]: secretValue },
      });
      setSecretName('');
      setSecretKey('value');
      setSecretValue('');
      retry();
    } catch (e) {
      // throw e;
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Box>
      <Header
        title="Environment Variables"
        customActions={
          <DialogTrigger>
            <Button variant="secondary" isDisabled={loading}>
              Create New Secret
            </Button>
            <Dialog>
              <DialogHeader>Create New Secret</DialogHeader>
              <DialogBody>
                <Flex direction="column" gap="3">
                  <TextField
                    label="Secret Name"
                    placeholder="Enter secret name"
                    value={secretName}
                    onChange={value => setSecretName(value)}
                  />
                  <TextField
                    label="Key"
                    placeholder="Enter key"
                    value={secretKey}
                    onChange={value => setSecretKey(value)}
                  />
                  <TextField
                    label="Value"
                    placeholder="Enter value"
                    value={secretValue}
                    onChange={value => setSecretValue(value)}
                  />
                </Flex>
              </DialogBody>
              <DialogFooter>
                <Button variant="secondary" slot="close">
                  Cancel
                </Button>
                <Button
                  variant="primary"
                  onClick={handleCreateSecret}
                  isDisabled={isCreating}
                >
                  {isCreating ? 'Creating...' : 'Create Secret'}
                </Button>
              </DialogFooter>
            </Dialog>
          </DialogTrigger>
        }
      />

      <SecretTable data={secrets} serviceName={serviceName} onRefresh={retry} />
    </Box>
  );
};
