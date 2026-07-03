import React from 'react';
import { FieldExtensionComponentProps } from '@backstage/plugin-scaffolder-react';
import { useApi } from '@backstage/core-plugin-api';
import { catalogApiRef } from '@backstage/plugin-catalog-react';
import useAsync from 'react-use/lib/useAsync';
import {
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
  Divider,
} from '@material-ui/core';
import { useTheme } from '@material-ui/core/styles';
import Autocomplete from '@material-ui/lab/Autocomplete';

// Reserved ports in Helios infrastructure (to prevent collision)
const RESERVED_PORTS = [
  3000, // Backstage frontend
  3030, // Gitea
  7007, // Backstage backend
  8001, // Taskfile dev port
  8080, // ArgoCD api / default dev api
  8081, // Health check / metrics port
];

interface BackendTemplateInfo {
  id: string;
  name: string;
  defaultPort: number;
}

const BACKEND_TEMPLATES: BackendTemplateInfo[] = [
  { id: 'nestjs', name: 'NestJS', defaultPort: 3001 },
  { id: 'nodejs', name: 'Node.js Service', defaultPort: 3002 },
  { id: 'dotnet', name: '.NET Web API', defaultPort: 5000 },
  { id: 'spring-boot', name: 'Spring Boot', defaultPort: 8081 },
  { id: 'postgrest', name: 'PostgREST (Instant REST API)', defaultPort: 3003 },
  { id: 'hasura', name: 'Hasura GraphQL Engine', defaultPort: 8082 },
  { id: 'postgraphile', name: 'PostGraphile (Instant GraphQL API)', defaultPort: 5433 },
];

interface PickerOption {
  type: 'template' | 'existing' | 'none';
  id: string;
  label: string;
}

interface BackendDatabaseConfig {
  dbType: string;
  dbName?: string;
  dbVersion?: string;
}

interface BackendConfigData {
  backendComponent?: string;
  backendType?: string;
  backendPort?: number;
  backendRepoName?: string;
  connectionType?: string;
  backendDatabaseConfig?: BackendDatabaseConfig;
}

// Find next available port not in RESERVED_PORTS and not in usedPorts
function findNextAvailablePort(startPort: number, usedPorts: Set<number>): number {
  let port = startPort;
  while (usedPorts.has(port) || RESERVED_PORTS.includes(port)) {
    port++;
  }
  return port;
}

