import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { DatabasePicker, DatabaseFormData } from './DatabasePicker';

/**
 * DatabasePickerFieldExtension
 * 
 * Register the DatabasePicker as a custom field extension.
 * 
 * Usage in template.yaml:
 *   properties:
 *     database:
 *       title: Database Configuration
 *       type: object
 *       ui:field: DatabasePicker
 */
export const DatabasePickerFieldExtension = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'DatabasePicker',
    component: DatabasePicker,
    validation: (value: DatabaseFormData | undefined, validation) => {
      // Skip validation if no value or type is 'none'
      if (!value?.type || value.type === 'none') {
        return;
      }

      // Validate PostgreSQL fields
      if (value.type === 'postgres') {
        if (!value.host) {
          validation.addError('Host is required for PostgreSQL');
        }
        if (!value.password) {
          validation.addError('Password is required for PostgreSQL');
        }
      }

      // Validate MongoDB fields
      if (value.type === 'mongodb') {
        if (!value.connectionString) {
          validation.addError('Connection string is required for MongoDB');
        }
        // Validate connection string format
        if (value.connectionString && !value.connectionString.startsWith('mongodb://')) {
          validation.addError('Connection string must start with mongodb://');
        }
      }
    },
  }),
);
