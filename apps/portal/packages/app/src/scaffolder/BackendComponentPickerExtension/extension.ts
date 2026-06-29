import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { BackendComponentPicker } from './BackendComponentPicker';

export const BackendComponentPickerExtension: any = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'BackendComponentPicker',
    component: BackendComponentPicker,
  }),
);
