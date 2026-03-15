import { makeStyles } from '@material-ui/core/styles';
import NoiseTexture from '../../assets/svgs/noise.svg';

const useStyles = makeStyles(theme => {
  const isDark = theme.palette.type === 'dark';
  return {
    root: {
      position: 'fixed',
      top: 0,
      left: 0,
      width: '100vw',
      height: '100vh',
      zIndex: -1,
      pointerEvents: 'none',
      backgroundColor: isDark ? '#0a0a0a' : '#fafafa', // Zinc-950 / Zinc-50 (COLORS_DARK vs COLORS_LIGHT.appBackground)
      overflow: 'hidden',
    },
    blobAmber: {
      position: 'absolute',
      top: '-20%',
      left: '-10%',
      width: '50%',
      height: '50%',
      borderRadius: '50%',
      backgroundColor: isDark ? 'rgba(245, 158, 11, 0.1)' : 'rgba(245, 158, 11, 0.2)', // Solar glow
      filter: 'blur(120px)',
      transform: 'translate3d(0, 0, 0)',
    },
    blobAccent: {
      position: 'absolute',
      bottom: '-10%',
      right: '-10%',
      width: '40%',
      height: '40%',
      borderRadius: '50%',
      backgroundColor: isDark ? 'rgba(79, 70, 229, 0.1)' : 'rgba(79, 70, 229, 0.2)',
      filter: 'blur(100px)',
      transform: 'translate3d(0, 0, 0)',
    },
    noiseLayer: {
      position: 'absolute',
      inset: 0,
      width: '100%',
      height: '100%',
      zIndex: 1,
      backgroundImage: `url(${NoiseTexture})`,
      backgroundRepeat: 'repeat',
      backgroundSize: '350px',
      opacity: 0.8,
      mixBlendMode: 'overlay',
    },
  };
});

export const HeliosBackground = () => {
  const classes = useStyles();

  return (
    <div className={classes.root} aria-hidden="true">
      <div className={classes.blobAmber} />
      <div className={classes.blobAccent} />
      <div className={classes.noiseLayer} />
    </div>
  );
};
