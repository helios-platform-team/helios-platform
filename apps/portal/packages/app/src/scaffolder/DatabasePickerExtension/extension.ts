import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { DatabasePicker } from './DatabasePicker';

export const DatabasePickerExtension: any = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'DatabasePicker',
    component: DatabasePicker,
    validation: (value: any, validation: any) => {
      // Custom validation: If postgres is selected, dbName MUST be provided
      if (value?.dbType === 'postgres') {
        if (!value?.dbName) {
          validation.addError('Database Name is required when PostgreSQL is selected');
        }
      }
    },
  }),
);