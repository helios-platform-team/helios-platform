import { makeStyles } from '@material-ui/core/styles';
import { Sun } from 'lucide-react';
import Box from '@material-ui/core/Box';

const useStyles = makeStyles(theme => ({
  wrapper: {
    position: 'relative',
    cursor: 'pointer',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    '&:hover $glow': {
      opacity: theme.palette.type === 'dark' ? 0.6 : 0.4,
    },
  },
  glow: {
    position: 'absolute',
    inset: -4,
    backgroundColor: theme.palette.primary.main,
    filter: 'blur(12px)',
    opacity: theme.palette.type === 'dark' ? 0.4 : 0.2,
    transition: 'opacity 0.3s ease',
    borderRadius: '50%',
  },
  logoContainer: {
    position: 'relative',
    display: 'flex',
    zIndex: 1,
  },
}));

const LogoIcon = () => {
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
      </Box>
    </div>
  );
};

export default LogoIcon;
