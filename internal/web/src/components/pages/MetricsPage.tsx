import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Grid,
  Typography,
  Card,
  CardContent,
  Button,
  IconButton,
  Chip,
  Switch,
  FormControlLabel,
  TextField,
  alpha,
  Tabs,
  Tab,
  Tooltip,
  Skeleton,
} from '@mui/material';
import {
  Refresh,
  Speed,
  Timeline,
  Memory,
  Storage,
  CloudQueue,
  TrendingUp,
  Warning,
  OpenInNew,
} from '@mui/icons-material';
import { colors } from '../../theme/theme2026';

interface MetricCategory {
  name: string;
  metrics: MetricData[];
}

interface MetricData {
  name: string;
  value: number;
  labels: string;
  timestamp: number;
}

function MetricCard({
  title,
  value,
  unit = '',
  icon,
  color = colors.primary[500],
  isPercentage = false,
}: {
  title: string;
  value: number;
  unit?: string;
  icon?: React.ReactNode;
  color?: string;
  isPercentage?: boolean;
}) {
  const displayValue = isPercentage
    ? `${(value * 100).toFixed(1)}%`
    : value >= 1_000_000
    ? `${(value / 1_000_000).toFixed(2)}M`
    : value >= 1_000
    ? `${(value / 1_000).toFixed(1)}K`
    : value.toFixed(value % 1 === 0 ? 0 : 2);

  return (
    <Card
      sx={{
        height: '100%',
        background: `linear-gradient(135deg, ${alpha(color, 0.08)} 0%, ${alpha(color, 0.03)} 100%)`,
        borderColor: alpha(color, 0.2),
        transition: 'all 0.3s ease',
        '&:hover': {
          transform: 'translateY(-2px)',
          borderColor: alpha(color, 0.4),
        },
      }}
    >
      <CardContent sx={{ p: 2.5 }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 1.5 }}>
          <Typography
            variant="caption"
            sx={{
              color: 'text.secondary',
              fontWeight: 500,
              textTransform: 'capitalize',
              letterSpacing: '0.02em',
              lineHeight: 1.3,
              maxWidth: '80%',
            }}
          >
            {title.replace(/_/g, ' ')}
          </Typography>
          {icon && (
            <Box sx={{ color: color, opacity: 0.7 }}>{icon}</Box>
          )}
        </Box>
        <Typography
          variant="h4"
          sx={{
            fontWeight: 700,
            color: color,
            lineHeight: 1,
          }}
        >
          {displayValue}
          {unit && (
            <Typography component="span" variant="body2" sx={{ ml: 0.5, color: 'text.secondary' }}>
              {unit}
            </Typography>
          )}
        </Typography>
      </CardContent>
    </Card>
  );
}

