import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'
import './favicon.svg'
import { ThemeProvider, createTheme, CssBaseline } from '@mui/material'

const theme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#60a5fa' },
    secondary: { main: '#22c55e' },
    error: { main: '#ef4444' },
    background: { default: '#0b1020', paper: '#12172a' },
    text: { primary: '#e6e9f5', secondary: '#a6accd' },
    divider: '#1f2542'
  },
})

const el = document.getElementById('root')!
createRoot(el).render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <App />
    </ThemeProvider>
  </React.StrictMode>
)
