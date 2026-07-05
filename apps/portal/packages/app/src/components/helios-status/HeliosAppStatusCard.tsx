import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Collapse,
  Grid,
  IconButton,
  LinearProgress,
  Tooltip,
  Typography,
} from '@material-ui/core';
import { Alert } from '@material-ui/lab';
import {
  ExpandMore as ExpandMoreIcon,
  Refresh as RefreshIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  HourglassEmpty as PendingIcon,
  DeleteForever as DeletingIcon,
  HelpOutline as UnknownIcon,
  OpenInNew as OpenInNewIcon,
  Notifications as NotificationsIcon,
  Info as InfoIcon,
  Warning as WarningIcon,
} from '@material-ui/icons';
import { makeStyles, Theme } from '@material-ui/core/styles';
import { useEntity } from '@backstage/plugin-catalog-react';
import { useApi, fetchApiRef, configApiRef } from '@backstage/core-plugin-api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type HeliosPhase = 'Pending' | 'Ready' | 'Failed' | 'Deleting' | 'Unknown';

interface HeliosCondition {
  type: string;
  status: string;
  reason: string;
  message: string;
  lastTransitionTime: string;
}

interface HeliosResourceRef {
  apiVersion: string;
  kind: string;
  name: string;
  namespace?: string;
}

interface HeliosComponentSummary {
  name: string;
  type: string;
}

interface HeliosEvent {
  type: string; // Normal | Warning
  reason: string;
  message: string;
  involvedObject: {
    kind: string;
    name: string;
  };
  count: number;
  firstTimestamp: string;
  lastTimestamp: string;
  source: string;
}

interface HeliosAppStatus {
  name: string;
  namespace: string;
  phase: HeliosPhase;
  message: string;
  conditions: HeliosCondition[];
  resourcesCreated: HeliosResourceRef[];
  initialBuildTriggered: boolean;
  lastAppliedHash: string;
  observedGeneration: number;
  owner: string;
  description: string;
  gitRepo: string;
  gitBranch: string;
  gitOpsRepo: string;
  gitOpsPath: string;
  imageRepo: string;
  replicas: number;
  port: number;
  pipelineName: string;
  components: HeliosComponentSummary[];
  createdAt: string;
}

// ---------------------------------------------------------------------------
// Phase → visual mapping
// ---------------------------------------------------------------------------

const PHASE_CONFIG: Record<
  HeliosPhase,
  {
    label: string;
    colorClass: string;
    Icon: typeof CheckCircleIcon;
  }
> = {
  Ready: {
    label: 'Ready',
    colorClass: 'helios-phase-ready',
    Icon: CheckCircleIcon,
  },
  Pending: {
    label: 'Pending',
    colorClass: 'helios-phase-pending',
    Icon: PendingIcon,
  },
  Failed: {
    label: 'Failed',
    colorClass: 'helios-phase-failed',
    Icon: ErrorIcon,
  },
  Deleting: {
    label: 'Deleting',
    colorClass: 'helios-phase-deleting',
    Icon: DeletingIcon,
  },
  Unknown: {
    label: 'Unknown',
    colorClass: 'helios-phase-unknown',
    Icon: UnknownIcon,
  },
};

// ---------------------------------------------------------------------------
// Styles (Material-UI + Helios CSS vars)
// ---------------------------------------------------------------------------

