import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Stack,
  TextField,
} from '@mui/material';
import { useState } from 'react';
import ArrowBackIosNewOutlinedIcon from '@mui/icons-material/ArrowBackIosNewOutlined';
import { useLocation, useNavigate } from 'react-router-dom';
import { SecretEntriesTable } from './SecretEntriesTable.tsx';
import { k8sSecretManagerApiRef } from '../../api.ts';
import { useApi } from '@backstage/core-plugin-api';
import { useEntity } from '@backstage/plugin-catalog-react';
import useAsyncRetry from 'react-use/lib/useAsyncRetry';
import LinearProgress from '@mui/material/LinearProgress';
import toast from 'react-hot-toast';

export const SecretEntriesPage = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const api = useApi(k8sSecretManagerApiRef);

  // Attempt to get the entity if used inside the Catalog, fallback if standalone
  const { entity } = useEntity();
  const namespace = 'default';
  const serviceName = entity?.metadata?.name ?? 'my-service';

  // Extract secret name from URL
  const path = location.pathname;
  const secretName = path.substring(path.lastIndexOf('/') + 1);

  // States for Add Dialog
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');

  const {
    value: entries,
    loading,
    retry,
  } = useAsyncRetry(async () => {
    return await api.getSecretEntries(serviceName, secretName, namespace);
  }, [api, serviceName, secretName, namespace]);

  const handleAddVariable = async () => {
    const trimmedKey = newKey.trim();
    if (!trimmedKey) {
      toast.error('Key is required!');
      return;
    }

    setIsSubmitting(true);
    try {
      await api.upsertSecretEntry(
        serviceName,
        namespace,
        secretName,
        trimmedKey,
        newValue,
      );

      toast.success(`Variable "${trimmedKey}" added`, { duration: 4000 });

      setNewKey('');
      setNewValue('');
      setIsDialogOpen(false);
      retry(); // Refresh the table
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to add variable';
      toast.error(message, { duration: 5000 });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleUpdateVariable = async (key: string, value: string) => {
    try {
      await api.upsertSecretEntry(
        serviceName,
        namespace,
        secretName,
        key,
        value,
      );
      toast.success(`Variable "${key}" updated`, { duration: 4000 });
      retry();
    } catch (e) {
      const message =
        e instanceof Error ? e.message : 'Failed to update variable';
      toast.error(message, { duration: 5000 });
    }
  };

  const handleDeleteVariable = async (key: string) => {
    try {
      await api.deleteSecretEntry(serviceName, namespace, secretName, key);
      toast.success(`Variable "${key}" deleted`, { duration: 4000 });
      retry();
    } catch (e) {
      const message =
        e instanceof Error ? e.message : 'Failed to delete variable';
      toast.error(message, { duration: 5000 });
    }
  };

  return (
    <Stack gap={4}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <Button
          startIcon={<ArrowBackIosNewOutlinedIcon />}
          sx={{
            width: 'fit-content',
          }}
          onClick={() => navigate(-1)}
        >
          Back
        </Button>
        <h2>{secretName}</h2>
        <Button
          variant="outlined"
          disabled={loading}
          sx={{
            width: '190px',
          }}
          onClick={() => setIsDialogOpen(true)}
        >
          Add new variable
        </Button>
      </Box>

      {loading && !entries ? (
        <LinearProgress />
      ) : (
        <SecretEntriesTable
          entries={entries || {}}
          onUpdate={handleUpdateVariable}
          onDelete={handleDeleteVariable}
        />
      )}

      {/* Add Variable Dialog */}
      <Dialog open={isDialogOpen} onClose={() => setIsDialogOpen(false)}>
        <DialogTitle>Add New Variable</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Add a new key-value pair to <strong>{secretName}</strong>.
          </DialogContentText>
          <Stack spacing={2} sx={{ mt: 2 }}>
            <TextField
              required
              label="Key"
              variant="standard"
              fullWidth
              value={newKey}
              onChange={e => setNewKey(e.target.value)}
            />
            <TextField
              label="Value"
              variant="standard"
              fullWidth
              value={newValue}
              onChange={e => setNewValue(e.target.value)}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setIsDialogOpen(false)}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button onClick={handleAddVariable} disabled={isSubmitting}>
            {isSubmitting ? 'Saving...' : 'Save'}
          </Button>
        </DialogActions>
      </Dialog>
    </Stack>
  );
};
