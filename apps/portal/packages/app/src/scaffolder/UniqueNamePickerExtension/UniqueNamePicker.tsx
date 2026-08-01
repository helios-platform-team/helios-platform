import { useEffect } from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { TextField } from '@material-ui/core';

const NOUNS = [
  'alpha',
  'beta',
  'gamma',
  'delta',
  'echo',
  'falcon',
  'griffin',
  'helix',
  'iris',
  'jazz',
  'koala',
  'lotus',
  'mango',
  'nexus',
  'omega',
  'panda',
  'quantum',
  'raven',
  'sphinx',
  'tiger',
  'umbra',
  'vortex',
  'wolf',
  'xenon',
  'yacht',
  'zenith',
  'app',
  'service',
  'api',
  'core',
  'data',
  'cloud',
  'web',
  'net',
  'sys',
  'node',
  'link',
  'flow',
  'grid',
  'base',
];

const generateName = () => {
  const noun1 = NOUNS[Math.floor(Math.random() * NOUNS.length)];
  const noun2 = NOUNS[Math.floor(Math.random() * NOUNS.length)];
  const num = Math.floor(1000 + Math.random() * 9000); // 4-digit number
  return `${noun1}-${noun2}-${num}`;
};

export const UniqueNamePicker = ({
  onChange,
  rawErrors,
  required,
  formData,
  idSchema,
}: FieldExtensionComponentProps<string>) => {
  useEffect(() => {
    if (!formData) {
      onChange(generateName());
    }
  }, []); // Only run once on mount

  return (
    <TextField
      id={idSchema?.$id}
      label="Name"
      value={formData || ''}
      onChange={e => onChange(e.target.value)}
      fullWidth
      margin="normal"
      required={required}
      helperText="Unique name of the component (e.g. my-service-1234)"
      error={rawErrors?.length > 0}
    />
  );
};
