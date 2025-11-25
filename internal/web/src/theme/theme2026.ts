import { createTheme, alpha } from '@mui/material/styles';

// 2026 Design System - Modern, Clean, Professional
// Inspired by: Linear, Vercel, Raycast, Arc Browser

const colors = {
  // Primary gradient palette - Electric indigo to violet
  primary: {
    50: '#eef2ff',
    100: '#e0e7ff',
    200: '#c7d2fe',
    300: '#a5b4fc',
    400: '#818cf8',
    500: '#6366f1',
    600: '#4f46e5',
    700: '#4338ca',
    800: '#3730a3',
    900: '#312e81',
  },
  // Success - Emerald
  success: {
    50: '#ecfdf5',
    100: '#d1fae5',
    200: '#a7f3d0',
    300: '#6ee7b7',
    400: '#34d399',
    500: '#10b981',
    600: '#059669',
    700: '#047857',
    800: '#065f46',
    900: '#064e3b',
  },
  // Warning - Amber
  warning: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b',
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
  },
  // Error - Rose
  error: {
    50: '#fff1f2',
    100: '#ffe4e6',
    200: '#fecdd3',
    300: '#fda4af',
    400: '#fb7185',
    500: '#f43f5e',
    600: '#e11d48',
    700: '#be123c',
    800: '#9f1239',
    900: '#881337',
  },
  // Neutral - Slate
  slate: {
    50: '#f8fafc',
    100: '#f1f5f9',
    200: '#e2e8f0',
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748b',
    600: '#475569',
    700: '#334155',
    800: '#1e293b',
    900: '#0f172a',
    950: '#020617',
  },
};

// Semantic shadows for depth
const shadows = {
  xs: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
  sm: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
  xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
  '2xl': '0 25px 50px -12px rgb(0 0 0 / 0.25)',
  glow: '0 0 40px -10px',
  inner: 'inset 0 2px 4px 0 rgb(0 0 0 / 0.05)',
};

