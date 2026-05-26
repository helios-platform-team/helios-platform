import {
  Dialog,
  DialogBody,
  DialogFooter,
  DialogHeader,
  Button,
} from '@backstage/ui';
import { type SecretTableRow } from './types';

type DeleteSecretDialogProps = {
  isOpen: boolean;
  onOpenChange: (nextOpen: boolean) => void;
  item: SecretTableRow;
  isDeleting: boolean;
  onDelete: () => void;
};

export const DeleteSecretDialog = ({
  isOpen,
  onOpenChange,
  item,
  isDeleting,
  onDelete,
}: DeleteSecretDialogProps) => (
  <Dialog isOpen={isOpen} onOpenChange={onOpenChange}>
    <DialogHeader>Delete Secret</DialogHeader>
    <DialogBody>
      Are you sure you want to delete secret &quot;{item.name}&quot;?
    </DialogBody>
    <DialogFooter>
      <Button variant="secondary" onClick={() => onOpenChange(false)}>
        Cancel
      </Button>
      <Button variant="primary" onClick={onDelete} isDisabled={isDeleting}>
        {isDeleting ? 'Deleting...' : 'Delete'}
      </Button>
    </DialogFooter>
  </Dialog>
);
