import React, { useState } from 'react';
import { useApi } from '@backstage/core-plugin-api';
import useAsyncRetry from 'react-use/lib/useAsyncRetry';
import { k8sSecretManagerApiRef } from '../../api';
import { SecretTable } from './SecretsTable';
import toast from 'react-hot-toast';
import { useEntity } from '@backstage/plugin-catalog-react';
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

export const SecretListPage = () => {
  const { entity } = useEntity();
  const api = useApi(k8sSecretManagerApiRef);

  // Form states
  const [secretName, setSecretName] = useState('');
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const [secretToDelete, setSecretToDelete] = useState<string | null>(null);
  const [openDeleteDialog, setOpenDeleteDialog] = useState(false);

  // Pagination states
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [tokenHistory, setTokenHistory] = useState<string[]>(['']);

  const namespace = 'default';
  const serviceName = entity.metadata.name ?? 'my-service';

  const {
    value: response,
    loading,
    retry,
  } = useAsyncRetry(async () => {
    const continueToken = tokenHistory[page] || undefined;
    return await api.listSecrets(
      serviceName,
      namespace,
      rowsPerPage,
      continueToken,
    );
  }, [api, namespace, serviceName, rowsPerPage, page]);

  const handleCreateSecret = async () => {
    const trimmedSecretName = secretName.trim();
    if (!trimmedSecretName) return;

    const createPromise = api.createSecret({
      serviceName,
      namespace,
      secretName: trimmedSecretName,
    });

    toast.promise(createPromise, {
      loading: 'Creating secret...',
      success: `Secret "${trimmedSecretName}" created successfully!`,
      error: err =>
        err instanceof Error ? err.message : 'Failed to create secret',
    });

    try {
      await createPromise;
      setSecretName('');
      setIsDialogOpen(false);
      setPage(0);
      setTokenHistory(['']);
      retry();
    } catch (e) {
      console.error('Error creating secret:', e);
    }
  };

  const handleDeleteSecret = (secretNameToDelete: string) => {
    setSecretToDelete(secretNameToDelete);
    setOpenDeleteDialog(true);
  };

  const handleConfirmDelete = async () => {
    if (!secretToDelete) return;

    setOpenDeleteDialog(false);

    const deletePromise = api.deleteSecret(
      serviceName,
      secretToDelete,
      namespace,
    );

    toast.promise(deletePromise, {
      loading: `Deleting secret "${secretToDelete}"...`,
      success: `Secret "${secretToDelete}" deleted successfully!`,
      error: err =>
        err instanceof Error ? err.message : 'Failed to delete secret',
    });

    try {
      await deletePromise;
      retry();
    } catch (e) {
      console.error('Error deleting secret:', e);
    } finally {
      setSecretToDelete(null); // Reset state
    }
  };

  const handleChangePage = (
    _event: React.MouseEvent<HTMLButtonElement> | null,
    newPage: number,
  ) => {
    if (newPage > page && response?.nextPageToken) {
      setTokenHistory(prev => {
        const next = [...prev];
        next[newPage] = response.nextPageToken!;
        return next;
      });
    }
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
    setTokenHistory(['']);
  };

  return (
    <Stack gap={4}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <h2>Environment Variables</h2>
        <Button
          variant="outlined"
          disabled={loading}
          sx={{ width: '190px' }}
          onClick={() => setIsDialogOpen(true)}
        >
          Create New Secret
        </Button>
      </Box>

      <Dialog
        open={isDialogOpen}
        onClose={() => setIsDialogOpen(false)}
        PaperProps={{
          component: 'form',
          onSubmit: (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault();
            setIsDialogOpen(false);
          },
        }}
      >
        <DialogTitle>Create new Secret</DialogTitle>
        <DialogContent>
          <DialogContentText>
            To create a new secret, please enter the details below.
          </DialogContentText>
          <TextField
            required
            margin="dense"
            id="secretName"
            name="secretName"
            label="Secret Name"
            type="text"
            fullWidth
            variant="standard"
            value={secretName}
            onChange={e => setSecretName(e.target.value)}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsDialogOpen(false)}>Cancel</Button>
          <Button type="submit" onClick={handleCreateSecret}>
            Create Secret
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={openDeleteDialog}
        onClose={() => setOpenDeleteDialog(false)}
      >
        <DialogTitle>Confirm Deletion</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete the secret{' '}
            <strong>{secretToDelete}</strong>? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDeleteDialog(false)} color="inherit">
            Cancel
          </Button>
          <Button
            onClick={handleConfirmDelete}
            color="error"
            variant="contained"
          >
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <SecretTable
        data={response?.items || []}
        loading={loading}
        page={page}
        rowsPerPage={rowsPerPage}
        hasNextPage={!!response?.nextPageToken}
        onPageChange={handleChangePage}
        onRowsPerPageChange={handleChangeRowsPerPage}
        onDeleteSecret={handleDeleteSecret}
      />
    </Stack>
  );
};
