import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { DatabasePicker } from './DatabasePicker';

export const DatabasePickerExtension = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'DatabasePicker',
    component: DatabasePicker,
    validation: (value, validation, context) => {
        // Todo: Implement validation logic
    },
  }),
);
