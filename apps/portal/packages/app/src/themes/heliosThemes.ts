import {
  createBaseThemeOptions,
  createUnifiedTheme,
  genPageTheme,
  palettes,
  shapes,
} from '@backstage/theme';
import { alpha } from '@mui/material/styles';

// --- HELIOS CONSTANTS ---
const COLORS_LIGHT = {
  // Base Surfaces
  appBackground: '#fafafa', // Zinc 50
  panelSurface: '#ffffff', // White
  glass: 'rgba(255, 255, 255, 0.7)', // Light frosted glass

  // Primary Brand (Solar)
  solar500: '#f59e0b', // Amber 500 (Core brand color)
  solar600: '#d97706', // Amber 600 (Better for text/hover contrast on light)
  solar700: '#b45309', // Amber 700 (Darker states)
  solarGlow: 'rgba(245, 158, 11, 0.2)', // Softer glow for light backgrounds

  // Borders & Text
  borderSubtle: 'rgba(0, 0, 0, 0.08)', // Equivalent to Zinc 200/300
  textPrimary: '#09090b', // Zinc 950
  textSecondary: '#52525b', // Zinc 600

  // Semantic (Shifted to 600-level for legibility on white)
  emerald: '#059669', // Emerald 600
  rose: '#e11d48', // Rose 600
  amber: '#d97706', // Amber 600
  cyan: '#0891b2', // Cyan 600
};

const COLORS_DARK = {
  voidBlack: '#000000',
  panelSurface: '#09090b',
  glass: 'rgba(10, 10, 10, 0.6)',
  solar500: '#f59e0b', // Amber 500
  solar400: '#fbbf24',
  solar700: '#b45309',
  solarGlow: 'rgba(245, 158, 11, 0.4)',
  borderSubtle: 'rgba(255, 255, 255, 0.08)',
  textPrimary: '#ffffff',
  textSecondary: '#a1a1aa', // Zinc 400
  // Semantic
  emerald: '#10b981',
  rose: '#f43f5e',
  amber: '#f59e0b',
  cyan: '#06b6d4',
};

const FONTS = {
  sans: '"Inter", "San Francisco", "Helvetica Neue", sans-serif',
  mono: '"JetBrains Mono", "Roboto Mono", monospace',
};

// Color Palette Definitions
const heliosPaletteDark = {
  ...palettes.dark,
  primary: {
    main: COLORS_DARK.solar500,
    light: COLORS_DARK.solar400,
    dark: COLORS_DARK.solar700,
    contrastText: '#000000',
  },
  background: {
    default: 'transparent', // Void Black
    paper: COLORS_DARK.panelSurface,
  },
  text: {
    primary: COLORS_DARK.textPrimary,
    secondary: COLORS_DARK.textSecondary,
  },
  status: {
    ok: COLORS_DARK.emerald,
    warning: COLORS_DARK.amber,
    error: COLORS_DARK.rose,
    running: COLORS_DARK.cyan,
    pending: COLORS_DARK.cyan,
    aborted: COLORS_DARK.textSecondary,
  },
  navigation: {
    background: COLORS_DARK.glass,
    indicator: COLORS_DARK.solar500,
    color: COLORS_DARK.textSecondary,
    selectedColor: COLORS_DARK.solar400,
  },
};

// Light Color Palette Definitions
const heliosPaletteLight = {
  ...palettes.light,
  primary: {
    main: COLORS_LIGHT.solar500,
    // For light mode, you often want a darker color for hover states
    light: COLORS_LIGHT.solar600,
    dark: COLORS_LIGHT.solar700,
    // Contrast text on a Solar 500 background should remain dark
    contrastText: '#09090b',
  },
  background: {
    default: COLORS_LIGHT.appBackground,
    paper: COLORS_LIGHT.panelSurface,
  },
  text: {
    primary: COLORS_LIGHT.textPrimary,
    secondary: COLORS_LIGHT.textSecondary,
  },
  status: {
    ok: COLORS_LIGHT.emerald,
    warning: COLORS_LIGHT.amber,
    error: COLORS_LIGHT.rose,
    running: COLORS_LIGHT.cyan,
    pending: COLORS_LIGHT.cyan,
    aborted: COLORS_LIGHT.textSecondary,
  },
  navigation: {
    background: COLORS_LIGHT.glass,
    indicator: COLORS_LIGHT.solar500,
    color: COLORS_LIGHT.textSecondary,
    selectedColor: COLORS_LIGHT.solar700,
  },
};