// Dark theme for 2026
export const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: {
      main: colors.primary[500],
      light: colors.primary[400],
      dark: colors.primary[600],
      contrastText: '#fff',
    },
    secondary: {
      main: colors.slate[400],
      light: colors.slate[300],
      dark: colors.slate[500],
    },
    success: {
      main: colors.success[500],
      light: colors.success[400],
      dark: colors.success[600],
    },
    warning: {
      main: colors.warning[500],
      light: colors.warning[400],
      dark: colors.warning[600],
    },
    error: {
      main: colors.error[500],
      light: colors.error[400],
      dark: colors.error[600],
    },
    background: {
      default: colors.slate[950],
      paper: colors.slate[900],
    },
    text: {
      primary: colors.slate[50],
      secondary: colors.slate[400],
      disabled: colors.slate[600],
    },
    divider: alpha(colors.slate[400], 0.12),
    action: {
      hover: alpha(colors.slate[400], 0.08),
      selected: alpha(colors.primary[500], 0.16),
      disabled: alpha(colors.slate[400], 0.3),
      focus: alpha(colors.primary[500], 0.12),
    },
  },
  typography: {
    fontFamily: '"Inter", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    h1: {
      fontSize: '2.5rem',
      fontWeight: 700,
      letterSpacing: '-0.025em',
      lineHeight: 1.2,
    },
    h2: {
      fontSize: '2rem',
      fontWeight: 700,
      letterSpacing: '-0.025em',
      lineHeight: 1.25,
    },
    h3: {
      fontSize: '1.5rem',
      fontWeight: 600,
      letterSpacing: '-0.02em',
      lineHeight: 1.3,
    },
    h4: {
      fontSize: '1.25rem',
      fontWeight: 600,
      letterSpacing: '-0.015em',
      lineHeight: 1.35,
    },
    h5: {
      fontSize: '1.125rem',
      fontWeight: 600,
      letterSpacing: '-0.01em',
      lineHeight: 1.4,
    },
    h6: {
      fontSize: '1rem',
      fontWeight: 600,
      letterSpacing: '0',
      lineHeight: 1.5,
    },
    subtitle1: {
      fontSize: '1rem',
      fontWeight: 500,
      lineHeight: 1.5,
      letterSpacing: '0',
    },
    subtitle2: {
      fontSize: '0.875rem',
      fontWeight: 500,
      lineHeight: 1.5,
      letterSpacing: '0.01em',
    },
    body1: {
      fontSize: '0.9375rem',
      lineHeight: 1.6,
      letterSpacing: '0',
    },
    body2: {
      fontSize: '0.875rem',
      lineHeight: 1.6,
      letterSpacing: '0',
    },
    button: {
      fontWeight: 500,
      letterSpacing: '0.01em',
      textTransform: 'none' as const,
    },
    caption: {
      fontSize: '0.75rem',
      lineHeight: 1.5,
      letterSpacing: '0.02em',
    },
    overline: {
      fontSize: '0.6875rem',
      fontWeight: 600,
      letterSpacing: '0.08em',
      textTransform: 'uppercase' as const,
    },
  },
  shape: {
    borderRadius: 12,
  },
  shadows: [
    'none',
    shadows.xs,
    shadows.sm,
    shadows.sm,
    shadows.md,
    shadows.md,
    shadows.md,
    shadows.lg,
    shadows.lg,
    shadows.lg,
    shadows.lg,
    shadows.xl,
    shadows.xl,
    shadows.xl,
    shadows.xl,
    shadows.xl,
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
    shadows['2xl'],
  ],
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          scrollbarWidth: 'thin',
          scrollbarColor: `${colors.slate[700]} transparent`,
          '&::-webkit-scrollbar': {
            width: 8,
            height: 8,
          },
          '&::-webkit-scrollbar-track': {
            background: 'transparent',
          },
          '&::-webkit-scrollbar-thumb': {
            background: colors.slate[700],
            borderRadius: 4,
            '&:hover': {
              background: colors.slate[600],
            },
          },
        },
        '*::selection': {
          background: alpha(colors.primary[500], 0.3),
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 10,
          padding: '10px 20px',
          fontSize: '0.875rem',
          fontWeight: 500,
          boxShadow: 'none',
          transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            boxShadow: 'none',
            transform: 'translateY(-1px)',
          },
          '&:active': {
            transform: 'translateY(0)',
          },
        },
        contained: {
          background: `linear-gradient(135deg, ${colors.primary[500]} 0%, ${colors.primary[600]} 100%)`,
          '&:hover': {
            background: `linear-gradient(135deg, ${colors.primary[400]} 0%, ${colors.primary[500]} 100%)`,
          },
        },
        containedSuccess: {
          background: `linear-gradient(135deg, ${colors.success[500]} 0%, ${colors.success[600]} 100%)`,
          '&:hover': {
            background: `linear-gradient(135deg, ${colors.success[400]} 0%, ${colors.success[500]} 100%)`,
          },
        },
        outlined: {
          borderColor: alpha(colors.slate[400], 0.3),
          '&:hover': {
            borderColor: colors.primary[500],
            backgroundColor: alpha(colors.primary[500], 0.08),
          },
        },
        text: {
          '&:hover': {
            backgroundColor: alpha(colors.slate[400], 0.08),
          },
        },
      },
      defaultProps: {
        disableElevation: true,
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          border: `1px solid ${alpha(colors.slate[400], 0.1)}`,
          background: `linear-gradient(180deg, ${alpha(colors.slate[800], 0.8)} 0%, ${alpha(colors.slate[900], 0.9)} 100%)`,
          backdropFilter: 'blur(20px)',
          transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            borderColor: alpha(colors.primary[500], 0.3),
            transform: 'translateY(-2px)',
            boxShadow: `0 20px 40px -15px ${alpha(colors.primary[500], 0.15)}`,
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          borderRadius: 16,
        },
        elevation1: {
          boxShadow: shadows.sm,
        },
        elevation2: {
          boxShadow: shadows.md,
        },
        elevation3: {
          boxShadow: shadows.lg,
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          fontWeight: 500,
          fontSize: '0.75rem',
          height: 28,
        },
        filled: {
          backgroundColor: alpha(colors.slate[400], 0.15),
          '&:hover': {
            backgroundColor: alpha(colors.slate[400], 0.25),
          },
        },
        colorPrimary: {
          background: `linear-gradient(135deg, ${alpha(colors.primary[500], 0.2)} 0%, ${alpha(colors.primary[600], 0.2)} 100%)`,
          border: `1px solid ${alpha(colors.primary[500], 0.3)}`,
          color: colors.primary[300],
        },
        colorSuccess: {
          background: `linear-gradient(135deg, ${alpha(colors.success[500], 0.2)} 0%, ${alpha(colors.success[600], 0.2)} 100%)`,
          border: `1px solid ${alpha(colors.success[500], 0.3)}`,
          color: colors.success[300],
        },
        colorError: {
          background: `linear-gradient(135deg, ${alpha(colors.error[500], 0.2)} 0%, ${alpha(colors.error[600], 0.2)} 100%)`,
          border: `1px solid ${alpha(colors.error[500], 0.3)}`,
          color: colors.error[300],
        },
        colorWarning: {
          background: `linear-gradient(135deg, ${alpha(colors.warning[500], 0.2)} 0%, ${alpha(colors.warning[600], 0.2)} 100%)`,
          border: `1px solid ${alpha(colors.warning[500], 0.3)}`,
          color: colors.warning[300],
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 10,
            backgroundColor: alpha(colors.slate[800], 0.5),
            transition: 'all 0.2s ease',
            '& fieldset': {
              borderColor: alpha(colors.slate[400], 0.15),
              transition: 'all 0.2s ease',
            },
            '&:hover fieldset': {
              borderColor: alpha(colors.slate[400], 0.3),
            },
            '&.Mui-focused fieldset': {
              borderColor: colors.primary[500],
              borderWidth: 1,
              boxShadow: `0 0 0 3px ${alpha(colors.primary[500], 0.15)}`,
            },
          },
        },
      },
    },
    MuiSelect: {
      styleOverrides: {
        root: {
          borderRadius: 10,
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: {
          borderRadius: 20,
          border: `1px solid ${alpha(colors.slate[400], 0.1)}`,
          background: `linear-gradient(180deg, ${colors.slate[800]} 0%, ${colors.slate[900]} 100%)`,
          backdropFilter: 'blur(20px)',
        },
      },
    },
    MuiDialogTitle: {
      styleOverrides: {
        root: {
          fontSize: '1.25rem',
          fontWeight: 600,
          padding: '24px 24px 16px',
        },
      },
    },
    MuiDialogContent: {
      styleOverrides: {
        root: {
          padding: '16px 24px',
        },
      },
    },
    MuiDialogActions: {
      styleOverrides: {
        root: {
          padding: '16px 24px 24px',
          gap: 12,
        },
      },
    },
    MuiTabs: {
      styleOverrides: {
        root: {
          minHeight: 44,
        },
        indicator: {
          height: 2,
          borderRadius: 1,
          background: `linear-gradient(90deg, ${colors.primary[500]} 0%, ${colors.primary[400]} 100%)`,
        },
      },
    },
    MuiTab: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 500,
          fontSize: '0.875rem',
          minHeight: 44,
          padding: '12px 16px',
          borderRadius: 10,
          transition: 'all 0.2s ease',
          '&:hover': {
            backgroundColor: alpha(colors.slate[400], 0.08),
          },
          '&.Mui-selected': {
            color: colors.primary[400],
          },
        },
      },
    },
    MuiSwitch: {
      styleOverrides: {
        root: {
          width: 44,
          height: 24,
          padding: 0,
        },
        switchBase: {
          padding: 2,
          '&.Mui-checked': {
            transform: 'translateX(20px)',
            '& + .MuiSwitch-track': {
              backgroundColor: colors.primary[500],
              opacity: 1,
            },
          },
        },
        thumb: {
          width: 20,
          height: 20,
          boxShadow: shadows.sm,
        },
        track: {
          borderRadius: 12,
          backgroundColor: colors.slate[600],
          opacity: 1,
        },
      },
    },
    MuiLinearProgress: {
      styleOverrides: {
        root: {
          height: 6,
          borderRadius: 3,
          backgroundColor: alpha(colors.slate[400], 0.15),
        },
        bar: {
          borderRadius: 3,
          background: `linear-gradient(90deg, ${colors.primary[500]} 0%, ${colors.primary[400]} 100%)`,
        },
      },
    },
    MuiStepper: {
      styleOverrides: {
        root: {
          padding: 0,
        },
      },
    },
    MuiStepLabel: {
      styleOverrides: {
        label: {
          fontWeight: 500,
          '&.Mui-active': {
            fontWeight: 600,
          },
        },
      },
    },
    MuiStepIcon: {
      styleOverrides: {
        root: {
          fontSize: 28,
          '&.Mui-active': {
            color: colors.primary[500],
          },
          '&.Mui-completed': {
            color: colors.success[500],
          },
        },
      },
    },
    MuiAlert: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          border: '1px solid',
        },
        standardSuccess: {
          backgroundColor: alpha(colors.success[500], 0.1),
          borderColor: alpha(colors.success[500], 0.3),
          color: colors.success[300],
          '& .MuiAlert-icon': {
            color: colors.success[400],
          },
        },
        standardError: {
          backgroundColor: alpha(colors.error[500], 0.1),
          borderColor: alpha(colors.error[500], 0.3),
          color: colors.error[300],
          '& .MuiAlert-icon': {
            color: colors.error[400],
          },
        },
        standardWarning: {
          backgroundColor: alpha(colors.warning[500], 0.1),
          borderColor: alpha(colors.warning[500], 0.3),
          color: colors.warning[300],
          '& .MuiAlert-icon': {
            color: colors.warning[400],
          },
        },
        standardInfo: {
          backgroundColor: alpha(colors.primary[500], 0.1),
          borderColor: alpha(colors.primary[500], 0.3),
          color: colors.primary[300],
          '& .MuiAlert-icon': {
            color: colors.primary[400],
          },
        },
      },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: colors.slate[700],
          borderRadius: 8,
          fontSize: '0.75rem',
          fontWeight: 500,
          padding: '8px 12px',
          boxShadow: shadows.lg,
        },
        arrow: {
          color: colors.slate[700],
        },
      },
    },
    MuiIconButton: {
      styleOverrides: {
        root: {
          borderRadius: 10,
          transition: 'all 0.2s ease',
          '&:hover': {
            backgroundColor: alpha(colors.slate[400], 0.1),
          },
        },
      },
    },
    MuiDivider: {
      styleOverrides: {
        root: {
          borderColor: alpha(colors.slate[400], 0.1),
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: `1px solid ${alpha(colors.slate[400], 0.1)}`,
          padding: '16px',
        },
        head: {
          fontWeight: 600,
          color: colors.slate[300],
          backgroundColor: alpha(colors.slate[800], 0.5),
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          transition: 'background-color 0.15s ease',
          '&:hover': {
            backgroundColor: alpha(colors.slate[700], 0.3),
          },
        },
      },
    },
  },
});

