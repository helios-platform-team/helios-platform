import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { Grid, Box, Typography, Divider } from '@material-ui/core';

export const ResponsiveGridField = ({
  formData,
  onChange,
  uiSchema,
  schema,
  idSchema,
  rawErrors,
  registry,
  onFocus,
  onBlur,
}: FieldExtensionComponentProps<any>) => {
  const { SchemaField } = registry.fields;
  
  // Get the properties from the schema
  const properties = schema.properties || {};
  const propertyNames = Object.keys(properties);

  return (
    <Box mb={4}>
      {(schema.title || schema.description) && (
        <Box mb={2}>
          {schema.title && (
            <Typography variant="h6" component="h3" gutterBottom>
              {schema.title}
            </Typography>
          )}
          {schema.description && (
            <Typography variant="body2" color="textSecondary" paragraph>
              {schema.description}
            </Typography>
          )}
          <Divider />
        </Box>
      )}
      
      <Grid container spacing={3}>
        {propertyNames.map(name => {
          const fieldSchema = properties[name];
          const isRequired = Array.isArray(schema.required) 
            ? schema.required.includes(name) 
            : false;
          const fieldTitle = fieldSchema.title || name;
            
          return (
            <Grid item xs={12} sm={6} key={name}>
              <Box display="flex" flexDirection="column">
                <Typography variant="subtitle2" style={{ marginBottom: '8px', fontWeight: 'bold' }}>
                  {fieldTitle} {isRequired && '*'}
                </Typography>
                <SchemaField
                  key={name}
                  name={name}
                  required={isRequired}
                  schema={fieldSchema as any}
                  uiSchema={(uiSchema as any)?.[name]}
                  idSchema={(idSchema as any)?.[name]}
                  formData={(formData as any)?.[name]}
                  onChange={(value: any) =>
                    onChange({ ...formData, [name]: value })
                  }
                  registry={registry}
                  rawErrors={(rawErrors as any)?.[name]}
                  onFocus={onFocus}
                  onBlur={onBlur}
                />
              </Box>
            </Grid>
          );
        })}
      </Grid>
    </Box>
  );
};
