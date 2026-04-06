import { createTheme } from '@mui/material/styles';

export const theme = createTheme({
  cssVariables: {
    colorSchemeSelector: 'data-theme',
  },
  colorSchemes: {
    light: {
      palette: {
        primary: { main: '#1976d2' },
        secondary: { main: '#424242' },
        success: { main: '#2e7d32' },
        error: { main: '#d32f2f' },
        warning: { main: '#ed6c02' },
        info: { main: '#0288d1' },
        background: { default: '#fafafa', paper: '#ffffff' },
      },
    },
    dark: {
      palette: {
        primary: { main: '#90caf9' },
        secondary: { main: '#90a4ae' },
        success: { main: '#a5d6a7' },
        error: { main: '#ef9a9a' },
        warning: { main: '#ffcc80' },
        info: { main: '#81d4fa' },
        background: { default: '#121212', paper: '#1e1e1e' },
      },
    },
  },
  typography: {
    fontFamily: 'Roboto, Arial, sans-serif',
    h6: { fontWeight: 600 },
    h5: { fontWeight: 500 },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: (theme) => ({
        html: { scrollbarGutter: 'stable' },
        body: {
          ...theme.applyStyles('dark', {
            scrollbarColor: '#555 #1e1e1e',
          }),
        },
      }),
    },
    MuiChip: {
      styleOverrides: {
        root: { fontWeight: 500 },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        head: ({ theme }) => ({
          fontWeight: 600,
          backgroundColor: '#f5f5f5',
          ...theme.applyStyles('dark', {
            backgroundColor: 'rgba(255, 255, 255, 0.05)',
          }),
        }),
      },
    },
    MuiContainer: {
      styleOverrides: {
        root: { paddingTop: '16px', paddingBottom: '16px' },
      },
    },
  },
});
