import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'

// Note: ThemeProvider and CssBaseline are now handled in App.tsx
// for dynamic theme switching support

const el = document.getElementById('root')!
createRoot(el).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
)
