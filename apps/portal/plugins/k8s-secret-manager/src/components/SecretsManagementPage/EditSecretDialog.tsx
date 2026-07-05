import {
  Dialog,
  DialogBody,
  DialogFooter,
  DialogHeader,
  Flex,
  TextField,
  Button,
} from '@backstage/ui';
import { type SecretTableRow } from './types';

type EditSecretDialogProps = {
  isOpen: boolean;
  onOpenChange: (nextOpen: boolean) => void;
  item: SecretTableRow;
  secretKey: string;
  onSecretKeyChange: (next: string) => void;
  secretValue: string;
  onSecretValueChange: (next: string) => void;
  isSaving: boolean;
  onSave: () => void;
};

export const EditSecretDialog = ({
  isOpen,
  onOpenChange,
  item,
  secretKey,
  onSecretKeyChange,
  secretValue,
  onSecretValueChange,
  isSaving,
  onSave,
}: EditSecretDialogProps) => (
  <Dialog isOpen={isOpen} onOpenChange={onOpenChange}>
    <DialogHeader>Edit Secret</DialogHeader>
    <DialogBody>
      <Flex direction="column" gap="3">
        <TextField label="Secret Name" value={item.name} isReadOnly />
        <TextField
          label="Key"
          value={secretKey}
          onChange={value => onSecretKeyChange(value)}
        />
        <TextField
          label="New Value"
          value={secretValue}
          onChange={value => onSecretValueChange(value)}
        />
      </Flex>
    </DialogBody>
    <DialogFooter>
      <Button variant="secondary" onClick={() => onOpenChange(false)}>
        Cancel
      </Button>
      <Button variant="primary" onClick={onSave} isDisabled={isSaving}>
        {isSaving ? 'Saving...' : 'Save'}
      </Button>
    </DialogFooter>
  </Dialog>
);
