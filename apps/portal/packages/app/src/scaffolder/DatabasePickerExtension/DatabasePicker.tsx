import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import {
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  FormHelperText,
} from '@material-ui/core';

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
  formContext,
  schema
}: FieldExtensionComponentProps<DatabaseConfig>) => {
  const rootFormData = formContext?.formData || {};
  const componentName = rootFormData.name || '';

  // Default to 'none' if no data is present
  const dbType = formData?.dbType || 'none';
  const dbName = formData?.dbName || '';
  const dbVersion = formData?.dbVersion || '18.4';

  // Keep dbName in sync with componentName by default unless customized by the user
  const prevComponentNameRef = React.useRef(componentName);

  React.useEffect(() => {
    const oldComponent = prevComponentNameRef.current;
    prevComponentNameRef.current = componentName;

    if (dbType === 'postgres') {
      if (!dbName || dbName === oldComponent) {
        onChange({
          dbType,
          dbName: componentName,
          dbVersion,
        });
      }
    }
  }, [componentName, dbType, dbName, dbVersion, onChange]);

  const handleTypeChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    const newType = event.target.value as string;
    onChange({
      dbType: newType,
      ...(newType === 'postgres'
        ? { dbName: dbName || componentName, dbVersion: dbVersion }
        : { dbName: '', dbVersion: '' }),
    });
  };

  const handleNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    onChange({
      dbType,
      dbName: event.target.value,
      dbVersion,
    });
  };

  const handleVersionChange = (
    event: React.ChangeEvent<{ value: unknown }>,
  ) => {
    onChange({
      dbType,
      dbName,
      dbVersion: event.target.value as string,
    });
  };

  return (
    <div>
      <FormControl fullWidth required={required} margin="normal">
        {/* Dynamically render title */}
        <InputLabel>{schema?.title || 'Database Type'}</InputLabel>
        <Select value={dbType} onChange={handleTypeChange}>
          <MenuItem value="none">No Database</MenuItem>
          <MenuItem value="postgres">PostgreSQL</MenuItem>
        </Select>
        {/* Dynamically render description */}
        {schema?.description && (
          <FormHelperText>{schema.description}</FormHelperText>
        )}
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
            helperText="The name of the database to create (by default this matches the component name)."
            error={rawErrors && rawErrors.length > 0 && !dbName}
          />

          <FormControl fullWidth margin="normal" required>
            <InputLabel>Database Version</InputLabel>
            <Select value={dbVersion} onChange={handleVersionChange}>
              <MenuItem value="18">18</MenuItem>
              <MenuItem value="18.4">18.4</MenuItem>
            </Select>
          </FormControl>
        </>
      )}
    </div>
  );
};