// Light theme for 2026
export const lightTheme = createTheme({
  ...darkTheme,
  palette: {
    mode: 'light',
    primary: {
      main: colors.primary[600],
      light: colors.primary[500],
      dark: colors.primary[700],
      contrastText: '#fff',
    },
    secondary: {
      main: colors.slate[600],
      light: colors.slate[500],
      dark: colors.slate[700],
    },
    success: {
      main: colors.success[600],
      light: colors.success[500],
      dark: colors.success[700],
    },
    warning: {
      main: colors.warning[600],
      light: colors.warning[500],
      dark: colors.warning[700],
    },
    error: {
      main: colors.error[600],
      light: colors.error[500],
      dark: colors.error[700],
    },
    background: {
      default: colors.slate[50],
      paper: '#ffffff',
    },
    text: {
      primary: colors.slate[900],
      secondary: colors.slate[600],
      disabled: colors.slate[400],
    },
    divider: alpha(colors.slate[900], 0.08),
  },
  components: {
    ...darkTheme.components,
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          border: `1px solid ${alpha(colors.slate[900], 0.08)}`,
          background: '#ffffff',
          boxShadow: shadows.sm,
          transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          '&:hover': {
            borderColor: alpha(colors.primary[500], 0.3),
            transform: 'translateY(-2px)',
            boxShadow: shadows.lg,
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          borderRadius: 16,
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 10,
            backgroundColor: colors.slate[50],
            '& fieldset': {
              borderColor: alpha(colors.slate[900], 0.15),
            },
            '&:hover fieldset': {
              borderColor: alpha(colors.slate[900], 0.3),
            },
            '&.Mui-focused fieldset': {
              borderColor: colors.primary[600],
              boxShadow: `0 0 0 3px ${alpha(colors.primary[600], 0.15)}`,
            },
          },
        },
      },
    },
    MuiDialog: {
      styleOverrides: {
        paper: {
          borderRadius: 20,
          border: `1px solid ${alpha(colors.slate[900], 0.08)}`,
          boxShadow: shadows['2xl'],
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        filled: {
          backgroundColor: colors.slate[100],
          '&:hover': {
            backgroundColor: colors.slate[200],
          },
        },
        colorPrimary: {
          background: alpha(colors.primary[600], 0.1),
          border: `1px solid ${alpha(colors.primary[600], 0.3)}`,
          color: colors.primary[700],
        },
        colorSuccess: {
          background: alpha(colors.success[600], 0.1),
          border: `1px solid ${alpha(colors.success[600], 0.3)}`,
          color: colors.success[700],
        },
        colorError: {
          background: alpha(colors.error[600], 0.1),
          border: `1px solid ${alpha(colors.error[600], 0.3)}`,
          color: colors.error[700],
        },
        colorWarning: {
          background: alpha(colors.warning[600], 0.1),
          border: `1px solid ${alpha(colors.warning[600], 0.3)}`,
          color: colors.warning[700],
        },
      },
    },
    MuiAlert: {
      styleOverrides: {
        standardSuccess: {
          backgroundColor: alpha(colors.success[600], 0.1),
          borderColor: alpha(colors.success[600], 0.3),
          color: colors.success[800],
          '& .MuiAlert-icon': {
            color: colors.success[600],
          },
        },
        standardError: {
          backgroundColor: alpha(colors.error[600], 0.1),
          borderColor: alpha(colors.error[600], 0.3),
          color: colors.error[800],
          '& .MuiAlert-icon': {
            color: colors.error[600],
          },
        },
        standardWarning: {
          backgroundColor: alpha(colors.warning[600], 0.1),
          borderColor: alpha(colors.warning[600], 0.3),
          color: colors.warning[800],
          '& .MuiAlert-icon': {
            color: colors.warning[600],
          },
        },
        standardInfo: {
          backgroundColor: alpha(colors.primary[600], 0.1),
          borderColor: alpha(colors.primary[600], 0.3),
          color: colors.primary[800],
          '& .MuiAlert-icon': {
            color: colors.primary[600],
          },
        },
      },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: {
          backgroundColor: colors.slate[800],
          borderRadius: 8,
          fontSize: '0.75rem',
          fontWeight: 500,
          padding: '8px 12px',
          boxShadow: shadows.lg,
        },
        arrow: {
          color: colors.slate[800],
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: `1px solid ${alpha(colors.slate[900], 0.08)}`,
        },
        head: {
          fontWeight: 600,
          color: colors.slate[700],
          backgroundColor: colors.slate[50],
        },
      },
    },
    MuiTableRow: {
      styleOverrides: {
        root: {
          '&:hover': {
            backgroundColor: colors.slate[50],
          },
        },
      },
    },
  },
});

// Export colors for use in components
export { colors, shadows };

// Default export
export default darkTheme;
