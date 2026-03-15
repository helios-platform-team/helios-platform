import { styled, alpha } from '@mui/material/styles';
import { Sun } from 'lucide-react';
import Box from '@mui/material/Box';

// Reusing the same styled components for consistency
const Wrapper = styled('div')(({ theme }) => ({
  position: 'relative',
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  '&:hover .glow-effect': {
    opacity: theme.palette.mode === 'dark' ? 0.6 : 0.4,
  },
}));

const Glow = styled('div')(({ theme }) => ({
  position: 'absolute',
  inset: -4, // Slightly negative inset makes the glow expand beyond the icon boundaries
  backgroundColor: theme.palette.primary.main,
  filter: 'blur(12px)', // Adjusted blur for a smaller icon
  opacity: theme.palette.mode === 'dark' ? 0.4 : 0.2,
  transition: 'opacity 0.3s ease',
  borderRadius: '50%',
}));

const LogoIcon = () => {
  return (
    <Wrapper>
      <Glow className="glow-effect" />
      <Box
        sx={{
          position: 'relative',
          display: 'flex',
          zIndex: 1,
        }}
      >
        <Sun
          size={28}
          stroke="currentColor"
          style={{
            color: 'var(--mui-palette-primary-main)',
            fill: alpha('#EDB506', 0.2),
          }}
        />
      </Box>
    </Wrapper>
  );
};

export default LogoIcon;