function CategorySection({
  title,
  metrics,
  icon,
  color = colors.primary[500],
}: {
  title: string;
  metrics: MetricData[];
  icon: React.ReactNode;
  color?: string;
}) {
  if (!metrics.length) return null;

  return (
    <Box sx={{ mb: 4 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
        <Box
          sx={{
            width: 36,
            height: 36,
            borderRadius: 2,
            bgcolor: alpha(color, 0.1),
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: color,
          }}
        >
          {icon}
        </Box>
        <Typography variant="h6" sx={{ fontWeight: 600 }}>
          {title}
        </Typography>
        <Chip
          size="small"
          label={`${metrics.length} metrics`}
          sx={{ bgcolor: alpha(color, 0.1), color: color }}
        />
      </Box>
      <Grid container spacing={2}>
        {metrics.map((metric, idx) => (
          <Grid item xs={6} sm={4} md={3} lg={2} key={`${metric.name}-${idx}`}>
            <MetricCard
              title={metric.name.replace(/^bibbl_\w+_/, '')}
              value={metric.value}
              color={color}
            />
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}

export default function MetricsPage() {
  const [rawMetrics, setRawMetrics] = useState<string>('');
  const [parsedMetrics, setParsedMetrics] = useState<Record<string, MetricData[]>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [view, setView] = useState<'dashboard' | 'raw'>('dashboard');
  const [searchQuery, setSearchQuery] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const parseMetrics = useCallback((text: string) => {
    const metrics: Record<string, MetricData[]> = {
      http: [],
      buffer: [],
      pipeline: [],
      system: [],
      azure: [],
      other: [],
    };

    const lines = text.split(/\r?\n/);
    for (const line of lines) {
      if (line.startsWith('#') || !line.trim()) continue;

      const match = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*(?:\{[^}]*\})?) ([\d.+-e]+)(?:\s+(\d+))?$/);
      if (match) {
        const [, nameWithLabels, value, timestamp] = match;
        const metricMatch = nameWithLabels.match(/^([^{]+)(.*)$/);
        if (metricMatch) {
          const [, name, labelsStr] = metricMatch;
          
          let category = 'other';
          if (name.includes('http')) category = 'http';
          else if (name.includes('buffer')) category = 'buffer';
          else if (name.includes('pipeline')) category = 'pipeline';
          else if (name.includes('system') || name.includes('process') || name.includes('go_')) category = 'system';
          else if (name.includes('azure') || name.includes('sentinel')) category = 'azure';

          metrics[category].push({
            name,
            value: parseFloat(value),
            labels: labelsStr,
            timestamp: timestamp ? parseInt(timestamp) : Date.now(),
          });
        }
      }
    }
    return metrics;
  }, []);

  const loadMetrics = useCallback(async () => {
    setError('');
    try {
      const res = await fetch('/metrics', { headers: { Accept: 'text/plain' } });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const text = await res.text();
      setRawMetrics(text);
      setParsedMetrics(parseMetrics(text));
      setLastUpdated(new Date());
    } catch (e: any) {
      setError(e?.message || 'Failed to load metrics');
    } finally {
      setLoading(false);
    }
  }, [parseMetrics]);

  useEffect(() => {
    loadMetrics();
  }, [loadMetrics]);

  useEffect(() => {
    if (autoRefresh) {
      const interval = setInterval(loadMetrics, 10000);
      return () => clearInterval(interval);
    }
  }, [autoRefresh, loadMetrics]);

  const filteredRawLines = searchQuery
    ? rawMetrics.split(/\r?\n/).filter((line) => line.toLowerCase().includes(searchQuery.toLowerCase()))
    : rawMetrics.split(/\r?\n/);

  const categoryConfig = [
    { key: 'http', title: 'HTTP Performance', icon: <Speed />, color: colors.primary[500] },
    { key: 'pipeline', title: 'Pipeline Metrics', icon: <Timeline />, color: colors.success[500] },
    { key: 'buffer', title: 'Buffer Status', icon: <Memory />, color: colors.warning[500] },
    { key: 'azure', title: 'Azure Integration', icon: <CloudQueue />, color: '#0078d4' },
    { key: 'system', title: 'System Resources', icon: <Storage />, color: colors.slate[500] },
    { key: 'other', title: 'Other Metrics', icon: <TrendingUp />, color: colors.slate[400] },
  ];

  return (
    <Box>
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: 4,
          flexWrap: 'wrap',
          gap: 2,
        }}
      >
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, mb: 0.5 }}>
            System Metrics
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Real-time performance monitoring and system health
            {lastUpdated && (
              <Typography component="span" sx={{ ml: 1, color: colors.success[400] }}>
                • Updated {lastUpdated.toLocaleTimeString()}
              </Typography>
            )}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <FormControlLabel
            control={
              <Switch
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                size="small"
              />
            }
            label={
              <Typography variant="body2" color="text.secondary">
                Auto-refresh (10s)
              </Typography>
            }
          />
          <Tooltip title="Refresh now">
            <IconButton onClick={loadMetrics} disabled={loading}>
              <Refresh />
            </IconButton>
          </Tooltip>
          <Button
            variant="outlined"
            size="small"
            href="/metrics"
            target="_blank"
            endIcon={<OpenInNew fontSize="small" />}
          >
            Raw Metrics
          </Button>
        </Box>
      </Box>

      {/* View Toggle */}
      <Box sx={{ mb: 3 }}>
        <Tabs
          value={view}
          onChange={(_, v) => setView(v)}
          sx={{
            '& .MuiTabs-indicator': {
              height: 3,
              borderRadius: 1.5,
            },
          }}
        >
          <Tab value="dashboard" label="Dashboard View" />
          <Tab value="raw" label="Raw Metrics" />
        </Tabs>
      </Box>

      {error && (
        <Box
          sx={{
            mb: 3,
            p: 2,
            borderRadius: 2,
            bgcolor: alpha(colors.error[500], 0.1),
            border: `1px solid ${alpha(colors.error[500], 0.3)}`,
            display: 'flex',
            alignItems: 'center',
            gap: 1,
          }}
        >
          <Warning sx={{ color: colors.error[400] }} />
          <Typography color="error">{error}</Typography>
        </Box>
      )}

      {view === 'dashboard' ? (
        loading ? (
          <Grid container spacing={2}>
            {[...Array(12)].map((_, i) => (
              <Grid item xs={6} sm={4} md={3} lg={2} key={i}>
                <Skeleton variant="rounded" height={100} sx={{ borderRadius: 2 }} />
              </Grid>
            ))}
          </Grid>
        ) : (
          categoryConfig.map((cat) => (
            <CategorySection
              key={cat.key}
              title={cat.title}
              metrics={parsedMetrics[cat.key] || []}
              icon={cat.icon}
              color={cat.color}
            />
          ))
        )
      ) : (
        <Box>
          <TextField
            fullWidth
            size="small"
            placeholder="Filter metrics (e.g., http_requests_total)"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            sx={{ mb: 2 }}
          />
          <Card>
            <CardContent sx={{ p: 0 }}>
              <Box
                sx={{
                  maxHeight: 600,
                  overflow: 'auto',
                  p: 2,
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                  fontSize: '0.75rem',
                  lineHeight: 1.5,
                  bgcolor: alpha(colors.slate[900], 0.5),
                  color: colors.slate[300],
                }}
              >
                {filteredRawLines.map((line, i) => (
                  <Box
                    key={i}
                    sx={{
                      py: 0.25,
                      color: line.startsWith('#') ? colors.slate[500] : colors.slate[300],
                    }}
                  >
                    {line || ' '}
                  </Box>
                ))}
              </Box>
            </CardContent>
          </Card>
        </Box>
      )}
    </Box>
  );
}
