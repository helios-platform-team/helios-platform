import {
  Table,
  useTable,
  CellText,
  Cell,
  type ColumnConfig,
} from '@backstage/ui';
import { Secret } from '../../api';
import { useMemo } from 'react';
import { SecretActionsCell } from './SecretActionsCell';
import type { SecretTableRow } from './types';

type SecretTableProps = {
  data?: Secret[];
  serviceName: string;
  onRefresh: () => void;
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
