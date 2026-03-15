import React from 'react';
import {
  Box,
  ButtonLink,
  Card,
  Flex,
  Grid,
  Text,
  useBreakpoint,
} from '@backstage/ui';
import {
  Activity,
  ArrowRight,
  Cpu,
  GitBranch,
  LayoutGrid,
  Plus,
  Server,
  Terminal,
  Zap,
} from 'lucide-react';
import { Link } from '@backstage/core-components';

// --- STATUS BADGE (theme-aware via styles.css vars) ---
type StatusType = 'healthy' | 'warning' | 'critical' | 'pending' | 'neutral';

const StatusBadge: React.FC<{ status: StatusType; label?: string }> = ({
  status,
  label,
}) => (
  <Flex
    align="center"
    gap="1"
    className={`helios-hp-status-badge helios-hp-status-${status}`}
    style={{
      padding: '4px 12px',
      borderRadius: 2,
      fontSize: 10,
      fontWeight: 700,
      letterSpacing: '0.05em',
      textTransform: 'uppercase',
      width: 'fit-content',
    }}
  >
    <Box
      className="helios-hp-status-dot"
      style={{ width: 6, height: 6, borderRadius: '50%', flexShrink: 0 }}
    />
    {label || status}
  </Flex>
);

// --- METRIC CARD (BUI Box + Text, custom HUD styling) ---
const MetricCard: React.FC<{
  title: string;
  value: string;
  sub: string;
  trend?: 'up' | 'down';
  delay?: string;
}> = ({ title, value, sub, trend, delay = '' }) => (
  <Box
    className={`animate-enter helios-metric-card ${delay}`}
    p="5"
    position="relative"
    style={{
      backgroundColor: 'var(--bui-bg-neutral-1)',
      border: '1px solid var(--bui-border-1)',
      borderRadius: 'var(--bui-radius-2)',
      overflow: 'hidden',
      transition: 'border-color 0.5s ease',
    }}
  >
    {/* Corner markers (solar accent) */}
    <Box
      position="absolute"
      style={{
        top: 0,
        left: 0,
        width: 8,
        height: 8,
        borderLeft: '1px solid var(--helios-solar-subtle)',
        borderTop: '1px solid var(--helios-solar-subtle)',
        opacity: 0.5,
      }}
    />
    <Box
      position="absolute"
      style={{
        top: 0,
        right: 0,
        width: 8,
        height: 8,
        borderRight: '1px solid var(--helios-solar-subtle)',
        borderTop: '1px solid var(--helios-solar-subtle)',
        opacity: 0.5,
      }}
    />
    <Box
      position="absolute"
      style={{
        bottom: 0,
        left: 0,
        width: 8,
        height: 8,
        borderLeft: '1px solid var(--helios-solar-subtle)',
        borderBottom: '1px solid var(--helios-solar-subtle)',
        opacity: 0.5,
      }}
    />
    <Box
      position="absolute"
      style={{
        bottom: 0,
        right: 0,
        width: 8,
        height: 8,
        borderRight: '1px solid var(--helios-solar-subtle)',
        borderBottom: '1px solid var(--helios-solar-subtle)',
        opacity: 0.5,
      }}
    />
    <Flex justify="between" align="start" mb="1">
      <Text
        variant="body-x-small"
        weight="bold"
        style={{
          fontFamily: 'var(--font-mono)',
          color: 'var(--bui-fg-secondary)',
          textTransform: 'uppercase',
          letterSpacing: '0.1em',
        }}
      >
        {title}
      </Text>
      {trend && (
        <Activity
          size={14}
          style={{
            color:
              trend === 'up'
                ? 'var(--bui-fg-success-on-bg)'
                : 'var(--bui-fg-danger-on-bg)',
          }}
        />
      )}
    </Flex>
    <Text
      as="div"
      className="helios-metric-value"
      variant="title-large"
      weight="bold"
      style={{
        fontFamily: 'var(--font-mono)',
        letterSpacing: '-0.02em',
        color: 'var(--bui-fg-primary)',
        marginBottom: 4,
        transition: 'text-shadow 0.3s ease',
      }}
    >
      {value}
    </Text>
    <Text
      as="div"
      className="helios-metric-sub"
      variant="body-small"
      color="secondary"
      style={{
        fontFamily: 'var(--font-mono)',
        borderLeft: '2px solid var(--bui-border-2)',
        paddingLeft: 8,
        transition: 'border-color 0.3s ease',
      }}
    >
      {sub}
    </Text>
  </Box>
);

// --- SERVICE ROW ---
interface Service {
  id: string;
  name: string;
  type: string;
  status: StatusType;
  cpu: number;
}

