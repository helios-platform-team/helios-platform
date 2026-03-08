import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { FormControl, InputLabel, Select, MenuItem, TextField } from '@material-ui/core';

// Define the exact state structure expected by PhuocHoan's CUE schema
interface DatabaseConfig {
  dbType: string;
  dbName?: string;
}

export const DatabasePicker = ({
  onChange,
  rawErrors,
  required,
  formData,
}: FieldExtensionComponentProps<DatabaseConfig>) => {
  // Default to 'none' if no data is present
  const dbType = formData?.dbType || 'none';
  const dbName = formData?.dbName || '';

  const handleTypeChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    const newType = event.target.value as string;
    onChange({
      dbType: newType,
      // Clear the dbName if they switch back to "No Database"
      ...(newType === 'postgres' ? { dbName: dbName } : { dbName: '' }),
    });
  };

  const handleNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({
      dbType,
      dbName: event.target.value,
    });
  };

  return (
    <div>
      <FormControl fullWidth required={required} margin="normal">
        <InputLabel>Database Type</InputLabel>
        <Select value={dbType} onChange={handleTypeChange}>
          <MenuItem value="none">No Database</MenuItem>
          <MenuItem value="postgres">PostgreSQL</MenuItem>
        </Select>
      </FormControl>

      {dbType === 'postgres' && (
        <TextField
          label="Database Name"
          value={dbName}
          onChange={handleNameChange}
          fullWidth
          margin="normal"
          required
          helperText="The name of the database to create (e.g., my_custom_db)"
          error={rawErrors?.length > 0 && !dbName}
        />
      )}
    </div>
  );
};