const heliosTypography = {
  fontFamily: FONTS.sans, // Base font for body

  // Display XL
  h1: { fontFamily: FONTS.sans, fontWeight: 700, letterSpacing: '-0.02em' },

  // Heading L
  h2: { fontFamily: FONTS.mono, fontWeight: 700, letterSpacing: '-0.03em' },

  // Heading M (Can be Sans or Mono, defaulting to Sans here for readability)
  h3: { fontFamily: FONTS.sans, fontWeight: 700, letterSpacing: '-0.01em' },

  // Lower level headings (extrapolated as Sans for standard UI text)
  h4: { fontFamily: FONTS.sans, fontWeight: 700 },
  h5: { fontFamily: FONTS.sans, fontWeight: 700 },
  h6: { fontFamily: FONTS.sans, fontWeight: 700 },

  // System / Technical elements
  button: { fontFamily: FONTS.mono, fontWeight: 700 },

  // Label S
  caption: {
    fontFamily: FONTS.mono,
    fontWeight: 700,
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
  },

  // Micro
  overline: {
    fontFamily: FONTS.mono,
    fontWeight: 400,
    textTransform: 'uppercase',
    letterSpacing: '0.05em',
  },
};

export const darkTheme = createUnifiedTheme({
  ...createBaseThemeOptions({
    palette: heliosPaletteDark,
  }),
  typography: heliosTypography,
  defaultPageTheme: 'home',
  pageTheme: {
    home: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
    documentation: genPageTheme({
      colors: [COLORS_DARK.cyan, COLORS_DARK.emerald],
      shape: shapes.wave,
    }),
    tool: genPageTheme({
      colors: [COLORS_DARK.rose, COLORS_DARK.amber],
      shape: shapes.round,
    }),
    service: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.emerald],
      shape: shapes.wave,
    }),
    website: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
    library: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
    other: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
    app: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
    apis: genPageTheme({
      colors: [COLORS_DARK.solar500, COLORS_DARK.solar400],
      shape: shapes.wave,
    }),
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: 'transparent',
          scrollbarColor: '#333 #000',
        },
      },
    },
    // GLASSMORPHISM PANELS
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundColor: COLORS_DARK.panelSurface,
          backgroundImage: 'none',
          backdropFilter: 'blur(12px)',
          border: `1px solid ${COLORS_DARK.borderSubtle}`,
          borderRadius: '4px',
          '&.glass-panel': {
            // Custom class for deeper glass
            backgroundColor: COLORS_DARK.glass,
          },
        },
        elevation1: { boxShadow: 'none' },
        elevation2: { boxShadow: 'none' },
        elevation3: { boxShadow: '0px 0px 20px 0px rgba(0,0,0,0.5)' },
      },
    },
    // CARDS (Sharp Technical Look)
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundColor: 'rgba(9, 9, 11, 0.7)',
          backdropFilter: 'blur(12px)',
          backgroundImage: 'none',
          border: '1px solid rgba(255, 255, 255, 0.08)',
          borderRadius: '4px',
          boxShadow: 'none',
          transition: 'border-color 0.3s ease',

          '&:hover': {
            borderColor: 'rgba(255, 255, 255, 0.2)',
          },
        },
      },
    },
    MuiTypography: {
      styleOverrides: {
        h5: {
          fontFamily: FONTS.sans,
        },
      },
    },
    // BUTTONS (Solar Flare)
    MuiButton: {
      defaultProps: {
        // Disabling the default MUI ripple
        disableRipple: true,
      },
      styleOverrides: {
        root: {
          borderRadius: '2px',
          textTransform: 'uppercase',
          fontFamily: 'monospace',
          fontWeight: 'bold',
          letterSpacing: '0.05em',
          overflow: 'hidden',
          transition: 'all 0.15s ease',
          '&:active': {
            transform: 'scale(0.97)',
          },
          '&.Mui-focusVisible': {
            outline: '2px solid #f59e0b',
            outlineOffset: '2px',
          },
        },

        /* --- PRIMARY VARIANT --- */
        containedPrimary: {
          backgroundColor: '#f59e0b', // amber-500
          color: '#000000',
          boxShadow: '0 0 20px rgba(245,158,11,0.4)', // The amber glow

          '&:hover': {
            backgroundColor: '#fbbf24', // amber-400
            boxShadow: '0 0 30px rgba(245, 158, 11, 0.6)',
          },

          // Active state (click)
          '&:active': {
            backgroundColor: '#d97706', // amber-600
            boxShadow: '0 0 10px rgba(245,158,11,0.4)',
          },

          // The CSS-only Hover Shine Effect
          '&::after': {
            content: '""',
            position: 'absolute',
            top: 0,
            left: '-100%',
            width: '50%',
            height: '100%',
            transform: 'skewX(-12deg)',
            background:
              'linear-gradient(to right, transparent, rgba(255,255,255,0.2))',
            transition: 'left 0.7s ease-out',
            zIndex: 0,
          },

          '&:hover::after': {
            left: '200%',
          },

          '& .MuiButton-startIcon, & .MuiButton-endIcon': {
            zIndex: 1, // Keep icons above the shine
            transition: 'transform 0.2s ease',
          },

          '&:hover .MuiButton-startIcon, &:hover .MuiButton-endIcon': {
            transform: 'scale(1.1)',
          },

          '&:active .MuiButton-startIcon, &:active .MuiButton-endIcon': {
            transform: 'scale(0.95)',
          },
        },

        /* --- SECONDARY VARIANT --- */
        outlinedSecondary: {
          backgroundColor: '#18181b', // zinc-900
          borderColor: '#3f3f46', // zinc-700
          color: '#f4f4f5', // zinc-100
          borderWidth: '1px',
          '&:hover': {
            backgroundColor: '#27272a', // zinc-800
            borderColor: '#71717a', // zinc-500
            borderWidth: '1px', // Prevents layout shift
          },
          '&:active': {
            backgroundColor: '#09090b', // zinc-950
            borderColor: '#52525b', // zinc-600
          },
        },

        /* --- NEON VARIANT --- */
        outlinedPrimary: {
          backgroundColor: 'transparent',
          borderColor: 'rgba(245,158,11,0.5)', // amber-500/50
          color: '#f59e0b', // amber-500
          boxShadow: 'inset 0 0 10px rgba(245,158,11,0.1)',
          borderWidth: '1px',
          '&:hover': {
            backgroundColor: 'rgba(245,158,11,0.1)', // amber-500/10
            borderColor: 'rgba(245,158,11,0.8)',
            boxShadow: 'inset 0 0 20px rgba(245,158,11,0.2)',
            borderWidth: '1px',
          },
          '&:active': {
            backgroundColor: 'rgba(245,158,11,0.2)', // amber-500/20
            boxShadow: 'none', // Kills the glow on click
          },
        },

        /* --- GHOST VARIANT --- */
        text: {
          backgroundColor: 'transparent',
          color: '#a1a1aa', // zinc-400
          '&:hover': {
            backgroundColor: 'rgba(255, 255, 255, 0.05)',
            color: '#ffffff',
          },
          '&:active': {
            backgroundColor: 'rgba(255, 255, 255, 0.1)',
            color: '#e4e4e7', // zinc-200
          },
        },
      },
    },
    // CHIPS/BADGES (Status Glows)
    MuiChip: {
      styleOverrides: {
        // "Healthy" status mapping (usually "success" or "primary" in standard MUI)
        filledSuccess: {
          backgroundColor: 'rgba(16, 185, 129, 0.1)', // bg-emerald-500/10
          color: '#34d399', // text-emerald-400
          border: '1px solid rgba(16, 185, 129, 0.3)',
          boxShadow: '0 0 10px rgba(16,185,129,0.2)',
        },
        // "Warning" status mapping
        filledWarning: {
          backgroundColor: 'rgba(245, 158, 11, 0.1)', // bg-amber-500/10
          color: '#fbbf24', // text-amber-400
          border: '1px solid rgba(245, 158, 11, 0.3)',
          boxShadow: '0 0 10px rgba(245,158,11,0.2)',
        },
        // "Error" status mapping
        filledError: {
          backgroundColor: 'rgba(244, 63, 94, 0.1)', // bg-rose-500/10
          color: '#fb7185', // text-rose-400
          border: '1px solid rgba(244, 63, 94, 0.3)',
          boxShadow: '0 0 10px rgba(244,63,94,0.2)',
        },
      },
    },
    // TABLES
    MuiTableHead: {
      styleOverrides: {
        root: {
          backgroundColor: 'rgba(255,255,255,0.02)',
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        head: {
          color: '#a1a1aa',
          fontFamily: '"JetBrains Mono", monospace',
          textTransform: 'uppercase',
          fontSize: '11px',
        },
        body: {
          borderBottom: '1px solid rgba(255,255,255,0.05)',
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          borderBottom: '1px solid rgba(255, 255, 255, 0.05)',
          '&:hover': {
            backgroundColor: 'rgba(255, 255, 255, 0.05) !important', // hover:bg-white/5
          },
        },
      },
    },
    // BACKSTAGE HEADER
    BackstageHeader: {
      styleOverrides: {
        header: {
          backgroundImage: 'none',
          boxShadow: 'none',
          // borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
          backgroundColor: 'transparent',
          padding: '48px 48px 24px 48px',
        },
        title: {
          color: COLORS_DARK.textPrimary,
          fontSize: '36px',
          fontFamily: FONTS.sans,
        },
        subtitle: {
          color: '#a1a1aa',
        },
      },
    },
    BackstageHeaderTabs: {
      styleOverrides: {
        tabsWrapper: {
          paddingLeft: '48px',
          paddingRight: '48px',
          borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
          backgroundColor: 'transparent',
        },
      },
    },
    BackstageContent: {
      styleOverrides: {
        root: {
          padding: '48px 48px',
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          backgroundColor: 'rgba(24, 24, 27, 0.5)', // zinc-900/50
          '& fieldset': { borderColor: '#27272a' }, // zinc-800
          '&:hover fieldset': { borderColor: '#f59e0b' }, // hover:border-amber-500
          '&.Mui-focused fieldset': {
            borderColor: '#f59e0b',
            boxShadow: '0 0 10px rgba(245, 158, 11, 0.2)',
          },
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: alpha(COLORS_DARK.voidBlack, 0.95),
          borderRight: `1px solid ${COLORS_DARK.borderSubtle}`,
          backdropFilter: 'blur(12px)',
        },
      },
    },
    MuiTabs: {
      styleOverrides: {
        indicator: {
          backgroundColor: COLORS_DARK.solar500,
          boxShadow: `0 0 10px ${COLORS_DARK.solar500}`,
          height: '3px',
        },
      },
    },
  },
});

