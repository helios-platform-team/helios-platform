import { makeStyles } from '@material-ui/core/styles';
import { Sun } from 'lucide-react';
import Box from '@material-ui/core/Box';

const useStyles = makeStyles(theme => ({
  wrapper: {
    position: 'relative',
    cursor: 'pointer',
    display: 'inline-flex',
    alignItems: 'center',
    '&:hover $glow': {
      opacity: theme.palette.type === 'dark' ? 0.6 : 0.4,
    },
  },
  glow: {
    position: 'absolute',
    inset: 0,
    backgroundColor: theme.palette.primary.main,
    filter: 'blur(16px)',
    opacity: theme.palette.type === 'dark' ? 0.4 : 0.2,
    transition: 'opacity 0.3s ease',
    borderRadius: '50%',
  },
  logoContainer: {
    position: 'relative',
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    zIndex: 1,
  },
  logoText: {
    display: 'none',
    [theme.breakpoints.up('lg')]: {
      display: 'block',
    },
    fontSize: '1.5rem',
    fontWeight: 'bold',
    letterSpacing: '-0.05em',
    color: theme.palette.text.primary,
  },
  logoHighlight: {
    color: theme.palette.primary.main,
  },
}));

const LogoFull = () => {
  const classes = useStyles();
  return (
    <div className={classes.wrapper}>
      <div className={`${classes.glow} glow-effect`} />
      <Box className={classes.logoContainer}>
        <Sun
          size={28}
          stroke="currentColor"
          style={{
            color: 'var(--mui-palette-primary-main)',
            fill: 'rgba(237, 181, 6, 0.2)',
          }}
        />
        <span className={classes.logoText}>
          HELIOS
          <span className={classes.logoHighlight}>.IDP</span>
        </span>
      </Box>
    </div>
  );
};

export default LogoFull;