const useStyles = makeStyles((theme: Theme) => ({
  card: {
    position: 'relative',
    overflow: 'hidden',
    border: '1px solid var(--bui-border-1)',
    backgroundColor: 'var(--bui-bg-neutral-1)',
    borderRadius: 'var(--bui-radius-2)',
    transition: 'border-color 0.3s ease',
    '&:hover': {
      borderColor: 'rgba(245, 158, 11, 0.25)',
    },
  },
  cardHeader: {
    padding: theme.spacing(2, 2.5),
    borderBottom: '1px solid var(--bui-border-1)',
    '& .MuiCardHeader-title': {
      fontFamily: 'var(--font-mono)',
      fontWeight: 700,
      fontSize: '0.8rem',
      letterSpacing: '0.08em',
      textTransform: 'uppercase',
      color: 'var(--bui-fg-primary)',
    },
    '& .MuiCardHeader-subheader': {
      fontFamily: 'var(--font-mono)',
      fontSize: '0.7rem',
      letterSpacing: '0.04em',
      color: 'var(--bui-fg-secondary)',
      marginTop: 2,
    },
  },
  cardContent: {
    padding: theme.spacing(2.5),
    '&:last-child': {
      paddingBottom: theme.spacing(2.5),
    },
  },
  phaseRow: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(2),
    marginBottom: theme.spacing(2),
  },
  phaseBadge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 8,
    padding: '6px 16px',
    borderRadius: 2,
    fontFamily: 'var(--font-mono)',
    fontWeight: 700,
    fontSize: '0.75rem',
    letterSpacing: '0.06em',
    textTransform: 'uppercase' as const,
    border: '1px solid',
  },
  phaseMessage: {
    fontFamily: 'var(--font-sans)',
    fontSize: '0.85rem',
    color: 'var(--bui-fg-secondary)',
    padding: theme.spacing(1.5, 2),
    backgroundColor: 'var(--bui-bg-neutral-2)',
    borderRadius: 'var(--bui-radius-2)',
    borderLeft: '3px solid var(--bui-border-2)',
    marginBottom: theme.spacing(2),
  },
  metaGrid: {
    marginBottom: theme.spacing(2),
  },
  metaItem: {
    padding: theme.spacing(1.5, 2),
    backgroundColor: 'var(--bui-bg-neutral-2)',
    border: '1px solid var(--bui-border-1)',
    borderRadius: 'var(--bui-radius-2)',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: 2,
  },
  metaLabel: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.65rem',
    fontWeight: 700,
    letterSpacing: '0.08em',
    textTransform: 'uppercase' as const,
    color: 'var(--bui-fg-secondary)',
  },
  metaValue: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.8rem',
    color: 'var(--bui-fg-primary)',
    wordBreak: 'break-all' as const,
  },
  sectionTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    cursor: 'pointer',
    marginBottom: theme.spacing(1),
    '&:hover': {
      color: 'var(--helios-solar)',
    },
  },
  sectionLabel: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.7rem',
    fontWeight: 700,
    letterSpacing: '0.06em',
    textTransform: 'uppercase' as const,
    color: 'var(--bui-fg-secondary)',
    transition: 'color 0.2s ease',
  },
  expandIcon: {
    transition: 'transform 0.2s ease',
    color: 'var(--bui-fg-secondary)',
    fontSize: '1rem',
  },
  expandIconOpen: {
    transform: 'rotate(180deg)',
  },
  conditionRow: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1.5),
    padding: theme.spacing(1, 1.5),
    borderBottom: '1px solid var(--bui-border-1)',
    '&:last-child': {
      borderBottom: 'none',
    },
  },
  conditionDot: {
    width: 8,
    height: 8,
    borderRadius: '50%',
    marginTop: 5,
    flexShrink: 0,
  },
  conditionContent: {
    flex: 1,
    minWidth: 0,
  },
  conditionType: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.75rem',
    fontWeight: 700,
    color: 'var(--bui-fg-primary)',
  },
  conditionMeta: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.65rem',
    color: 'var(--bui-fg-secondary)',
    marginTop: 2,
  },
  resourceChip: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.7rem',
    fontWeight: 600,
    letterSpacing: '0.02em',
    borderColor: 'var(--bui-border-2)',
    color: 'var(--bui-fg-secondary)',
    '& .MuiChip-icon': {
      color: 'var(--bui-fg-secondary)',
    },
  },
  refreshBtn: {
    color: 'var(--bui-fg-secondary)',
    '&:hover': {
      color: 'var(--helios-solar)',
    },
  },
  refreshing: {
    animation: '$spin 1s linear infinite',
  },
  '@keyframes spin': {
    from: { transform: 'rotate(0deg)' },
    to: { transform: 'rotate(360deg)' },
  },
  loadingBar: {
    position: 'absolute' as const,
    top: 0,
    left: 0,
    right: 0,
    '& .MuiLinearProgress-barColorPrimary': {
      backgroundColor: 'var(--helios-solar)',
    },
    '& .MuiLinearProgress-colorPrimary': {
      backgroundColor: 'var(--bui-bg-neutral-2)',
    },
  },
  cornerMark: {
    position: 'absolute' as const,
    width: 10,
    height: 10,
    opacity: 0.4,
  },
  componentChip: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.65rem',
    fontWeight: 600,
    borderColor: 'var(--helios-solar)',
    color: 'var(--helios-solar)',
  },
  eventRow: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1.5),
    padding: theme.spacing(1, 1.5),
    borderBottom: '1px solid var(--bui-border-1)',
    transition: 'background-color 0.15s ease',
    '&:last-child': {
      borderBottom: 'none',
    },
    '&:hover': {
      backgroundColor: 'var(--bui-bg-neutral-1-hover)',
    },
  },
  eventIcon: {
    marginTop: 3,
    fontSize: '1rem',
    flexShrink: 0,
  },
  eventContent: {
    flex: 1,
    minWidth: 0,
  },
  eventHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    flexWrap: 'wrap' as const,
  },
  eventReason: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.72rem',
    fontWeight: 700,
    color: 'var(--bui-fg-primary)',
  },
  eventKind: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.6rem',
    fontWeight: 600,
    color: 'var(--bui-fg-secondary)',
    padding: '1px 6px',
    borderRadius: 2,
    backgroundColor: 'var(--bui-bg-neutral-2)',
    border: '1px solid var(--bui-border-1)',
  },
  eventMessage: {
    fontFamily: 'var(--font-sans)',
    fontSize: '0.72rem',
    color: 'var(--bui-fg-secondary)',
    marginTop: 2,
    lineHeight: 1.4,
  },
  eventTimestamp: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.6rem',
    color: 'var(--bui-fg-secondary)',
    marginTop: 2,
    opacity: 0.7,
  },
  eventCount: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.6rem',
    fontWeight: 600,
    color: 'var(--bui-fg-warning)',
    marginLeft: 4,
  },
  hashValue: {
    fontFamily: 'var(--font-mono)',
    fontSize: '0.7rem',
    color: 'var(--bui-fg-secondary)',
    letterSpacing: '0.03em',
    backgroundColor: 'var(--bui-bg-neutral-2)',
    padding: '2px 8px',
    borderRadius: 2,
    border: '1px solid var(--bui-border-1)',
    display: 'inline-block',
    cursor: 'pointer',
    transition: 'border-color 0.2s ease',
    '&:hover': {
      borderColor: 'var(--helios-solar)',
    },
  },
  descriptionBox: {
    fontFamily: 'var(--font-sans)',
    fontSize: '0.8rem',
    color: 'var(--bui-fg-secondary)',
    padding: theme.spacing(1.5, 2),
    backgroundColor: 'var(--bui-bg-neutral-2)',
    borderRadius: 'var(--bui-radius-2)',
    border: '1px solid var(--bui-border-1)',
    marginBottom: theme.spacing(2),
    lineHeight: 1.5,
    fontStyle: 'italic' as const,
  },
  eventsNotificationBar: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    padding: theme.spacing(1, 1.5),
    backgroundColor: 'var(--bui-bg-warning)',
    border: '1px solid var(--bui-border-warning)',
    borderRadius: 'var(--bui-radius-2)',
    marginBottom: theme.spacing(2),
    cursor: 'pointer',
    transition: 'all 0.2s ease',
    '&:hover': {
      borderColor: 'var(--helios-solar)',
    },
  },
  eventsNotificationNormal: {
    backgroundColor: 'var(--bui-bg-info)',
    border: '1px solid var(--bui-border-info)',
    '&:hover': {
      borderColor: 'var(--bui-fg-info)',
    },
  },
}));

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const REFRESH_INTERVAL_MS = 15_000;

