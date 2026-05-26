import { ButtonIcon, Menu, MenuItem, MenuTrigger } from '@backstage/ui';
import { alertApiRef, useApi } from '@backstage/core-plugin-api';
import { RiDeleteBinLine, RiEditLine, RiMore2Line } from '@remixicon/react';
import { useState } from 'react';
import { k8sSecretManagerApiRef } from '../../api';
import { EditSecretDialog } from './EditSecretDialog';
import { DeleteSecretDialog } from './DeleteSecretDialog';
import type { SecretActionsCellProps } from './types';

export const SecretActionsCell = ({
  item,
  serviceName,
  onRefresh,
}: SecretActionsCellProps) => {
  const api = useApi(k8sSecretManagerApiRef);
  const alertApi = useApi(alertApiRef);

  const [secretKey, setSecretKey] = useState('value');
  const [secretValue, setSecretValue] = useState('');
  const [isUpdating, setIsUpdating] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  const handleEdit = async () => {
    const trimmedSecretKey = secretKey.trim();
    if (!trimmedSecretKey) {
      alertApi.post({ message: 'Secret key is required', severity: 'error' });
      return;
    }

    setIsUpdating(true);
    try {
      await api.createSecret({
        serviceName,
        namespace: item.namespace,
        secretName: item.name,
        secretData: { [trimmedSecretKey]: secretValue },
      });

      alertApi.post({
        message: `Secret "${item.name}" updated`,
        severity: 'success',
        display: 'transient',
      });

      setIsEditDialogOpen(false);
      onRefresh();
    } catch (e) {
      const message =
        e instanceof Error ? e.message : 'Failed to update secret';
      alertApi.post({ message, severity: 'error', display: 'transient' });
    } finally {
      setIsUpdating(false);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await api.deleteSecret(item.namespace, serviceName, item.name);

      alertApi.post({
        message: `Secret "${item.name}" deleted`,
        severity: 'success',
        display: 'transient',
      });

      setIsDeleteDialogOpen(false);
      onRefresh();
    } catch (e) {
      const message =
        e instanceof Error ? e.message : 'Failed to delete secret';
      alertApi.post({ message, severity: 'error', display: 'transient' });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <>
      <MenuTrigger>
        <ButtonIcon
          icon={<RiMore2Line />}
          variant="tertiary"
          aria-label="More actions"
        />
        <Menu>
          <MenuItem
            iconStart={<RiEditLine />}
            color="primary"
            onAction={() => setIsEditDialogOpen(true)}
          >
            Edit
          </MenuItem>
          <MenuItem
            iconStart={<RiDeleteBinLine />}
            color="danger"
            onAction={() => setIsDeleteDialogOpen(true)}
          >
            Delete
          </MenuItem>
        </Menu>
      </MenuTrigger>

      <EditSecretDialog
        isOpen={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
        item={item}
        secretKey={secretKey}
        onSecretKeyChange={setSecretKey}
        secretValue={secretValue}
        onSecretValueChange={setSecretValue}
        isSaving={isUpdating}
        onSave={handleEdit}
      />

      <DeleteSecretDialog
        isOpen={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        item={item}
        isDeleting={isDeleting}
        onDelete={handleDelete}
      />
    </>
  );
};
