import { useEntity } from '@backstage/plugin-catalog-react';
import { InfoCard } from '@backstage/core-components';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Box, Typography } from '@material-ui/core';
import { useApi } from '@backstage/core-plugin-api';
import { kubernetesApiRef } from '@backstage/plugin-kubernetes';
import useAsync from 'react-use/lib/useAsync';

export const ResourceUtilizationChart = () => {
  const { entity } = useEntity();
  const kubernetesApi = useApi(kubernetesApiRef);

  // Fetch live K8s objects associated with this entity
  const { value: k8sObjects } = useAsync(async () => {
    return await kubernetesApi.getObjectsByEntity({ entity });
  }, [kubernetesApi, entity]);

  // Extract HeliosApp custom resource from the response
  const heliosAppRaw = k8sObjects?.items?.find((item: any) => 
    item.resources?.some((r: any) => r.kind === 'HeliosApp')
  );
  
  // Drill down to the specific resource
  // Structure: items[].resources[].resources (array of actual k8s objects)
  const heliosAppGroup = heliosAppRaw?.resources?.find((r: any) => r.kind === 'HeliosApp');
  const heliosAppResource = heliosAppGroup?.resources?.[0];

  // Fallback to statis spec if K8s not ready, but prefer K8s status
  const spec = (entity as any).spec || {};
  const liveStatus = heliosAppResource?.status || {};
  
  const resources = spec.resources || {};
  const requests = resources.requests || {};
  const limits = resources.limits || {};

  // Parse Value helpers
  const parseValue = (val: string) => {
    if (!val) return 0;
    if (val.endsWith('m')) return parseInt(val);
    if (val.endsWith('n')) return parseInt(val) / 1000000; // Convert nano to milli
    if (val.endsWith('Mi')) return parseInt(val);
    if (val.endsWith('Gi')) return parseInt(val) * 1024;
    return parseFloat(val) * 1000; // Assume cores if no suffix, convert to m
  };

  const cpuLimit = parseValue(limits.cpu);
  const cpuRequest = parseValue(requests.cpu);
  // Use LIVE status from K8s, not the stale entity status
  const currentCpu = parseValue(liveStatus.currentCPU);

  const memLimit = parseValue(limits.memory);
  const memRequest = parseValue(requests.memory);
  // const currentMem = parseValue(status.currentMemory); // Future use

  // Data for BarChart (Snapshot of current state)
  const data = [
    {
      name: 'CPU (m)',
      Usage: currentCpu,
      Request: cpuRequest,
      Limit: cpuLimit,
    },
    {
      name: 'Memory (Mi)',
      // Usage: currentMem, // Not yet available in backend logic
      Request: memRequest,
      Limit: memLimit,
    },
  ];

  return (
    <InfoCard title="Real-time Resource Utilization">
      <Box height={300}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={data}
            margin={{ top: 20, right: 30, left: 20, bottom: 5 }}
          >
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="Usage" fill="#ff7300" name="Current Usage (Real)" />
            <Bar dataKey="Request" fill="#8884d8" name="Request" />
            <Bar dataKey="Limit" fill="#82ca9d" name="Limit" />
          </BarChart>
        </ResponsiveContainer>
      </Box>
      <Box mt={2}>
        <Typography variant="caption" color="textSecondary">
          * Real data fetched from Kubernetes Pod Metrics via Helios Operator
        </Typography>
      </Box>
    </InfoCard>
  );
};
