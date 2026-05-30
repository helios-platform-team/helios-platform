import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { TextField } from '@material-ui/core';

export const UniqueNamePicker = ({
  onChange,
  rawErrors,
  required,
  formData,
  idSchema,
}: FieldExtensionComponentProps<string>) => {
  return (
    <TextField
      id={idSchema?.$id}
      label="Name"
      value={formData || ''}
      onChange={e => onChange(e.target.value)}
      fullWidth
      margin="normal"
      required={required}
      helperText="Unique name of the component (e.g. my-service)"
      error={rawErrors?.length > 0}
    />
  );
};
