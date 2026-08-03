import React, { useEffect } from 'react';
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
  schema, // <--- Add schema here so we can read the YAML
}: FieldExtensionComponentProps<string>) => {
  useEffect(() => {
    if (!formData) {
      onChange(generateName());
    }
  }, []); // Only run once on mount

  return (
    <TextField
      id={idSchema?.$id}
      label={schema?.title || 'Name'} // Now reads the title from YAML
      value={formData || ''}
      onChange={e => onChange(e.target.value)}
      fullWidth
      margin="normal"
      required={required}
      helperText={schema?.description || 'Unique name of the component'} // Now reads the description from YAML
      error={rawErrors?.length > 0}
    />
  );
};
