import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Grid,
  Typography,
  Chip,
  IconButton,
  TextField,
  Button,
  CircularProgress,
} from '@material-ui/core';
import { Alert } from '@material-ui/lab';
import {
  Visibility,
  VisibilityOff,
  FileCopy,
  FileCopyOutlined,
} from '@material-ui/icons';
import { makeStyles } from '@material-ui/core/styles';
import { useEntity } from '@backstage/plugin-catalog-react';
import { useApi, fetchApiRef, configApiRef } from '@backstage/core-plugin-api';

const useStyles = makeStyles(theme => ({
  root: {
    padding: theme.spacing(3),
  },
  card: {
    marginBottom: theme.spacing(2),
  },
  cardTitle: {
    marginBottom: theme.spacing(2),
  },
  connectivityGrid: {
    marginBottom: theme.spacing(2),
  },
  connectivityItem: {
    padding: theme.spacing(2),
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  credentialField: {
    marginBottom: theme.spacing(2),
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  },
  passwordField: {
    flex: 1,
  },
  statusBadge: {
    marginBottom: theme.spacing(2),
  },
  snippet: {
    marginBottom: theme.spacing(2),
  },
  snippetCode: {
    backgroundColor: theme.palette.grey[900],
    color: theme.palette.grey[50],
    padding: theme.spacing(2),
    borderRadius: theme.shape.borderRadius,
    fontFamily: 'monospace',
    fontSize: '0.875rem',
    overflowX: 'auto',
  },
  snippetButton: {
    marginLeft: theme.spacing(1),
  },
}));

interface DatabaseInfo {
  host: string;
  port: number;
  user: string;
  password: string;
  database: string;
  status: 'Running' | 'Failed' | 'Unknown';
}

export const DatabaseTab: React.FC = () => {
  const classes = useStyles();
  const { entity } = useEntity();
  const [databaseInfo, setDatabaseInfo] = useState<DatabaseInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const fetchApi = useApi(fetchApiRef);
  const configApi = useApi(configApiRef);

  const componentName = entity.metadata.name;

  useEffect(() => {
    const fetchDatabaseInfo = async () => {
      try {
        setLoading(true);
        const backendUrl = configApi.getString('backend.baseUrl');
        const response = await fetchApi.fetch(
          `${backendUrl}/api/helios/info/${componentName}`,
        );
        if (!response.ok) {
          throw new Error(`Failed to fetch database info: ${response.statusText}`);
        }
        const data = await response.json();
        setDatabaseInfo(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        setDatabaseInfo(null);
      } finally {
        setLoading(false);
      }
    };

    fetchDatabaseInfo();
  }, [componentName]);

  const handleCopyToClipboard = (text: string, fieldName: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const generatePsqlSnippet = () => {
    if (!databaseInfo) return '';
    return `psql -h ${databaseInfo.host} -U ${databaseInfo.user} -d ${databaseInfo.database} -p ${databaseInfo.port}`;
  };

  const generateNodeSnippet = (masked: boolean = true) => {
    if (!databaseInfo) return '';
    const password = masked ? '********' : databaseInfo.password;
    return `const pg = require('pg');
const client = new pg.Client({
  host: '${databaseInfo.host}',
  port: ${databaseInfo.port},
  user: '${databaseInfo.user}',
  password: '${password}',
  database: '${databaseInfo.database}',
});

client.connect();
// Use client for queries`;
  };

  const generatePythonSnippet = (masked: boolean = true) => {
    if (!databaseInfo) return '';
    const password = masked ? '********' : databaseInfo.password;
    return `import psycopg2

conn = psycopg2.connect(
    host="${databaseInfo.host}",
    port=${databaseInfo.port},
    user="${databaseInfo.user}",
    password="${password}",
    database="${databaseInfo.database}"
)
cursor = conn.cursor()`;
  };

  if (loading) {
    return (
      <Box className={classes.root}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Box className={classes.root}>
        <Alert severity="error">
          <strong>Failed to fetch database information:</strong>
          <br />
          {error}
          <br />
          <Typography variant="caption" style={{ marginTop: '8px', display: 'block' }}>
            Make sure the backend service is running and the component has a corresponding
            Kubernetes secret named &quot;{componentName}-backend-db-secret&quot; and pod with label
            &quot;app={componentName}-backend-db&quot;.
          </Typography>
        </Alert>
      </Box>
    );
  }

  if (!databaseInfo) {
    return (
      <Box className={classes.root}>
        <Alert severity="info">No database information available</Alert>
      </Box>
    );
  }

  return (
    <Box className={classes.root}>
      <Grid container spacing={3}>
        {/* Status Card */}
        <Grid item xs={12}>
          <Card className={classes.card}>
            <CardHeader
              title="Database Status"
              className={classes.cardTitle}
            />
            <CardContent>
              <Box className={classes.statusBadge}>
                <Chip
                  label={databaseInfo.status}
                  color={
                    databaseInfo.status === 'Running' ? 'primary' : 'secondary'
                  }
                  variant="outlined"
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Connectivity Information */}
        <Grid item xs={12}>
          <Card className={classes.card}>
            <CardHeader
              title="Connectivity Details"
              className={classes.cardTitle}
            />
            <CardContent>
              <Grid
                container
                spacing={2}
                className={classes.connectivityGrid}
              >
                {/* Host */}
                <Grid item xs={12} sm={6}>
                  <Box className={classes.connectivityItem}>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Host
                      </Typography>
                      <Typography variant="body2">
                        {databaseInfo.host}
                      </Typography>
                    </Box>
                    <IconButton
                      size="small"
                      onClick={() =>
                        handleCopyToClipboard(
                          databaseInfo.host,
                          'host',
                        )
                      }
                      title="Copy to clipboard"
                    >
                      <FileCopyOutlined fontSize="small" />
                    </IconButton>
                  </Box>
                </Grid>

                {/* Port */}
                <Grid item xs={12} sm={6}>
                  <Box className={classes.connectivityItem}>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Port
                      </Typography>
                      <Typography variant="body2">
                        {databaseInfo.port}
                      </Typography>
                    </Box>
                    <IconButton
                      size="small"
                      onClick={() =>
                        handleCopyToClipboard(
                          String(databaseInfo.port),
                          'port',
                        )
                      }
                      title="Copy to clipboard"
                    >
                      <FileCopyOutlined fontSize="small" />
                    </IconButton>
                  </Box>
                </Grid>

                {/* User */}
                <Grid item xs={12} sm={6}>
                  <Box className={classes.connectivityItem}>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        User
                      </Typography>
                      <Typography variant="body2">
                        {databaseInfo.user}
                      </Typography>
                    </Box>
                    <IconButton
                      size="small"
                      onClick={() =>
                        handleCopyToClipboard(
                          databaseInfo.user,
                          'user',
                        )
                      }
                      title="Copy to clipboard"
                    >
                      <FileCopyOutlined fontSize="small" />
                    </IconButton>
                  </Box>
                </Grid>

                {/* Database */}
                <Grid item xs={12} sm={6}>
                  <Box className={classes.connectivityItem}>
                    <Box>
                      <Typography variant="caption" color="textSecondary">
                        Database
                      </Typography>
                      <Typography variant="body2">
                        {databaseInfo.database}
                      </Typography>
                    </Box>
                    <IconButton
                      size="small"
                      onClick={() =>
                        handleCopyToClipboard(
                          databaseInfo.database,
                          'database',
                        )
                      }
                      title="Copy to clipboard"
                    >
                      <FileCopyOutlined fontSize="small" />
                    </IconButton>
                  </Box>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Grid>

        {/* Credentials */}
        <Grid item xs={12}>
          <Card className={classes.card}>
            <CardHeader title="Credentials" className={classes.cardTitle} />
            <CardContent>
              {/* Password Field */}
              <Box className={classes.credentialField}>
                <TextField
                  disabled
                  label="Password"
                  type={showPassword ? 'text' : 'password'}
                  value={databaseInfo.password}
                  className={classes.passwordField}
                  variant="outlined"
                  size="small"
                />
                <IconButton
                  size="small"
                  onClick={() => setShowPassword(!showPassword)}
                  title={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? <VisibilityOff /> : <Visibility />}
                </IconButton>
                <IconButton
                  size="small"
                  onClick={() =>
                    handleCopyToClipboard(
                      databaseInfo.password,
                      'password',
                    )
                  }
                  className={classes.snippetButton}
                  title="Copy to clipboard"
                >
                  <FileCopyOutlined fontSize="small" />
                </IconButton>
                {copiedField === 'password' && (
                  <Typography variant="caption" color="primary">
                    Copied!
                  </Typography>
                )}
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Connection Snippets */}
        <Grid item xs={12}>
          <Card className={classes.card}>
            <CardHeader
              title="Connection Snippets"
              className={classes.cardTitle}
            />
            <CardContent>
              {/* psql Snippet */}
              <Box className={classes.snippet}>
                <Typography variant="subtitle2" gutterBottom>
                  psql
                </Typography>
                <Box className={classes.snippetCode}>
                  {generatePsqlSnippet()}
                </Box>
                <Button
                  size="small"
                  startIcon={<FileCopy />}
                  onClick={() =>
                    handleCopyToClipboard(
                      generatePsqlSnippet(),
                      'psql',
                    )
                  }
                  className={classes.snippetButton}
                >
                  Copy
                </Button>
              </Box>

              {/* Node.js Snippet */}
              <Box className={classes.snippet}>
                <Typography variant="subtitle2" gutterBottom>
                  Node.js (pg package)
                </Typography>
                <Box className={classes.snippetCode}>
                  <pre>{generateNodeSnippet(!showPassword)}</pre>
                </Box>
                <Button
                  size="small"
                  startIcon={<FileCopy />}
                  onClick={() =>
                    handleCopyToClipboard(
                      generateNodeSnippet(!showPassword),
                      'nodejs',
                    )
                  }
                  className={classes.snippetButton}
                >
                  Copy
                </Button>
              </Box>

              {/* Python Snippet */}
              <Box className={classes.snippet}>
                <Typography variant="subtitle2" gutterBottom>
                  Python (psycopg2)
                </Typography>
                <Box className={classes.snippetCode}>
                  <pre>{generatePythonSnippet(!showPassword)}</pre>
                </Box>
                <Button
                  size="small"
                  startIcon={<FileCopy />}
                  onClick={() =>
                    handleCopyToClipboard(
                      generatePythonSnippet(!showPassword),
                      'python',
                    )
                  }
                  className={classes.snippetButton}
                >
                  Copy
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};
