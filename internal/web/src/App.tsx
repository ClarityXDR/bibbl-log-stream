import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ThemeProvider, CssBaseline, Box, Typography, alpha } from '@mui/material';
import { darkTheme, lightTheme, colors } from './theme/theme2026';
import AppShell from './components/layout/AppShell';
import Dashboard from './components/dashboard/Dashboard';
import SourcesConfig from './components/SourcesConfig';
import TransformWorkbench from './components/TransformWorkbench';
import DestinationsConfig from './components/DestinationsConfig';
import SimpleSetupWizard from './components/SimpleSetupWizard';
import MetricsPage from './components/pages/MetricsPage';
import AzurePage from './components/pages/AzurePage';

type TabKey = 'home' | 'setup' | 'sources' | 'transform' | 'destinations' | 'logevents' | 'azure' | 'settings' | 'help';

function useFetcher<T>(url: string, intervalMs?: number) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  
  const fetcher = useMemo(
    () => async () => {
      try {
        setLoading(true);
        setError(null);
        const r = await fetch(url);
        if (!r.ok) throw new Error(`${r.status}`);
        const json = await r.json();
        const unwrapped: any = Array.isArray(json)
          ? json
          : Array.isArray(json?.items)
          ? json.items
          : json;
        setData(unwrapped);
      } catch (e: any) {
        setError(e?.message || 'error');
      } finally {
        setLoading(false);
      }
    },
    [url]
  );

  useEffect(() => {
    fetcher();
    if (!intervalMs) return;
    const id = setInterval(fetcher, intervalMs);
    return () => clearInterval(id);
  }, [fetcher, intervalMs]);

  return { data, error, loading, refresh: fetcher };
}

export default function App() {
  const [isDarkMode, setIsDarkMode] = useState(true);
  const [tab, setTab] = useState<TabKey>('home');
  const [filtersInitialSelected, setFiltersInitialSelected] = useState<string | undefined>(undefined);

  // Listen for navigation events from other components
  useEffect(() => {
    const handler = (e: Event) => {
      const ce = e as CustomEvent<{ file?: string }>;
      const file = ce.detail?.file;
      if (file) setFiltersInitialSelected(file);
      setTab('transform');
    };
    window.addEventListener('open-filters', handler as EventListener);
    return () => window.removeEventListener('open-filters', handler as EventListener);
  }, []);

  const handleThemeToggle = useCallback(() => {
    setIsDarkMode((prev) => !prev);
  }, []);

  const theme = isDarkMode ? darkTheme : lightTheme;

  const handleNavigate = useCallback((newTab: string) => {
    setTab(newTab as TabKey);
  }, []);

  const renderContent = () => {
    switch (tab) {
      case 'home':
        return <Dashboard onNavigate={handleNavigate} />;
      case 'setup':
        return <SimpleSetupWizard />;
      case 'sources':
        return <SourcesConfig />;
      case 'transform':
        return <TransformWorkbench filtersInitialSelected={filtersInitialSelected} />;
      case 'destinations':
        return <DestinationsConfig />;
      case 'logevents':
        return <MetricsPage />;
      case 'azure':
        return <AzurePage />;
      case 'settings':
        return <PlaceholderPage title="Settings" />;
      case 'help':
        return <PlaceholderPage title="Help & Documentation" />;
      default:
        return <Dashboard onNavigate={handleNavigate} />;
    }
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AppShell
        activeTab={tab}
        onTabChange={(newTab) => setTab(newTab as TabKey)}
        onThemeToggle={handleThemeToggle}
        isDarkMode={isDarkMode}
      >
        {renderContent()}
      </AppShell>
    </ThemeProvider>
  );
}

// Placeholder for pages not yet implemented
function PlaceholderPage({ title }: { title: string }) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        minHeight: '60vh',
        textAlign: 'center',
      }}
    >
      <Box
        sx={{
          width: 80,
          height: 80,
          borderRadius: 4,
          background: `linear-gradient(135deg, ${alpha(colors.primary[500], 0.2)} 0%, ${alpha(colors.primary[600], 0.1)} 100%)`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 36,
          mb: 3,
        }}
      >
        🚧
      </Box>
      <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>
        {title}
      </Typography>
      <Typography variant="body1" color="text.secondary">
        This page is coming soon.
      </Typography>
    </Box>
  );
}