export const BackendComponentPicker = ({
  onChange,
  formData,
  idSchema,
}: FieldExtensionComponentProps<BackendConfigData>) => {
  const catalogApi = useApi(catalogApiRef);
  const theme = useTheme();
  const isDarkMode = theme.palette.type === 'dark';

  const { value: catalogEntities, loading } = useAsync(async () => {
    try {
      const response = await catalogApi.getEntities({
        filter: {
          kind: 'component',
        },
      });
      return response.items;
    } catch (e) {
      // Failed to fetch backend components from catalog
      return [];
    }
  }, [catalogApi]);

  const catalogOptions = React.useMemo(() => {
    if (!catalogEntities) return [];
    // Only list backend components (spec.type !== 'website') for connection
    return catalogEntities
      .filter(entity => entity.spec?.type !== 'website')
      .map(entity => entity.metadata.name);
  }, [catalogEntities]);

  // Read ports of already registered catalog components
  const usedPorts = React.useMemo(() => {
    const ports = new Set<number>();
    if (catalogEntities) {
      catalogEntities.forEach(entity => {
        // Skip current component name if it's the one we are editing
        if (formData && entity.metadata.name === formData.backendComponent) {
          return;
        }
        const portStr = entity.metadata.annotations?.['helios.io/port'];
        if (portStr) {
          const p = parseInt(portStr, 10);
          if (!isNaN(p)) {
            ports.add(p);
          }
        }
      });
    }
    return ports;
  }, [catalogEntities, formData]);

  const options = React.useMemo(() => {
    const list: PickerOption[] = [
      { type: 'none', id: 'none', label: 'No Backend (Frontend Only)' },
    ];
    BACKEND_TEMPLATES.forEach(t => {
      list.push({ type: 'template', id: t.id, label: t.name });
    });
    catalogOptions.forEach(name => {
      list.push({ type: 'existing', id: name, label: name });
    });
    return list;
  }, [catalogOptions]);

  const currentOption = React.useMemo(() => {
    const val = formData || {};
    if (!val.backendComponent) {
      return options[0];
    }
    if (val.connectionType === 'Existing') {
      return options.find(opt => opt.type === 'existing' && opt.id === val.backendComponent) || options[0];
    }
    return options.find(opt => opt.type === 'template' && opt.id === val.backendType) || options[0];
  }, [formData, options]);

  const handleSelectionChange = (_: any, option: PickerOption | null) => {
    if (!option || option.id === 'none') {
      onChange({
        backendComponent: undefined,
        backendType: undefined,
        backendPort: undefined,
        backendRepoName: undefined,
        connectionType: 'None',
        backendDatabaseConfig: undefined,
      } as any);
      return;
    }

    if (option.type === 'template') {
      const template = BACKEND_TEMPLATES.find(t => t.id === option.id);
      if (template) {
        const defaultName = `${template.id}-backend`;
        const autoPort = findNextAvailablePort(template.defaultPort, usedPorts);
        onChange({
          backendComponent: defaultName,
          backendType: template.id,
          backendPort: autoPort,
          backendRepoName: defaultName,
          connectionType: 'New',
          backendDatabaseConfig: {
            dbType: 'none',
            dbName: defaultName,
            dbVersion: '18.4',
          },
        });
      }
    } else if (option.type === 'existing') {
      const entity = catalogEntities?.find(e => e.metadata.name === option.id);
      const portStr = entity?.metadata.annotations?.['helios.io/port'];
      const entityPort = portStr ? parseInt(portStr, 10) : 8080;
      onChange({
        backendComponent: option.id,
        backendType: 'existing',
        backendPort: entityPort,
        backendRepoName: option.id,
        connectionType: 'Existing',
        backendDatabaseConfig: {
          dbType: 'none',
        },
      });
    }
  };

  const val = formData || {};

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newName = e.target.value;
    const oldName = val.backendComponent;
    const currentDbName = val.backendDatabaseConfig?.dbName;
    onChange({
      ...val,
      backendComponent: newName,
      backendRepoName: newName,
      backendDatabaseConfig: val.backendDatabaseConfig
        ? {
            ...val.backendDatabaseConfig,
            dbName: (!currentDbName || currentDbName === oldName) ? newName : currentDbName,
          }
        : undefined,
    });
  };

  const handleDbNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newDbName = e.target.value;
    onChange({
      ...val,
      backendDatabaseConfig: val.backendDatabaseConfig
        ? {
            ...val.backendDatabaseConfig,
            dbName: newDbName,
          }
        : undefined,
    });
  };

  const handlePortChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newPort = parseInt(e.target.value, 10);
    onChange({
      ...val,
      backendPort: isNaN(newPort) ? undefined : newPort,
    });
  };

  const handleDbTypeChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    const newType = e.target.value as string;
    onChange({
      ...val,
      backendDatabaseConfig: {
        dbType: newType,
        dbName: newType === 'postgres' ? val.backendComponent : '',
        dbVersion: newType === 'postgres' ? (val.backendDatabaseConfig?.dbVersion || '18.4') : '',
      },
    });
  };

  const handleDbVersionChange = (e: React.ChangeEvent<{ value: unknown }>) => {
    onChange({
      ...val,
      backendDatabaseConfig: val.backendDatabaseConfig
        ? {
            ...val.backendDatabaseConfig,
            dbVersion: e.target.value as string,
          }
        : undefined,
    });
  };

  // Port collision validation:
  const isPortConflict = React.useMemo(() => {
    if (!val.backendPort) return false;
    // Conflict if port is in usedPorts or RESERVED_PORTS
    return usedPorts.has(val.backendPort) || RESERVED_PORTS.includes(val.backendPort);
  }, [val.backendPort, usedPorts]);

  const getGroupHeader = (type: string) => {
    if (type === 'template') {
      return '📦 Create New Backend';
    }
    if (type === 'existing') {
      return '🔗 Connect to Existing Backend';
    }
    return 'None';
  };

  const getHelperText = () => {
    if (val.connectionType === 'Existing') {
      return 'ReadOnly port of the existing service.';
    }
    if (isPortConflict) {
      return `Conflict! Port ${val.backendPort} is reserved or already in use. Please select another.`;
    }
    return 'Port this backend service will run on.';
  };

  return (
    <div style={{ marginTop: '16px', marginBottom: '16px' }}>
      <Autocomplete
        id={idSchema?.$id}
        options={options}
        loading={loading}
        value={currentOption}
        getOptionLabel={option => option.label}
        groupBy={option => getGroupHeader(option.type)}
        onChange={handleSelectionChange}
        renderOption={option => {
          if (option.type === 'none') {
            return <span>{option.label}</span>;
          }
          const isTemplate = option.type === 'template';
          
          // Theme-aware colors
          const bg = isDarkMode
            ? (isTemplate ? 'rgba(76, 175, 80, 0.15)' : 'rgba(33, 150, 243, 0.15)')
            : (isTemplate ? '#e8f5e9' : '#e3f2fd');
          const fg = isDarkMode
            ? (isTemplate ? '#81c784' : '#64b5f6')
            : (isTemplate ? '#2e7d32' : '#1565c0');
          const border = isDarkMode
            ? (isTemplate ? '1px solid rgba(76, 175, 80, 0.3)' : '1px solid rgba(33, 150, 243, 0.3)')
            : (isTemplate ? '1px solid #c8e6c9' : '1px solid #bbdefb');
            
          const text = isTemplate ? 'Template' : 'Existing';

          return (
            <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%', alignItems: 'center' }}>
              <span>{option.label}</span>
              <span
                style={{
                  backgroundColor: bg,
                  color: fg,
                  border: border,
                  padding: '2px 8px',
                  borderRadius: '4px',
                  fontSize: '0.7rem',
                  fontWeight: 'bold',
                  marginLeft: '8px',
                  textTransform: 'uppercase',
                  letterSpacing: '0.5px',
                }}
              >
                {text}
              </span>
            </div>
          );
        }}
        renderInput={params => (
          <TextField
            {...params}
            label="Backend Component Connection"
            variant="standard"
            helperText="Connect a backend service template or select an existing backend from the catalog."
            fullWidth
            margin="normal"
          />
        )}
      />

      {val.backendType && (
        <div style={{ paddingLeft: '16px', borderLeft: '2px solid #ccc', marginTop: '16px', marginBottom: '16px' }}>
          <Typography variant="subtitle1" gutterBottom style={{ fontWeight: 'bold' }}>
            Backend Component Configuration Details
          </Typography>

          <TextField
            label="Backend Component Name"
            value={val.backendComponent || ''}
            onChange={handleNameChange}
            disabled={val.connectionType === 'Existing'}
            fullWidth
            margin="normal"
            required
            helperText={val.connectionType === 'Existing' ? 'ReadOnly name of the existing catalog component.' : 'Name of the backend to create.'}
          />

          <TextField
            label="Backend Component Port"
            type="number"
            value={val.backendPort || ''}
            onChange={handlePortChange}
            disabled={val.connectionType === 'Existing'}
            fullWidth
            margin="normal"
            required
            error={isPortConflict}
            helperText={getHelperText()}
          />

          {val.connectionType !== 'Existing' && val.backendDatabaseConfig && (
            <>
              <Divider style={{ marginTop: '16px', marginBottom: '16px' }} />
              <Typography variant="subtitle2" gutterBottom style={{ fontWeight: 'bold' }}>
                Backend Database Settings
              </Typography>

              <FormControl fullWidth margin="normal">
                <InputLabel>Database Type</InputLabel>
                <Select value={val.backendDatabaseConfig.dbType || 'none'} onChange={handleDbTypeChange}>
                  <MenuItem value="none">No Database</MenuItem>
                  <MenuItem value="postgres">PostgreSQL</MenuItem>
                </Select>
              </FormControl>

              {val.backendDatabaseConfig.dbType === 'postgres' && (
                <>
                  <TextField
                    label="Database Name"
                    value={val.backendDatabaseConfig.dbName || ''}
                    onChange={handleDbNameChange}
                    fullWidth
                    margin="normal"
                    helperText="By default this matches the component name, but you can customize it."
                  />

                  <FormControl fullWidth margin="normal">
                    <InputLabel>Database Version</InputLabel>
                    <Select value={val.backendDatabaseConfig.dbVersion || '18.4'} onChange={handleDbVersionChange}>
                      <MenuItem value="18">18</MenuItem>
                      <MenuItem value="18.4">18.4</MenuItem>
                    </Select>
                  </FormControl>
                </>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
};
