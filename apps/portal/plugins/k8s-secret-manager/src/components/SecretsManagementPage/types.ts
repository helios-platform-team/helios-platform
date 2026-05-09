import type { Secret } from '../../api';

export type SecretTableRow = Secret & {
  id: string;
  actions: string;
};

export type SecretActionsCellProps = {
  item: SecretTableRow;
  serviceName: string;
  onRefresh: () => void;
};

