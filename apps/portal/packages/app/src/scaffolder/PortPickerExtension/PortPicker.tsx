import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { useApi } from '@backstage/core-plugin-api';
import { catalogApiRef } from '@backstage/plugin-catalog-react';
import { TextField } from '@material-ui/core';

const RESERVED_PORTS = [3000, 3030, 7007, 8001, 8080, 8081];

export const PortPicker = ({
  onChange,
  rawErrors,
  required,
  formData,
  idSchema,
  schema,
}: FieldExtensionComponentProps<number>) => {
  const catalogApi = useApi(catalogApiRef);

  const hasAutoSelectedRef = React.useRef(false);

  React.useEffect(() => {
    if (hasAutoSelectedRef.current) {
      return;
    }

    let isMounted = true;
    const findFreePort = async () => {
      try {
        const response = await catalogApi.getEntities({
          filter: {
            kind: 'component',
          },
        });

        if (!isMounted) return;

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

        const startPort = schema?.default || 5173;
        const currentVal =
          formData !== undefined && formData !== null ? Number(formData) : 0;

        const isCurrentValUsed =
          usedPorts.has(currentVal) || RESERVED_PORTS.includes(currentVal);
        const isDefaultValUsed =
          usedPorts.has(Number(startPort)) ||
          RESERVED_PORTS.includes(Number(startPort));

        if (
          currentVal === 0 ||
          (currentVal === Number(startPort) && isDefaultValUsed) ||
          isCurrentValUsed
        ) {
          let candidate = Number(startPort);
          while (
            RESERVED_PORTS.includes(candidate) ||
            usedPorts.has(candidate)
          ) {
            candidate += 1;
          }
          onChange(candidate);
        }

        hasAutoSelectedRef.current = true;
      } catch (error) {
        console.error('Failed to auto-select free port:', error);
        hasAutoSelectedRef.current = true;
      }
    };

    findFreePort();
    return () => {
      isMounted = false;
    };
  }, [catalogApi, formData, onChange, schema]);

  return (
    <TextField
      id={idSchema?.$id}
      label="API Port"
      type="number"
      value={formData || ''}
      onChange={e => {
        const val = parseInt(e.target.value, 10);
        onChange(isNaN(val) ? undefined : val);
      }}
      fullWidth
      margin="normal"
      required={required}
      helperText="Standardized platform networking port for web services."
      error={rawErrors && rawErrors.length > 0}
    />
  );
};