export const lightTheme = createUnifiedTheme({
  ...createBaseThemeOptions({
    palette: heliosPaletteLight,
  }),
  typography: heliosTypography,
  defaultPageTheme: 'home',
  pageTheme: {
    home: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
    documentation: genPageTheme({
      colors: [COLORS_LIGHT.cyan, COLORS_LIGHT.emerald],
      shape: shapes.wave,
    }),
    tool: genPageTheme({
      colors: [COLORS_LIGHT.rose, COLORS_LIGHT.amber],
      shape: shapes.round,
    }),
    service: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.emerald],
      shape: shapes.wave,
    }),
    website: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
    library: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
    other: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
    app: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
    apis: genPageTheme({
      colors: [COLORS_LIGHT.solar500, COLORS_LIGHT.solar600],
      shape: shapes.wave,
    }),
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: 'transparent', // Usually set by the app background layer
          scrollbarColor: '#d4d4d8 #f4f4f5', // Zinc 300 on Zinc 100
        },
      },
    },
    // GLASSMORPHISM PANELS
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundColor: COLORS_LIGHT.panelSurface,
          backgroundImage: 'none',
          backdropFilter: 'blur(12px)',
          border: `1px solid ${COLORS_LIGHT.borderSubtle}`,
          borderRadius: '4px',
          '&.glass-panel': {
            // Custom class for frosted glass
            backgroundColor: COLORS_LIGHT.glass,
            boxShadow:
              '0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -1px rgba(0, 0, 0, 0.03)',
          },
        },
        elevation1: { boxShadow: 'none' },
        elevation2: { boxShadow: 'none' },
        elevation3: { boxShadow: '0px 4px 20px 0px rgba(0,0,0,0.08)' },
      },
    },
    // CARDS (Sharp Technical Look)
    MuiCard: {
      styleOverrides: {
        root: {
          backgroundColor: '#ffffff',
          backdropFilter: 'none', // Removed heavy blur for base light cards to keep it clean
          backgroundImage: 'none',
          border: '1px solid #e4e4e7', // Zinc 200
          borderRadius: '4px',
          boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.05)',
          transition: 'border-color 0.3s ease, box-shadow 0.3s ease',

          '&:hover': {
            borderColor: '#fcd34d', // Amber 300
            boxShadow: '0 4px 6px -1px rgba(245, 158, 11, 0.1)',
          },
        },
      },
    },
    // BUTTONS (Solar Flare)
    MuiButton: {
      defaultProps: {
        disableRipple: true,
      },
      styleOverrides: {
        root: {
          borderRadius: '2px',
          textTransform: 'uppercase',
          fontFamily: 'monospace',
          fontWeight: 'bold',
          letterSpacing: '0.05em',
          overflow: 'hidden',
          transition: 'all 0.15s ease',
          '&:active': {
            transform: 'scale(0.97)',
          },
          '&.Mui-focusVisible': {
            outline: '2px solid #f59e0b',
            outlineOffset: '2px',
          },
        },

        /* --- PRIMARY VARIANT --- */
        containedPrimary: {
          backgroundColor: '#f59e0b', // amber-500
          color: '#09090b', // zinc-950 for high contrast
          boxShadow: '0 4px 14px rgba(245,158,11,0.3)', // Soft downward glow

          '&:hover': {
            backgroundColor: '#fbbf24', // amber-400
            boxShadow: '0 6px 20px rgba(245, 158, 11, 0.4)',
          },

          '&:active': {
            backgroundColor: '#d97706', // amber-600
            boxShadow: '0 2px 8px rgba(245,158,11,0.4)',
          },

          // The CSS-only Hover Shine Effect
          '&::after': {
            content: '""',
            position: 'absolute',
            top: 0,
            left: '-100%',
            width: '50%',
            height: '100%',
            transform: 'skewX(-12deg)',
            background:
              'linear-gradient(to right, transparent, rgba(255,255,255,0.4))', // Slightly brighter shine for light mode
            transition: 'left 0.7s ease-out',
            zIndex: 0,
          },

          '&:hover::after': {
            left: '200%',
          },

          '& .MuiButton-startIcon, & .MuiButton-endIcon': {
            zIndex: 1,
            transition: 'transform 0.2s ease',
          },

          '&:hover .MuiButton-startIcon, &:hover .MuiButton-endIcon': {
            transform: 'scale(1.1)',
          },

          '&:active .MuiButton-startIcon, &:active .MuiButton-endIcon': {
            transform: 'scale(0.95)',
          },
        },

        /* --- SECONDARY VARIANT --- */
        outlinedSecondary: {
          backgroundColor: '#ffffff',
          borderColor: '#d4d4d8', // zinc-300
          color: '#27272a', // zinc-800
          borderWidth: '1px',
          boxShadow: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
          '&:hover': {
            backgroundColor: '#fafafa', // zinc-50
            borderColor: '#a1a1aa', // zinc-400
            borderWidth: '1px',
          },
          '&:active': {
            backgroundColor: '#f4f4f5', // zinc-100
            borderColor: '#71717a', // zinc-500
          },
        },

        /* --- NEON VARIANT --- */
        outlinedPrimary: {
          backgroundColor: '#fffbeb', // amber-50
          borderColor: '#fbbf24', // amber-400
          color: '#b45309', // amber-700
          boxShadow: '0 0 10px rgba(245,158,11,0.1)',
          borderWidth: '1px',
          '&:hover': {
            backgroundColor: '#fef3c7', // amber-100
            borderColor: '#f59e0b', // amber-500
            boxShadow: '0 0 16px rgba(245,158,11,0.2)',
            borderWidth: '1px',
          },
          '&:active': {
            backgroundColor: '#fde68a', // amber-200
            boxShadow: 'none',
          },
        },

        /* --- GHOST VARIANT --- */
        text: {
          backgroundColor: 'transparent',
          color: '#52525b', // zinc-600
          '&:hover': {
            backgroundColor: '#f4f4f5', // zinc-100
            color: '#09090b', // zinc-950
          },
          '&:active': {
            backgroundColor: '#e4e4e7', // zinc-200
            color: '#18181b', // zinc-900
          },
        },
      },
    },
    // CHIPS/BADGES (Status Glows updated for light bg)
    MuiChip: {
      styleOverrides: {
        filledSuccess: {
          backgroundColor: '#ecfdf5', // emerald-50
          color: '#047857', // emerald-700
          border: '1px solid #6ee7b7', // emerald-300
          boxShadow: 'none', // Removed heavy glow for cleaner look on light, relying on border
        },
        filledWarning: {
          backgroundColor: '#fffbeb', // amber-50
          color: '#b45309', // amber-700
          border: '1px solid #fcd34d', // amber-300
          boxShadow: 'none',
        },
        filledError: {
          backgroundColor: '#fff1f2', // rose-50
          color: '#be123c', // rose-700
          border: '1px solid #fda4af', // rose-300
          boxShadow: 'none',
        },
      },
    },
    // TABLES
    MuiTableHead: {
      styleOverrides: {
        root: {
          backgroundColor: '#fafafa', // zinc-50
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        head: {
          color: '#71717a', // zinc-500
          fontFamily: '"JetBrains Mono", monospace',
          textTransform: 'uppercase',
          fontSize: '11px',
        },
        body: {
          borderBottom: '1px solid #e4e4e7', // zinc-200
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          borderBottom: '1px solid #e4e4e7',
          '&:hover': {
            backgroundColor: '#f4f4f5 !important', // hover:bg-zinc-100
          },
        },
      },
    },
    // BACKSTAGE HEADER
    BackstageHeader: {
      styleOverrides: {
        header: {
          backgroundImage: 'none',
          boxShadow: 'none',
          backgroundColor: 'transparent',
          padding: '48px 48px 24px 48px',
        },
        title: {
          color: COLORS_LIGHT.textPrimary,
          fontSize: '36px',
          fontFamily: FONTS.sans,
        },
        type: {
          color: COLORS_LIGHT.textSecondary,
          fontSize: '16px',
          fontFamily: FONTS.mono,
        },
        subtitle: {
          color: COLORS_LIGHT.textSecondary,
        },
      },
    },
    BackstageHeaderLabel: {
      styleOverrides: {
        label: {
          color: COLORS_LIGHT.textPrimary,
          fontSize: '16px',
          fontFamily: FONTS.sans,
        },
        value: {
          color: COLORS_LIGHT.textSecondary,
          fontSize: '14px',
          fontFamily: FONTS.sans,
        },
      },
    },
    BackstageHeaderTabs: {
      styleOverrides: {
        tabsWrapper: {
          borderBottom: `1px solid ${COLORS_LIGHT.borderSubtle}`,
          backgroundColor: 'transparent',
          fontFamily: FONTS.sans,
          paddingRight: '48px',
          paddingLeft: '48px',
        },
      },
    },
    // FORMS
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          backgroundColor: '#ffffff',
          '& fieldset': { borderColor: '#d4d4d8' }, // zinc-300
          '&:hover fieldset': { borderColor: '#f59e0b' }, // hover:border-amber-500
          '&.Mui-focused fieldset': {
            borderColor: '#f59e0b',
            boxShadow: '0 0 0 2px rgba(245, 158, 11, 0.2)', // Focus ring instead of heavy glow
          },
        },
      },
    },
    MuiDrawer: {
      styleOverrides: {
        paper: {
          backgroundColor: alpha(COLORS_LIGHT.appBackground, 0.95), // Zinc 50 base
          borderRight: `1px solid ${COLORS_LIGHT.borderSubtle}`,
          backdropFilter: 'blur(12px)',
        },
      },
    },
    MuiTabs: {
      styleOverrides: {
        indicator: {
          backgroundColor: COLORS_LIGHT.solar500,
          boxShadow: `0 -2px 10px rgba(245, 158, 11, 0.3)`, // Subtle upward glow
          height: '3px',
        },
      },
    },
  },
});