const ServiceRow: React.FC<{ service: Service; index: number }> = ({
  service,
  index,
}) => {
  const { up } = useBreakpoint();
  const showCpu = up('md');
  const delayClass = `delay-${((index + 1) * 100) % 500}`;

  return (
    <Flex
      className={`animate-enter helios-hp-service-row ${delayClass}`}
      align="center"
      justify="between"
      p="4"
      style={{
        borderBottom: '1px solid var(--bui-border-1)',
        cursor: 'pointer',
        transition: 'background-color 0.2s ease',
      }}
    >
      <Flex align="center" gap="4">
        <Box
          p="2"
          style={{
            borderRadius: 8,
            backgroundColor: 'var(--bui-bg-neutral-2)',
            border: '1px solid var(--bui-border-1)',
            color: 'var(--bui-fg-secondary)',
          }}
        >
          <Server size={18} />
        </Box>
        <Box>
          <Text
            as="div"
            variant="body-medium"
            weight="bold"
            style={{ color: 'var(--bui-fg-primary)' }}
          >
            {service.name}
          </Text>
          <Text
            as="div"
            variant="body-x-small"
            color="secondary"
            style={{
              fontFamily: 'var(--font-mono)',
              color: 'var(--bui-fg-secondary)',
              textTransform: 'uppercase',
            }}
          >
            {service.id} // {service.type}
          </Text>
        </Box>
      </Flex>
      <Flex align="center" gap="6">
        {showCpu && (
          <Flex align="center" gap="2">
            <Text
              as="span"
              variant="body-x-small"
              style={{
                fontFamily: 'var(--font-mono)',
                color: 'var(--bui-fg-secondary)',
              }}
            >
              CPU_LOAD
            </Text>
            <Box
              style={{
                width: 96,
                height: 4,
                backgroundColor: 'var(--bui-bg-neutral-2)',
                borderRadius: 'var(--bui-radius-2)',
                overflow: 'hidden',
              }}
            >
              <Box
                style={{
                  width: `${service.cpu}%`,
                  height: '100%',
                  borderRadius: 'var(--bui-radius-2)',
                  backgroundColor:
                    service.cpu > 80
                      ? 'var(--bui-fg-danger)'
                      : 'var(--helios-solar)',
                  transition: 'width 0.3s ease',
                }}
              />
            </Box>
          </Flex>
        )}
        <StatusBadge status={service.status} />
      </Flex>
    </Flex>
  );
};

const SERVICES: Service[] = [
  {
    id: 'CORE-01',
    name: 'payment-gateway',
    type: 'gRPC',
    status: 'healthy',
    cpu: 45,
  },
  {
    id: 'DATA-09',
    name: 'user-shard-01',
    type: 'Postgres',
    status: 'warning',
    cpu: 82,
  },
  {
    id: 'EDGE-22',
    name: 'cdn-worker-auth',
    type: 'Lambda',
    status: 'healthy',
    cpu: 12,
  },
  {
    id: 'AI-MOD',
    name: 'recommendation-eng',
    type: 'Python',
    status: 'pending',
    cpu: 0,
  },
];

const QUICK_ACTIONS = [
  {
    title: 'Microservice API',
    desc: 'Go + Gin + gRPC',
    icon: <Cpu size={24} />,
    colorVar: '--helios-solar',
    to: '/create',
  },
  {
    title: 'Frontend App',
    desc: 'React + Vite + Edge',
    icon: <LayoutGrid size={24} />,
    colorVar: '--bui-fg-info',
    to: '/create',
  },
  {
    title: 'Data Pipeline',
    desc: 'Python + Kafka',
    icon: <GitBranch size={24} />,
    colorVar: '--bui-fg-danger',
    to: '/create',
  },
];

