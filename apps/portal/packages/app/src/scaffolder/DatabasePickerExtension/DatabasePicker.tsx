import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { FormControl, InputLabel, Select, MenuItem, TextField } from '@material-ui/core';

// Define the exact state structure expected by PhuocHoan's CUE schema
interface DatabaseConfig {
  dbType: string;
  dbName?: string;
  dbVersion?: string;
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
  const dbVersion = formData?.dbVersion || '16';

  const handleTypeChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    const newType = event.target.value as string;
    onChange({
      dbType: newType,
      ...(newType === 'postgres' ? { dbName: dbName, dbVersion: dbVersion } : { dbName: '', dbVersion: '' }),
    });
  };

  const handleNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({
      dbType,
      dbName: event.target.value,
      dbVersion,
    });
  };

  const handleVersionChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    onChange({
      dbType,
      dbName,
      dbVersion: event.target.value as string,
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
        <>
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

          <FormControl fullWidth margin="normal" required>
            <InputLabel>Database Version</InputLabel>
            <Select value={dbVersion} onChange={handleVersionChange}>
              <MenuItem value="13">13</MenuItem>
              <MenuItem value="14">14</MenuItem>
              <MenuItem value="15">15</MenuItem>
              <MenuItem value="16">16</MenuItem>
            </Select>
          </FormControl>
        </>
      )}
    </div>
  );
};