function formatTimestamp(iso: string): string {
  if (!iso) return '—';
  try {
    const d = new Date(iso);
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return iso;
  }
}

function relativeTime(iso: string): string {
  if (!iso) return '';
  try {
    const diff = Date.now() - new Date(iso).getTime();
    const secs = Math.floor(diff / 1000);
    if (secs < 60) return `${secs}s ago`;
    const mins = Math.floor(secs / 60);
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  } catch {
    return '';
  }
}

const RESOURCE_KIND_ICONS: Record<string, string> = {
  Deployment: '🚀',
  StatefulSet: '🗄️',
  Service: '🌐',
  Ingress: '🔗',
  Secret: '🔑',
  ConfigMap: '📄',
  PipelineRun: '⚙️',
  Application: '📦',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const HeliosAppStatusCard: React.FC = () => {
  const classes = useStyles();
  const { entity } = useEntity();
  const fetchApi = useApi(fetchApiRef);
  const configApi = useApi(configApiRef);

  const componentName = entity.metadata.name;
  const namespace =
    entity.metadata.annotations?.['helios.io/namespace'] ?? 'default';

  const [status, setStatus] = useState<HeliosAppStatus | null>(null);
  const [events, setEvents] = useState<HeliosEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conditionsOpen, setConditionsOpen] = useState(false);
  const [resourcesOpen, setResourcesOpen] = useState(false);
  const [eventsOpen, setEventsOpen] = useState(false);
  const [specDetailsOpen, setSpecDetailsOpen] = useState(false);

  const fetchStatus = useCallback(
    async (isManual = false) => {
      try {
        if (isManual) setRefreshing(true);
        const backendUrl = configApi.getString('backend.baseUrl');

        // Fetch status + events in parallel
        const [statusResponse, eventsResponse] = await Promise.all([
          fetchApi.fetch(
            `${backendUrl}/api/helios/status/${componentName}?namespace=${namespace}`,
          ),
          fetchApi
            .fetch(
              `${backendUrl}/api/helios/events/${componentName}?namespace=${namespace}&limit=20`,
            )
            .catch(() => null),
        ]);

        if (statusResponse.status === 404) {
          const data = await statusResponse.json();
          setStatus(data);
          setError(null);
          return;
        }

        if (!statusResponse.ok) {
          throw new Error(
            `HTTP ${statusResponse.status}: ${statusResponse.statusText}`,
          );
        }

        const data = await statusResponse.json();
        setStatus(data);
        setError(null);

        // Parse events (best effort)
        if (eventsResponse && eventsResponse.ok) {
          const eventsData = await eventsResponse.json();
          setEvents(eventsData.events ?? []);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [componentName, namespace, fetchApi, configApi],
  );

  // Initial fetch + polling
  useEffect(() => {
    fetchStatus();
    const interval = setInterval(() => fetchStatus(), REFRESH_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchStatus]);

  // Derived: count warning events
  const warningEvents = events.filter(e => e.type === 'Warning');
  const hasWarnings = warningEvents.length > 0;

  // ---------- Loading state ----------
  if (loading) {
    return (
      <Card className={classes.card}>
        <LinearProgress className={classes.loadingBar} />
        <CardHeader
          className={classes.cardHeader}
          title="Helios Application Status"
          subheader="Fetching from cluster..."
        />
        <CardContent className={classes.cardContent}>
          <Box display="flex" justifyContent="center" py={3}>
            <Typography
              variant="body2"
              style={{
                fontFamily: 'var(--font-mono)',
                color: 'var(--bui-fg-secondary)',
              }}
            >
              Connecting to K8s API…
            </Typography>
          </Box>
        </CardContent>
      </Card>
    );
  }

  // ---------- Error state ----------
  if (error) {
    return (
      <Card className={classes.card}>
        <CardHeader
          className={classes.cardHeader}
          title="Helios Application Status"
          action={
            <Tooltip title="Retry">
              <IconButton
                size="small"
                className={classes.refreshBtn}
                onClick={() => {
                  setLoading(true);
                  fetchStatus(true);
                }}
              >
                <RefreshIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          }
        />
        <CardContent className={classes.cardContent}>
          <Alert severity="warning" variant="outlined">
            <Typography variant="body2">
              Could not fetch HeliosApp status: {error}
            </Typography>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  // ---------- Not a Helios app ----------
  if (!status || status.phase === 'Unknown') {
    return null; // Gracefully hide — entity is not managed by Helios
  }

  const phaseConfig = PHASE_CONFIG[status.phase] ?? PHASE_CONFIG.Unknown;
  const { Icon: PhaseIcon, colorClass, label: phaseLabel } = phaseConfig;

  return (
    <Card className={`${classes.card} helios-status-card`}>
      {refreshing && <LinearProgress className={classes.loadingBar} />}

      {/* Corner accents */}
      <Box
        className={classes.cornerMark}
        style={{
          top: 0,
          left: 0,
          borderLeft: '1px solid var(--helios-solar-subtle)',
          borderTop: '1px solid var(--helios-solar-subtle)',
        }}
      />
      <Box
        className={classes.cornerMark}
        style={{
          top: 0,
          right: 0,
          borderRight: '1px solid var(--helios-solar-subtle)',
          borderTop: '1px solid var(--helios-solar-subtle)',
        }}
      />
      <Box
        className={classes.cornerMark}
        style={{
          bottom: 0,
          left: 0,
          borderLeft: '1px solid var(--helios-solar-subtle)',
          borderBottom: '1px solid var(--helios-solar-subtle)',
        }}
      />
      <Box
        className={classes.cornerMark}
        style={{
          bottom: 0,
          right: 0,
          borderRight: '1px solid var(--helios-solar-subtle)',
          borderBottom: '1px solid var(--helios-solar-subtle)',
        }}
      />

      <CardHeader
        className={classes.cardHeader}
        title="Helios Application Status"
        subheader={`${status.namespace}/${status.name}  ·  ${
          status.createdAt ? relativeTime(status.createdAt) : ''
        }`}
        action={
          <Tooltip title="Refresh now (auto-refreshes every 15s)">
            <IconButton
              size="small"
              className={classes.refreshBtn}
              onClick={() => fetchStatus(true)}
            >
              <RefreshIcon
                fontSize="small"
                className={refreshing ? classes.refreshing : undefined}
              />
            </IconButton>
          </Tooltip>
        }
      />

      <CardContent className={classes.cardContent}>
        {/* Phase badge + build status */}
        <Box className={classes.phaseRow}>
          <Box className={`${classes.phaseBadge} ${colorClass}`}>
            <PhaseIcon style={{ fontSize: 16 }} />
            {phaseLabel}
          </Box>
          {status.initialBuildTriggered && (
            <Chip
              label="Initial Build Triggered"
              variant="outlined"
              size="small"
              className={classes.componentChip}
            />
          )}
          {status.components.map(comp => (
            <Chip
              key={comp.name}
              label={`${comp.name} (${comp.type})`}
              variant="outlined"
              size="small"
              className={classes.componentChip}
            />
          ))}
        </Box>

        {/* Status message */}
        {status.message && (
          <Box className={classes.phaseMessage}>{status.message}</Box>
        )}

        {/* Description */}
        {status.description && (
          <Box className={classes.descriptionBox}>{status.description}</Box>
        )}

        {/* Events notification bar */}
        {events.length > 0 && (
          <Box
            className={`${classes.eventsNotificationBar} ${
              !hasWarnings ? classes.eventsNotificationNormal : ''
            }`}
            onClick={() => setEventsOpen(prev => !prev)}
          >
            {hasWarnings ? (
              <WarningIcon
                style={{ fontSize: 18, color: 'var(--bui-fg-warning)' }}
              />
            ) : (
              <InfoIcon style={{ fontSize: 18, color: 'var(--bui-fg-info)' }} />
            )}
            <Typography
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: '0.7rem',
                fontWeight: 700,
                color: hasWarnings
                  ? 'var(--bui-fg-warning)'
                  : 'var(--bui-fg-info)',
                letterSpacing: '0.04em',
                flex: 1,
              }}
            >
              {hasWarnings
                ? `${warningEvents.length} warning${
                    warningEvents.length > 1 ? 's' : ''
                  } detected`
                : `${events.length} recent event${
                    events.length > 1 ? 's' : ''
                  }`}
            </Typography>
            <NotificationsIcon
              style={{
                fontSize: 14,
                color: hasWarnings
                  ? 'var(--bui-fg-warning)'
                  : 'var(--bui-fg-info)',
                opacity: 0.6,
              }}
            />
          </Box>
        )}

        {/* Spec metadata */}
        <Grid container spacing={1} className={classes.metaGrid}>
          {status.owner && (
            <Grid item xs={6} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Owner</Typography>
                <Typography className={classes.metaValue}>
                  {status.owner}
                </Typography>
              </Box>
            </Grid>
          )}
          {status.replicas > 0 && (
            <Grid item xs={6} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Replicas</Typography>
                <Typography className={classes.metaValue}>
                  {status.replicas}
                </Typography>
              </Box>
            </Grid>
          )}
          {status.port > 0 && (
            <Grid item xs={6} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Port</Typography>
                <Typography className={classes.metaValue}>
                  :{status.port}
                </Typography>
              </Box>
            </Grid>
          )}
          {status.gitBranch && (
            <Grid item xs={6} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Branch</Typography>
                <Typography className={classes.metaValue}>
                  {status.gitBranch}
                </Typography>
              </Box>
            </Grid>
          )}
          {status.imageRepo && (
            <Grid item xs={12} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Image</Typography>
                <Tooltip title={status.imageRepo}>
                  <Typography className={classes.metaValue} noWrap>
                    {status.imageRepo}
                  </Typography>
                </Tooltip>
              </Box>
            </Grid>
          )}
          {status.pipelineName && (
            <Grid item xs={12} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Pipeline</Typography>
                <Typography className={classes.metaValue}>
                  {status.pipelineName}
                </Typography>
              </Box>
            </Grid>
          )}
          {status.createdAt && (
            <Grid item xs={12} sm={3}>
              <Box className={classes.metaItem}>
                <Typography className={classes.metaLabel}>Created</Typography>
                <Tooltip title={status.createdAt}>
                  <Typography className={classes.metaValue}>
                    {formatTimestamp(status.createdAt)}
                  </Typography>
                </Tooltip>
              </Box>
            </Grid>
          )}
        </Grid>

        {/* Last Applied Hash */}
        {status.lastAppliedHash && (
          <Box mb={2} display="flex" alignItems="center" style={{ gap: 8 }}>
            <Typography
              className={classes.metaLabel}
              style={{ whiteSpace: 'nowrap' }}
            >
              Last Sync Hash
            </Typography>
            <Tooltip
              title={`Full hash: ${status.lastAppliedHash} — Click to copy`}
            >
              <span
                className={classes.hashValue}
                onClick={() => {
                  navigator.clipboard?.writeText(status.lastAppliedHash);
                }}
              >
                {status.lastAppliedHash.length > 12
                  ? `${status.lastAppliedHash.slice(0, 12)}…`
                  : status.lastAppliedHash}
              </span>
            </Tooltip>
            {status.observedGeneration > 0 && (
              <Typography
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: '0.65rem',
                  color: 'var(--bui-fg-secondary)',
                }}
              >
                gen {status.observedGeneration}
              </Typography>
            )}
          </Box>
        )}

        {/* Conditions (expandable) */}
        {status.conditions.length > 0 && (
          <Box mb={2}>
            <Box
              className={classes.sectionTitle}
              onClick={() => setConditionsOpen(prev => !prev)}
            >
              <ExpandMoreIcon
                className={`${classes.expandIcon} ${
                  conditionsOpen ? classes.expandIconOpen : ''
                }`}
              />
              <Typography className={classes.sectionLabel}>
                Conditions ({status.conditions.length})
              </Typography>
            </Box>
            <Collapse in={conditionsOpen}>
              <Box
                style={{
                  backgroundColor: 'var(--bui-bg-neutral-2)',
                  borderRadius: 'var(--bui-radius-2)',
                  border: '1px solid var(--bui-border-1)',
                  overflow: 'hidden',
                }}
              >
                {status.conditions.map((cond, i) => (
                  <Box key={i} className={classes.conditionRow}>
                    <Box
                      className={classes.conditionDot}
                      style={{
                        backgroundColor:
                          cond.status === 'True'
                            ? 'var(--bui-fg-success)'
                            : cond.status === 'False'
                              ? 'var(--bui-fg-danger)'
                              : 'var(--bui-fg-warning)',
                      }}
                    />
                    <Box className={classes.conditionContent}>
                      <Typography className={classes.conditionType}>
                        {cond.type}
                        {cond.reason && (
                          <span
                            style={{
                              color: 'var(--bui-fg-secondary)',
                              fontWeight: 400,
                              marginLeft: 8,
                            }}
                          >
                            ({cond.reason})
                          </span>
                        )}
                      </Typography>
                      {cond.message && (
                        <Typography className={classes.conditionMeta}>
                          {cond.message}
                        </Typography>
                      )}
                      {cond.lastTransitionTime && (
                        <Typography className={classes.conditionMeta}>
                          {formatTimestamp(cond.lastTransitionTime)}
                        </Typography>
                      )}
                    </Box>
                  </Box>
                ))}
              </Box>
            </Collapse>
          </Box>
        )}

        {/* Resources Created (expandable) */}
        {status.resourcesCreated.length > 0 && (
          <Box mb={2}>
            <Box
              className={classes.sectionTitle}
              onClick={() => setResourcesOpen(prev => !prev)}
            >
              <ExpandMoreIcon
                className={`${classes.expandIcon} ${
                  resourcesOpen ? classes.expandIconOpen : ''
                }`}
              />
              <Typography className={classes.sectionLabel}>
                Resources Created ({status.resourcesCreated.length})
              </Typography>
            </Box>
            <Collapse in={resourcesOpen}>
              <Box display="flex" flexWrap="wrap" style={{ gap: 6 }}>
                {status.resourcesCreated.map((res, i) => (
                  <Tooltip
                    key={i}
                    title={`${res.apiVersion}/${res.kind} — ${
                      res.namespace ?? 'default'
                    }/${res.name}`}
                  >
                    <Chip
                      icon={
                        <span style={{ fontSize: 14, lineHeight: 1 }}>
                          {RESOURCE_KIND_ICONS[res.kind] ?? '📦'}
                        </span>
                      }
                      label={`${res.kind}/${res.name}`}
                      variant="outlined"
                      size="small"
                      className={classes.resourceChip}
                      deleteIcon={<OpenInNewIcon style={{ fontSize: 12 }} />}
                    />
                  </Tooltip>
                ))}
              </Box>
            </Collapse>
          </Box>
        )}

        {/* Spec Details (expandable) */}
        {(status.gitOpsRepo || status.gitOpsPath) && (
          <Box mb={2}>
            <Box
              className={classes.sectionTitle}
              onClick={() => setSpecDetailsOpen(prev => !prev)}
            >
              <ExpandMoreIcon
                className={`${classes.expandIcon} ${
                  specDetailsOpen ? classes.expandIconOpen : ''
                }`}
              />
              <Typography className={classes.sectionLabel}>
                Spec Details
              </Typography>
            </Box>
            <Collapse in={specDetailsOpen}>
              <Box
                style={{
                  backgroundColor: 'var(--bui-bg-neutral-2)',
                  borderRadius: 'var(--bui-radius-2)',
                  border: '1px solid var(--bui-border-1)',
                  padding: 12,
                  overflow: 'hidden',
                }}
              >
                <Grid container spacing={1}>
                  {status.gitOpsRepo && (
                    <Grid item xs={12} sm={6}>
                      <Typography className={classes.metaLabel}>
                        GitOps Repo
                      </Typography>
                      <Tooltip title={status.gitOpsRepo}>
                        <Typography className={classes.metaValue} noWrap>
                          {status.gitOpsRepo.replace(/^https?:\/\//, '')}
                        </Typography>
                      </Tooltip>
                    </Grid>
                  )}
                  {status.gitOpsPath && (
                    <Grid item xs={12} sm={6}>
                      <Typography className={classes.metaLabel}>
                        GitOps Path
                      </Typography>
                      <Typography className={classes.metaValue}>
                        {status.gitOpsPath}
                      </Typography>
                    </Grid>
                  )}
                </Grid>
              </Box>
            </Collapse>
          </Box>
        )}

        {/* Recent Events (expandable) */}
        {events.length > 0 && (
          <Box>
            <Box
              className={classes.sectionTitle}
              onClick={() => setEventsOpen(prev => !prev)}
            >
              <ExpandMoreIcon
                className={`${classes.expandIcon} ${
                  eventsOpen ? classes.expandIconOpen : ''
                }`}
              />
              <Typography className={classes.sectionLabel}>
                Recent Events ({events.length})
                {hasWarnings && (
                  <span
                    style={{ color: 'var(--bui-fg-warning)', marginLeft: 8 }}
                  >
                    ⚠ {warningEvents.length}
                  </span>
                )}
              </Typography>
            </Box>
            <Collapse in={eventsOpen}>
              <Box
                style={{
                  backgroundColor: 'var(--bui-bg-neutral-2)',
                  borderRadius: 'var(--bui-radius-2)',
                  border: '1px solid var(--bui-border-1)',
                  overflow: 'hidden',
                  maxHeight: 320,
                  overflowY: 'auto',
                }}
              >
                {events.map((evt, i) => {
                  const isWarning = evt.type === 'Warning';
                  return (
                    <Box key={i} className={classes.eventRow}>
                      {isWarning ? (
                        <WarningIcon
                          className={classes.eventIcon}
                          style={{ color: 'var(--bui-fg-warning)' }}
                        />
                      ) : (
                        <InfoIcon
                          className={classes.eventIcon}
                          style={{ color: 'var(--bui-fg-info)' }}
                        />
                      )}
                      <Box className={classes.eventContent}>
                        <Box className={classes.eventHeader}>
                          <Typography className={classes.eventReason}>
                            {evt.reason}
                          </Typography>
                          <span className={classes.eventKind}>
                            {evt.involvedObject.kind}/{evt.involvedObject.name}
                          </span>
                          {evt.count > 1 && (
                            <span className={classes.eventCount}>
                              ×{evt.count}
                            </span>
                          )}
                        </Box>
                        {evt.message && (
                          <Typography className={classes.eventMessage}>
                            {evt.message}
                          </Typography>
                        )}
                        <Typography className={classes.eventTimestamp}>
                          {formatTimestamp(evt.lastTimestamp)}
                          {evt.source && ` · ${evt.source}`}
                        </Typography>
                      </Box>
                    </Box>
                  );
                })}
              </Box>
            </Collapse>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
