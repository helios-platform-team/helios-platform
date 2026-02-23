import React from 'react';
import {
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  FormHelperText,
  Box,
} from '@material-ui/core';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';

/**
 * DatabasePicker - Custom Field Extension Demo
 * 
 * This component demonstrates how to create a custom field extension
 * that shows/hides fields based on user selection.
 * 
 * Use Case: When user selects a database type, show relevant configuration fields.
 */

export interface DatabaseFormData {
  type: string;
  host?: string;
  port?: number;
  password?: string;
  connectionString?: string;
}

export const DatabasePicker = (
  props: FieldExtensionComponentProps<DatabaseFormData | undefined>,
) => {
  const {
    onChange,
    rawErrors,
    required,
    formData,
  } = props;

  const dbType = formData?.type || '';

  const handleTypeChange = (event: React.ChangeEvent<{ value: unknown }>) => {
    const newType = event.target.value as string;
    
    // Reset fields when type changes
    if (newType === 'postgres') {
      onChange({
        type: newType,
        host: 'localhost',
        port: 5432,
        password: '',
      });
    } else if (newType === 'mongodb') {
      onChange({
        type: newType,
        connectionString: '',
      });
    } else {
      onChange({
        type: newType,
      });
    }
  };

  const handleFieldChange = (field: keyof Omit<DatabaseFormData, 'type'>, value: string | number) => {
    onChange({
      ...formData,
      type: formData?.type || '',
      [field]: value,
    });
  };

  const hasError = rawErrors && rawErrors.length > 0;

  return (
    <Box>
      <FormControl fullWidth required={required} error={hasError} margin="normal">
        <InputLabel id="db-type-label">Database Type</InputLabel>
        <Select
          labelId="db-type-label"
          value={dbType}
          onChange={handleTypeChange}
        >
          <MenuItem value="none">No Database</MenuItem>
          <MenuItem value="postgres">PostgreSQL</MenuItem>
          <MenuItem value="mongodb">MongoDB</MenuItem>
        </Select>
        <FormHelperText>
          Select the database type for your service
        </FormHelperText>
      </FormControl>

      {dbType === 'postgres' && (
        <Box mt={2}>
          <TextField
            label="Host"
            value={formData?.host || ''}
            onChange={(e) => handleFieldChange('host', e.target.value)}
            fullWidth
            margin="normal"
            required
            helperText="Database server hostname"
          />
          <TextField
            label="Port"
            type="number"
            value={formData?.port || 5432}
            onChange={(e) => handleFieldChange('port', parseInt(e.target.value, 10))}
            fullWidth
            margin="normal"
            helperText="Database server port (default: 5432)"
          />
          <TextField
            label="Password"
            type="password"
            value={formData?.password || ''}
            onChange={(e) => handleFieldChange('password', e.target.value)}
            fullWidth
            margin="normal"
            required
            helperText="Database password"
          />
        </Box>
      )}

      {dbType === 'mongodb' && (
        <Box mt={2}>
          <TextField
            label="Connection String"
            value={formData?.connectionString || ''}
            onChange={(e) => handleFieldChange('connectionString', e.target.value)}
            fullWidth
            margin="normal"
            required
            placeholder="mongodb://user:pass@host:27017/database"
            helperText="MongoDB connection URI"
          />
        </Box>
      )}

      {dbType === 'none' && (
        <Box mt={2}>
          <FormHelperText>
            No database configuration needed.
          </FormHelperText>
        </Box>
      )}
    </Box>
  );
};
