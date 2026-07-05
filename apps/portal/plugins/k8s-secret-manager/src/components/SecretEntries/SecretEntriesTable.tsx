import Grid from '@mui/material/Grid';
import { SecretEntryItem } from './SecretEntryItem.tsx';
import Typography from '@mui/material/Typography';

interface SecretEntriesTableProps {
  entries: Record<string, string>;
  onUpdate: (key: string, value: string) => Promise<void>;
  onDelete: (key: string) => Promise<void>;
}

export const SecretEntriesTable = ({
  entries,
  onUpdate,
  onDelete,
}: SecretEntriesTableProps) => {
  const keys = Object.keys(entries);

  if (keys.length === 0) {
    return (
      <Typography sx={{ mt: 2, color: 'text.secondary' }}>
        No variables found in this secret.
      </Typography>
    );
  }

  return (
    <div>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '5fr 6fr 1fr',
          gap: '16px',
        }}
      >
        <p style={{ fontSize: '17px', fontWeight: 'bold' }}>Key</p>
        <p style={{ fontSize: '17px', fontWeight: 'bold' }}>Value</p>
        <div />
      </div>

      {keys.map(key => (
        <SecretEntryItem
          key={key}
          entryKey={key}
          entryValue={entries[key]}
          onUpdate={onUpdate}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
};
