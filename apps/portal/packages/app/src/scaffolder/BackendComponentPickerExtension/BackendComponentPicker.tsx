import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { useApi } from '@backstage/core-plugin-api';
import { catalogApiRef } from '@backstage/plugin-catalog-react';
import useAsync from 'react-use/lib/useAsync';
import { TextField } from '@material-ui/core';
import Autocomplete from '@material-ui/lab/Autocomplete';

const DEFAULT_BACKEND_COMPONENTS = [
  'nodejs-backend',
  'nestjs-prisma-backend',
  'dotnet-backend',
  'springboot-backend',
  'postgrest-backend',
  'hasura-backend',
  'postgraphile-backend',
];

export const BackendComponentPicker = ({
  onChange,
  rawErrors,
  required,
  formData,
  idSchema,
  uiSchema,
}: FieldExtensionComponentProps<string>) => {
  const catalogApi = useApi(catalogApiRef);

  const { value: catalogEntities, loading } = useAsync(async () => {
    try {
      const response = await catalogApi.getEntities({
        filter: {
          kind: 'component',
          'spec.type': 'service',
        },
      });
      return response.items;
    } catch (e) {
      console.error('Failed to fetch backend components from catalog', e);
      return [];
    }
  }, [catalogApi]);

  const catalogOptions = React.useMemo(() => {
    if (!catalogEntities) return [];
    return catalogEntities.map(entity => entity.metadata.name);
  }, [catalogEntities]);

  // Merge default options with what's actually in the catalog, deduplicated
  const allOptions = React.useMemo(() => {
    const set = new Set([...catalogOptions, ...DEFAULT_BACKEND_COMPONENTS]);
    return Array.from(set);
  }, [catalogOptions]);

  const value = formData || '';
  const allowCustom = uiSchema?.['ui:options']?.allowCustom !== false;

  return (
    <Autocomplete
      id={idSchema?.$id}
      freeSolo={allowCustom}
      options={allOptions}
      loading={loading}
      value={value}
      onChange={(_, newValue) => {
        onChange(newValue || '');
      }}
      onInputChange={
        allowCustom
          ? (_, newInputValue) => {
              onChange(newInputValue || '');
            }
          : undefined
      }
      renderInput={params => (
        <TextField
          {...params}
          label="Backend Component"
          variant="standard"
          required={required}
          error={rawErrors?.length > 0}
          helperText={
            allowCustom
              ? 'Select an existing backend component from the catalog, choose a template-based backend default, or type a custom name.'
              : 'Select an existing backend component from the catalog or choose a template-based backend default.'
          }
          fullWidth
          margin="normal"
        />
      )}
    />
  );
};
