import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { DatabasePicker, type DatabaseConfig } from './DatabasePicker';

interface ValidationContext {
  addError: (message: string) => void;
}

export const DatabasePickerExtension = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'DatabasePicker',
    component: DatabasePicker,
    validation: (value: DatabaseConfig | undefined, validation: ValidationContext) => {
      // Custom validation: If postgres is selected, dbName MUST be provided
      if (value?.dbType === 'postgres') {
        const dbName =
          typeof value?.dbName === 'string' ? value.dbName.trim() : '';
        if (!dbName) {
          validation.addError(
            'Database Name is required when PostgreSQL is selected',
          );
        }
      }
    },
  }),
);
