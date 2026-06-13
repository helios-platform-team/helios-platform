import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { catalogApiRef } from '@backstage/plugin-catalog-react';
import { UniqueNamePicker } from './UniqueNamePicker';

export const UniqueNamePickerExtension: any = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'UniqueNamePicker',
    component: UniqueNamePicker,
    validation: async (
      value: string | undefined,
      validation: any,
      { apiHolder }: any,
    ) => {
      if (!value) {
        return;
      }

      // Check if it's alphanumeric/lowercase/hyphens (standard Backstage entity name rule)
      if (!/^[a-z0-9-]+$/.test(value)) {
        validation.addError(
          'Name must be lowercase and contain only alphanumeric characters and hyphens (no spaces or underscores).',
        );
        return;
      }

      try {
        const catalogApi = apiHolder.get(catalogApiRef);
        if (catalogApi) {
          // Check if an entity with this name already exists in the catalog
          const entity = await catalogApi.getEntityByRef(
            `component:default/${value}`,
          );
          if (entity) {
            validation.addError(
              `A component named "${value}" already exists in the catalog. Please choose a unique name.`,
            );
          }
        }
      } catch (error) {
        // If entity is not found (which is a successful uniqueness validation), or API fails, we let it pass
        console.error('Failed to validate component name uniqueness:', error);
      }
    },
  }),
);
