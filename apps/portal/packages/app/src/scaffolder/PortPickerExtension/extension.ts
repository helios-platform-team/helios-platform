import { scaffolderPlugin } from '@backstage/plugin-scaffolder';
import { createScaffolderFieldExtension } from '@backstage/plugin-scaffolder-react';
import { catalogApiRef } from '@backstage/plugin-catalog-react';
import { PortPicker } from './PortPicker';

const RESERVED_PORTS = [
  3000, // Backstage frontend
  3030, // Gitea
  7007, // Backstage backend
  8001, // Taskfile dev port
  8080, // ArgoCD api / default dev api
  8081, // Health check / metrics port
];

export const PortPickerExtension: any = scaffolderPlugin.provide(
  createScaffolderFieldExtension({
    name: 'PortPicker',
    component: PortPicker,
    validation: async (
      value: number | undefined,
      validation: any,
      { apiHolder }: any,
    ) => {
      if (!value) {
        return;
      }

      if (RESERVED_PORTS.includes(value)) {
        validation.addError(
          `Port ${value} is reserved for Helios infrastructure core services.`,
        );
        return;
      }

      try {
        const catalogApi = apiHolder.get(catalogApiRef);
        if (catalogApi) {
          const response = await catalogApi.getEntities({
            filter: {
              kind: 'component',
            },
          });

          const usedPorts = new Set<number>();
          response.items.forEach((entity: any) => {
            const portStr = entity.metadata.annotations?.['helios.io/port'];
            if (portStr) {
              const p = parseInt(portStr, 10);
              if (!isNaN(p)) {
                usedPorts.add(p);
              }
            }
          });

          if (usedPorts.has(value)) {
            validation.addError(
              `Port ${value} is already in use by another component.`,
            );
          }
        }
      } catch (error) {
        console.error('Failed to validate port uniqueness:', error);
      }
    },
  }),
);
