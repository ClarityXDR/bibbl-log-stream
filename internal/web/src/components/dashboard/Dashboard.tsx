import React, { useEffect, useState, useMemo, useCallback, useRef } from 'react';
import {
  Box,
  Grid,
  Typography,
  Card,
  CardContent,
  IconButton,
  Chip,
  alpha,
  LinearProgress,
  Button,
  Tooltip,
  Skeleton,
} from '@mui/material';
import {
  TrendingUp,
  TrendingDown,
  Speed,
  Memory,
  Storage,
  CloudQueue,
  Refresh,
  ArrowForward,
  PlayArrow,
  Pause,
  Settings,
  CheckCircle,
  Warning,
  Error as ErrorIcon,
} from '@mui/icons-material';
import { colors } from '../../theme/theme2026';

interface DashboardProps {
  onNavigate: (tab: string) => void;
}

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: React.ReactNode;
  trend?: { value: number; isPositive: boolean };
  color?: string;
  loading?: boolean;
}

interface HealthStatus {
  status: 'ok' | 'degraded' | 'down';
  uptime?: number;
  lastCheck?: string;
}

interface PipelineStats {
  processed: number;
  filtered: number;
  errors: number;
  avgLatency: number;
}

// Stat Card Component
function StatCard({ title, value, subtitle, icon, trend, color = colors.primary[500], loading }: StatCardProps) {
  return (
    <Card
      sx={{
        height: '100%',
        position: 'relative',
        overflow: 'hidden',
        '&::before': {
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: 3,
          background: `linear-gradient(90deg, ${color} 0%, ${alpha(color, 0.5)} 100%)`,
        },
      }}
    >
      <CardContent sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
          <Typography
            variant="overline"
            sx={{ color: 'text.secondary', fontWeight: 600, letterSpacing: '0.1em' }}
          >
            {title}
          </Typography>
          {icon && (
            <Box
              sx={{
                width: 40,
                height: 40,
                borderRadius: 2,
                background: alpha(color, 0.1),
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: color,
              }}
            >
              {icon}
            </Box>
          )}
        </Box>

        {loading ? (
          <Skeleton variant="text" width="60%" height={48} />
        ) : (
          <Typography
            variant="h3"
            sx={{
              fontWeight: 700,
              background: `linear-gradient(135deg, ${color} 0%, ${alpha(color, 0.7)} 100%)`,
              backgroundClip: 'text',
              WebkitBackgroundClip: 'text',
              color: 'transparent',
              mb: 0.5,
            }}
          >
            {value}
          </Typography>
        )}

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {subtitle && (
            <Typography variant="body2" color="text.secondary">
              {subtitle}
            </Typography>
          )}
          {trend && (
            <Chip
              size="small"
              icon={trend.isPositive ? <TrendingUp fontSize="small" /> : <TrendingDown fontSize="small" />}
              label={`${trend.isPositive ? '+' : ''}${trend.value}%`}
              sx={{
                height: 22,
                bgcolor: trend.isPositive ? alpha(colors.success[500], 0.15) : alpha(colors.error[500], 0.15),
                color: trend.isPositive ? colors.success[400] : colors.error[400],
                '& .MuiChip-icon': {
                  color: 'inherit',
                },
              }}
            />
          )}
        </Box>
      </CardContent>
    </Card>
  );
}

