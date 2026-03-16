import { styled } from '@mui/material/styles';
import { alpha } from '@mui/material/styles';
import { Sun } from 'lucide-react';
import Box from '@mui/material/Box';

const Wrapper = styled('div')(({ theme }) => ({
  position: 'relative',
  cursor: 'pointer',
  display: 'inline-flex', // Keeps the wrapper tight around the content
  alignItems: 'center',
  '&:hover .glow-effect': {
    opacity: theme.palette.mode === 'dark' ? 0.6 : 0.4,
  },
}));

const Glow = styled('div')(({ theme }) => ({
  position: 'absolute',
  inset: 0,
  backgroundColor: theme.palette.primary.main,
  filter: 'blur(16px)',
  opacity: theme.palette.mode === 'dark' ? 0.4 : 0.2,
  transition: 'opacity 0.3s ease',
  borderRadius: '50%', // Helps contain the glow
}));

const LogoFull = () => {
  return (
    <Wrapper>
      <Glow className="glow-effect" />
      <Box
        sx={{
          position: 'relative',
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          zIndex: 1,
        }}
      >
        <Sun
          size={28}
          stroke={'currentColor'}
          style={{
            color: 'var(--mui-palette-primary-main)',
            fill: alpha('#EDB506', 0.2),
          }}
        />

        <Box
          component="span"
          sx={{
            display: { xs: 'none', lg: 'block' },
            fontSize: '1.5rem',
            fontWeight: 'bold',
            letterSpacing: '-0.05em',
            color: 'text.primary',
          }}
        >
          HELIOS
          <Box component="span" sx={{ color: 'primary.main' }}>
            .IDP
          </Box>
        </Box>
      </Box>
    </Wrapper>
  );
};

export default LogoFull;
