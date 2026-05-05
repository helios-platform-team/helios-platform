import {
  Table,
  useTable,
  CellText,
  Cell,
  type ColumnConfig,
  ButtonIcon,
  MenuTrigger,
  Menu,
  MenuItem,
  DialogTrigger,
  Dialog,
  DialogHeader,
  DialogBody,
  DialogFooter,
  TextField,
  Flex,
  Button,
} from '@backstage/ui';
import { alertApiRef, useApi } from '@backstage/core-plugin-api';
import { Secret, k8sSecretManagerApiRef } from '../../api';
import { RiMore2Line, RiEditLine, RiDeleteBinLine } from '@remixicon/react';
import { useMemo, useState } from 'react';

type SecretTableProps = {
  data?: Secret[];
  serviceName: string;
  onRefresh: () => void;
};

type SecretTableRow = Secret & {
  id: string;
  actions: string;
};

type SecretActionsCellProps = {
  item: SecretTableRow;
  serviceName: string;
  onRefresh: () => void;
};

const SecretActionsCell = ({
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
      alertApi.post({ message: `Secret "${item.name}" updated`, severity: 'info' });
      onRefresh();
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to update secret';
      alertApi.post({ message, severity: 'error' });
    } finally {
      setIsUpdating(false);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      await api.deleteSecret(item.namespace, serviceName, item.name);
      alertApi.post({ message: `Secret "${item.name}" deleted`, severity: 'info' });
      onRefresh();
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to delete secret';
      alertApi.post({ message, severity: 'error' });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <MenuTrigger>
      <ButtonIcon icon={<RiMore2Line />} variant="tertiary" aria-label="More actions" />
      <Menu>
        <DialogTrigger>
          <MenuItem iconStart={<RiEditLine />} color="primary">
            Edit
          </MenuItem>
          <Dialog>
            <DialogHeader>Edit Secret</DialogHeader>
            <DialogBody>
              <Flex direction="column" gap="3">
                <TextField label="Secret Name" value={item.name} isReadOnly />
                <TextField
                  label="Key"
                  value={secretKey}
                  onChange={value => setSecretKey(value)}
                />
                <TextField
                  label="New Value"
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
                onClick={handleEdit}
                isDisabled={isUpdating}
                slot="close"
              >
                {isUpdating ? 'Saving...' : 'Save'}
              </Button>
            </DialogFooter>
          </Dialog>
        </DialogTrigger>

        <DialogTrigger>
          <MenuItem iconStart={<RiDeleteBinLine />} color="danger">
            Delete
          </MenuItem>
          <Dialog>
            <DialogHeader>Delete Secret</DialogHeader>
            <DialogBody>
              Are you sure you want to delete secret &quot;{item.name}&quot;?
            </DialogBody>
            <DialogFooter>
              <Button variant="secondary" slot="close">
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={handleDelete}
                isDisabled={isDeleting}
                slot="close"
              >
                {isDeleting ? 'Deleting...' : 'Delete'}
              </Button>
            </DialogFooter>
          </Dialog>
        </DialogTrigger>
      </Menu>
    </MenuTrigger>
  );
};

const getColumns = (
  serviceName: string,
  onRefresh: () => void,
): ColumnConfig<SecretTableRow>[] => [
  {
    id: 'name',
    label: 'Name',
    isRowHeader: true,
    isSortable: true,
    cell: item => <CellText title={item.name} />,
  },
  {
    id: 'namespace',
    label: 'Namespace',
    cell: item => <CellText title={item.namespace} />,
  },
  {
    id: 'createdAt',
    label: 'Created At',
    cell: item => (
      <CellText title={new Date(item.createdAt ?? '').toLocaleDateString()} />
    ),
  },
  {
    id: 'actions',
    label: '',
    cell: item => (
      <Cell>
        <SecretActionsCell
          item={item}
          serviceName={serviceName}
          onRefresh={onRefresh}
        />
      </Cell>
    ),
  },
];

export const SecretTable = ({ data, serviceName, onRefresh }: SecretTableProps) => {
  const rows: SecretTableRow[] = useMemo(
    () =>
      (data ?? []).map(secret => ({
        ...secret,
        id: `${secret.namespace}:${secret.name}`,
        actions: '',
      })),
    [data],
  );
  const columns = useMemo(
    () => getColumns(serviceName, onRefresh),
    [serviceName, onRefresh],
  );

  const { tableProps } = useTable({
    mode: 'complete',
    data: rows,
  });

  return (
    <Table
      columnConfig={columns}
      {...tableProps}
      emptyState="No secrets created for this service."
    />
  );
};