// Live Activity Stream
function LiveActivityStream({ sourceId }: { sourceId?: string }) {
  const [lines, setLines] = useState<string[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [status, setStatus] = useState<'idle' | 'streaming' | 'error'>('idle');
  const logBoxRef = useRef<HTMLDivElement>(null);

  const fetchLines = useCallback(async () => {
    if (!sourceId || isPaused) return;
    try {
      const r = await fetch(`/api/v1/sources/${sourceId}/stream?limit=8`);
      if (!r.ok) throw new Error('Failed to fetch');
      const text = await r.text();
      const newLines = text.split(/\r?\n/).filter(Boolean).slice(-8);
      setLines(newLines);
      setStatus('streaming');
    } catch {
      setStatus('error');
    }
  }, [sourceId, isPaused]);

  useEffect(() => {
    if (!sourceId || isPaused) return;
    fetchLines();
    const id = setInterval(fetchLines, 3000);
    return () => clearInterval(id);
  }, [sourceId, isPaused, fetchLines]);

  useEffect(() => {
    if (logBoxRef.current) {
      logBoxRef.current.scrollTop = logBoxRef.current.scrollHeight;
    }
  }, [lines]);

  return (
    <Card sx={{ height: '100%' }}>
      <CardContent sx={{ p: 0 }}>
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            p: 2,
            borderBottom: `1px solid ${alpha(colors.slate[400], 0.1)}`,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Live Activity
            </Typography>
            <Chip
              size="small"
              label={status === 'streaming' ? 'LIVE' : status.toUpperCase()}
              sx={{
                height: 20,
                fontSize: '0.65rem',
                fontWeight: 700,
                bgcolor:
                  status === 'streaming'
                    ? alpha(colors.success[500], 0.15)
                    : status === 'error'
                    ? alpha(colors.error[500], 0.15)
                    : alpha(colors.slate[500], 0.15),
                color:
                  status === 'streaming'
                    ? colors.success[400]
                    : status === 'error'
                    ? colors.error[400]
                    : colors.slate[400],
              }}
            />
          </Box>
          <Box sx={{ display: 'flex', gap: 0.5 }}>
            <Tooltip title={isPaused ? 'Resume' : 'Pause'}>
              <IconButton size="small" onClick={() => setIsPaused(!isPaused)}>
                {isPaused ? <PlayArrow fontSize="small" /> : <Pause fontSize="small" />}
              </IconButton>
            </Tooltip>
            <Tooltip title="Refresh">
              <IconButton size="small" onClick={fetchLines}>
                <Refresh fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Box
          ref={logBoxRef}
          sx={{
            p: 2,
            height: 240,
            overflowY: 'auto',
            fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
            fontSize: '0.75rem',
            lineHeight: 1.6,
            bgcolor: alpha(colors.slate[900], 0.5),
          }}
        >
          {lines.length > 0 ? (
            lines.map((line, i) => (
              <Box
                key={i}
                sx={{
                  py: 0.5,
                  px: 1,
                  borderRadius: 1,
                  mb: 0.5,
                  bgcolor: alpha(colors.slate[700], 0.3),
                  color: colors.slate[300],
                  wordBreak: 'break-all',
                  transition: 'all 0.2s ease',
                  '&:hover': {
                    bgcolor: alpha(colors.primary[500], 0.1),
                  },
                }}
              >
                <Box
                  component="span"
                  sx={{ color: colors.slate[500], mr: 1, fontSize: '0.65rem' }}
                >
                  {String(i + 1).padStart(2, '0')}
                </Box>
                {line}
              </Box>
            ))
          ) : (
            <Box
              sx={{
                height: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: colors.slate[500],
              }}
            >
              {sourceId ? 'Waiting for logs...' : 'No source selected'}
            </Box>
          )}
        </Box>
      </CardContent>
    </Card>
  );
}

// Quick Action Card
function QuickActionCard({
  icon,
  title,
  description,
  onClick,
  color = colors.primary[500],
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  onClick: () => void;
  color?: string;
}) {
  return (
    <Card
      onClick={onClick}
      sx={{
        cursor: 'pointer',
        transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        '&:hover': {
          transform: 'translateY(-4px) scale(1.02)',
          borderColor: alpha(color, 0.4),
          boxShadow: `0 20px 40px -15px ${alpha(color, 0.25)}`,
        },
        '&:active': {
          transform: 'translateY(-2px) scale(1.01)',
        },
      }}
    >
      <CardContent sx={{ p: 3 }}>
        <Box
          sx={{
            width: 48,
            height: 48,
            borderRadius: 2,
            background: `linear-gradient(135deg, ${alpha(color, 0.2)} 0%, ${alpha(color, 0.1)} 100%)`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: color,
            mb: 2,
          }}
        >
          {icon}
        </Box>
        <Typography variant="h6" sx={{ fontWeight: 600, mb: 0.5 }}>
          {title}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {description}
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', color: color }}>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            Get started
          </Typography>
          <ArrowForward sx={{ ml: 0.5, fontSize: 16 }} />
        </Box>
      </CardContent>
    </Card>
  );
}

// Health Status Display
function HealthStatusCard({ status, uptime }: { status: HealthStatus; uptime?: number }) {
  const statusConfig = {
    ok: { color: colors.success[500], icon: <CheckCircle />, label: 'All Systems Operational' },
    degraded: { color: colors.warning[500], icon: <Warning />, label: 'Degraded Performance' },
    down: { color: colors.error[500], icon: <ErrorIcon />, label: 'System Down' },
  };

  const config = statusConfig[status.status] || statusConfig.ok;

  return (
    <Card
      sx={{
        background: `linear-gradient(135deg, ${alpha(config.color, 0.1)} 0%, ${alpha(config.color, 0.05)} 100%)`,
        borderColor: alpha(config.color, 0.3),
      }}
    >
      <CardContent sx={{ p: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Box
            sx={{
              width: 48,
              height: 48,
              borderRadius: 2,
              bgcolor: alpha(config.color, 0.15),
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: config.color,
            }}
          >
            {config.icon}
          </Box>
          <Box sx={{ flex: 1 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, color: config.color }}>
              {config.label}
            </Typography>
            {uptime !== undefined && (
              <Typography variant="body2" color="text.secondary">
                Uptime: {formatUptime(uptime)}
              </Typography>
            )}
          </Box>
          <Chip
            size="small"
            label={status.status.toUpperCase()}
            sx={{
              bgcolor: alpha(config.color, 0.15),
              color: config.color,
              fontWeight: 700,
              letterSpacing: '0.05em',
            }}
          />
        </Box>
      </CardContent>
    </Card>
  );
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}

function formatNumber(num: number): string {
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`;
  return String(num);
}

export default function Dashboard({ onNavigate }: DashboardProps) {
  const [healthData, setHealthData] = useState<HealthStatus>({ status: 'ok' });
  const [versionData, setVersionData] = useState<{ version: string } | null>(null);
  const [sourcesData, setSourcesData] = useState<any[]>([]);
  const [pipelineStats, setPipelineStats] = useState<PipelineStats>({
    processed: 0,
    filtered: 0,
    errors: 0,
    avgLatency: 0,
  });
  const [loading, setLoading] = useState(true);
  const [activeSourceId, setActiveSourceId] = useState<string | undefined>();

  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      try {
        const [healthRes, versionRes, sourcesRes, statsRes] = await Promise.all([
          fetch('/api/v1/health').then((r) => r.json()).catch(() => ({ status: 'down' })),
          fetch('/api/v1/version').then((r) => r.json()).catch(() => null),
          fetch('/api/v1/sources').then((r) => r.json()).catch(() => []),
          fetch('/api/v1/pipelines/stats').then((r) => r.json()).catch(() => []),
        ]);

        setHealthData(healthRes);
        setVersionData(versionRes);

        const sources = Array.isArray(sourcesRes)
          ? sourcesRes
          : Array.isArray(sourcesRes?.items)
          ? sourcesRes.items
          : [];
        setSourcesData(sources);

        // Find first flowing source for live preview
        const flowingSource = sources.find((s: any) => s.enabled && s.flow);
        const firstEnabled = sources.find((s: any) => s.enabled);
        setActiveSourceId(flowingSource?.id || firstEnabled?.id);

        // Aggregate pipeline stats
        const stats = Array.isArray(statsRes) ? statsRes : [];
        const aggregated = stats.reduce(
          (acc: PipelineStats, s: any) => ({
            processed: acc.processed + (s.processed || 0),
            filtered: acc.filtered + (s.filtered || 0),
            errors: acc.errors + (s.errors || 0),
            avgLatency: acc.avgLatency + (s.latency || 0),
          }),
          { processed: 0, filtered: 0, errors: 0, avgLatency: 0 }
        );
        if (stats.length > 0) {
          aggregated.avgLatency = aggregated.avgLatency / stats.length;
        }
        setPipelineStats(aggregated);
      } catch (error) {
        console.error('Failed to load dashboard data:', error);
      } finally {
        setLoading(false);
      }
    };

    loadData();
    const interval = setInterval(loadData, 15000);
    return () => clearInterval(interval);
  }, []);

  const activeSources = sourcesData.filter((s) => s.enabled).length;
  const flowingSources = sourcesData.filter((s) => s.flow).length;

  return (
    <Box>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Typography variant="h3" sx={{ fontWeight: 700, mb: 1 }}>
          Welcome to Bibbl
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 600 }}>
          Your single-binary log pipeline is ready. Monitor performance, configure sources, and route
          logs across multi-cloud destinations.
        </Typography>
      </Box>

      {/* Health Status */}
      <Box sx={{ mb: 3 }}>
        <HealthStatusCard status={healthData} uptime={healthData.uptime} />
      </Box>

      {/* Stats Grid */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Events Processed"
            value={formatNumber(pipelineStats.processed)}
            subtitle="Last 24 hours"
            icon={<Speed />}
            trend={{ value: 12, isPositive: true }}
            color={colors.primary[500]}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Active Sources"
            value={`${activeSources}/${sourcesData.length}`}
            subtitle={`${flowingSources} flowing`}
            icon={<Storage />}
            color={colors.success[500]}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Filtered Events"
            value={formatNumber(pipelineStats.filtered)}
            subtitle="Dropped by rules"
            icon={<Memory />}
            trend={{ value: 5, isPositive: false }}
            color={colors.warning[500]}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Avg Latency"
            value={`${pipelineStats.avgLatency.toFixed(1)}ms`}
            subtitle="End-to-end"
            icon={<CloudQueue />}
            color={colors.success[500]}
            loading={loading}
          />
        </Grid>
      </Grid>

      {/* Quick Actions */}
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 2 }}>
        Quick Actions
      </Typography>
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={4}>
          <QuickActionCard
            icon={<Speed fontSize="large" />}
            title="Quick Setup Wizard"
            description="Configure your log pipeline in 4 simple steps"
            onClick={() => onNavigate('setup')}
            color={colors.primary[500]}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <QuickActionCard
            icon={<Storage fontSize="large" />}
            title="Add Data Source"
            description="Connect syslog, HTTP, Kafka, or other inputs"
            onClick={() => onNavigate('sources')}
            color={colors.success[500]}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <QuickActionCard
            icon={<CloudQueue fontSize="large" />}
            title="Configure Destinations"
            description="Route logs to Sentinel, S3, Splunk & more"
            onClick={() => onNavigate('destinations')}
            color={colors.warning[500]}
          />
        </Grid>
      </Grid>

      {/* Live Activity */}
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 2 }}>
        Live Activity
      </Typography>
      <LiveActivityStream sourceId={activeSourceId} />

      {/* Footer Info */}
      <Box
        sx={{
          mt: 4,
          pt: 3,
          borderTop: `1px solid ${alpha(colors.slate[400], 0.1)}`,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Typography variant="body2" color="text.secondary">
          Bibbl Log Stream {versionData?.version || '...'} • © {new Date().getFullYear()} Bibbl
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            size="small"
            href="/metrics"
            target="_blank"
            sx={{ color: colors.slate[400] }}
          >
            Prometheus Metrics
          </Button>
          <Button
            size="small"
            href="https://github.com/ClarityXDR/bibbl-log-stream"
            target="_blank"
            sx={{ color: colors.slate[400] }}
          >
            Documentation
          </Button>
        </Box>
      </Box>
    </Box>
  );
}