export const HeliosHomepage = () => {
  return (
    <Box
      style={{
        minHeight: '100%',
        color: 'var(--bui-fg-primary)',
        fontFamily: 'var(--font-sans)',
        padding: '48px',
      }}
    >
      {/* Header */}
      <Flex
        className="animate-enter"
        justify="between"
        align="center"
        mb="12"
        style={{ flexWrap: 'wrap', gap: 16 }}
      >
        <Box>
          <Text
            as="h1"
            variant="title-large"
            weight="bold"
            style={{
              fontSize: 36,
              letterSpacing: '-0.02em',
              marginBottom: 8,
              color: 'var(--bui-fg-primary)',
            }}
          >
            Mission Control
          </Text>
        </Box>
        <Flex align="center" gap="4">
          <ButtonLink
            href="/create"
            variant="primary"
            size="medium"
            iconStart={<Plus />}
            className="helios-mui-primary-btn"
          >
            DEPLOY NEW
          </ButtonLink>
        </Flex>
      </Flex>

      {/* Metrics Grid */}
      <Grid.Root columns={{ initial: '1', sm: '2', lg: '4' }} gap="4" mb="12">
        <Grid.Item>
          <MetricCard
            title="Total Requests"
            value="24.8M"
            sub="2,402 / sec"
            trend="up"
            delay="delay-100"
          />
        </Grid.Item>
        <Grid.Item>
          <MetricCard
            title="Avg Latency"
            value="42ms"
            sub="-12% vs avg"
            trend="down"
            delay="delay-200"
          />
        </Grid.Item>
        <Grid.Item>
          <MetricCard
            title="Error Rate"
            value="0.01%"
            sub="Within SLA"
            delay="delay-300"
          />
        </Grid.Item>
        <Grid.Item>
          <MetricCard
            title="Active Nodes"
            value="842"
            sub="Auto-scaling"
            trend="up"
            delay="delay-100"
          />
        </Grid.Item>
      </Grid.Root>

      {/* Quick Actions */}
      <Box className="animate-enter delay-200" mb="12">
        <Flex align="center" gap="2" mb="6">
          <Zap size={20} style={{ color: 'var(--helios-solar)' }} />
          <Text variant="title-medium" weight="bold">
            Initiate Protocol
          </Text>
        </Flex>
        <Grid.Root columns={{ initial: '1', md: '3' }} gap="6">
          {QUICK_ACTIONS.map((card, i) => (
            <Grid.Item key={i}>
              <Link to={card.to} underline="none">
                <Box
                  className="helios-quick-action-card"
                  p="6"
                  display="flex"
                  style={{
                    flexDirection: 'column',
                    height: '100%',
                    borderRadius: 'var(--bui-radius-2)',
                    border: '1px solid var(--bui-border-1)',
                    backgroundColor: 'var(--bui-bg-neutral-1)',
                    transition:
                      'border-color 0.3s ease, background-color 0.3s ease',
                    cursor: 'pointer',
                  }}
                >
                  <Box
                    p="3"
                    style={{
                      width: 'fit-content',
                      borderRadius: 'var(--bui-radius-2)',
                      backgroundColor: 'var(--bui-bg-neutral-1-hover)',
                      color: `var(${card.colorVar})`,
                      marginBottom: 16,
                    }}
                  >
                    {card.icon}
                  </Box>
                  <Text
                    variant="title-small"
                    weight="bold"
                    style={{
                      fontFamily: 'var(--font-mono)',
                      color: 'var(--bui-fg-primary)',
                      marginBottom: 4,
                    }}
                  >
                    {card.title}
                  </Text>
                  <Text
                    variant="body-small"
                    color="secondary"
                    style={{
                      color: 'var(--bui-fg-secondary)',
                      flex: 1,
                      marginBottom: 24,
                    }}
                  >
                    {card.desc}
                  </Text>
                  <Flex align="center" gap="2">
                    <Text
                      className="helios-quick-action-cta"
                      variant="body-x-small"
                      weight="bold"
                      style={{
                        textTransform: 'uppercase',
                        letterSpacing: '0.05em',
                        color: 'var(--bui-fg-secondary)',
                        transition: 'color 0.2s ease',
                      }}
                    >
                      Launch
                    </Text>
                    <ArrowRight size={14} />
                  </Flex>
                </Box>
              </Link>
            </Grid.Item>
          ))}
        </Grid.Root>
      </Box>

      {/* Service Monitor */}
      <Box className="animate-enter delay-300">
        <Card
          className="helios-glass"
          style={{ borderRadius: 'var(--bui-radius-2)', overflow: 'hidden' }}
        >
          <Box
            p="4"
            style={{
              borderBottom: '1px solid var(--bui-border-1)',
              backgroundColor: 'var(--bui-bg-neutral-1)',
            }}
          >
            <Flex justify="between" align="center">
              <Flex align="center" gap="2">
                <Terminal
                  size={16}
                  style={{ color: 'var(--bui-fg-secondary)' }}
                />
                <Text
                  variant="body-medium"
                  weight="bold"
                  style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--bui-fg-primary)',
                  }}
                >
                  LIVE_FEED
                </Text>
              </Flex>
              <Flex align="center" gap="2">
                <Box
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: '50%',
                    backgroundColor: 'var(--bui-fg-danger)',
                    animation: 'pulse 2s ease-in-out infinite',
                  }}
                />
                <Text
                  as="span"
                  variant="body-x-small"
                  style={{
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--bui-fg-secondary)',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                  }}
                >
                  Realtime
                </Text>
              </Flex>
            </Flex>
          </Box>
          <Box>
            {SERVICES.map((svc, i) => (
              <ServiceRow key={svc.id} service={svc} index={i} />
            ))}
          </Box>
        </Card>
      </Box>
    </Box>
  );
};
