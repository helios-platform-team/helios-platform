import type { SecretDto } from '../../api';

export type SecretTableRow = SecretDto & {
  id: string;
  actions: string;
};

export type SecretActionsCellProps = {
  item: SecretTableRow;
  serviceName: string;
  onRefresh: () => void;
};
