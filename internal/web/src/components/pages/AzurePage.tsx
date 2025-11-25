import React, { useState } from 'react';
import {
  Box,
  Grid,
  Typography,
  Card,
  CardContent,
  Button,
  TextField,
  Chip,
  alpha,
  Stepper,
  Step,
  StepLabel,
  StepContent,
  Alert,
  IconButton,
  Tooltip,
  Collapse,
} from '@mui/material';
import {
  Cloud,
  CloudDone,
  CloudOff,
  ContentCopy,
  Check,
  ExpandMore,
  ExpandLess,
  Info,
  PlayArrow,
  Link as LinkIcon,
} from '@mui/icons-material';
import { colors } from '../../theme/theme2026';

interface AzureResource {
  id: string;
  name: string;
  type: 'dce' | 'dcr' | 'workspace';
  status: 'active' | 'pending' | 'error';
  endpoint?: string;
}

function ResourceCard({
  title,
  description,
  icon,
  color,
  children,
  expanded,
  onToggle,
}: {
  title: string;
  description: string;
  icon: React.ReactNode;
  color: string;
  children: React.ReactNode;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <Card
      sx={{
        mb: 2,
        overflow: 'visible',
        borderColor: alpha(color, expanded ? 0.4 : 0.2),
        transition: 'all 0.3s ease',
      }}
    >
      <CardContent sx={{ p: 0 }}>
        <Box
          onClick={onToggle}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 2,
            p: 3,
            cursor: 'pointer',
            transition: 'background 0.2s ease',
            '&:hover': {
              bgcolor: alpha(color, 0.05),
            },
          }}
        >
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
            }}
          >
            {icon}
          </Box>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              {title}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {description}
            </Typography>
          </Box>
          <IconButton size="small">
            {expanded ? <ExpandLess /> : <ExpandMore />}
          </IconButton>
        </Box>
        <Collapse in={expanded}>
          <Box sx={{ p: 3, pt: 0, borderTop: `1px solid ${alpha(colors.slate[400], 0.1)}` }}>
            {children}
          </Box>
        </Collapse>
      </CardContent>
    </Card>
  );
}

function CopyableField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Box sx={{ mb: 2 }}>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
        {label}
      </Typography>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          p: 1.5,
          borderRadius: 1,
          bgcolor: alpha(colors.slate[700], 0.3),
          fontFamily: 'monospace',
          fontSize: '0.875rem',
        }}
      >
        <Typography
          variant="body2"
          sx={{
            flex: 1,
            fontFamily: 'inherit',
            wordBreak: 'break-all',
            color: colors.slate[300],
          }}
        >
          {value}
        </Typography>
        <Tooltip title={copied ? 'Copied!' : 'Copy to clipboard'}>
          <IconButton size="small" onClick={handleCopy}>
            {copied ? <Check fontSize="small" color="success" /> : <ContentCopy fontSize="small" />}
          </IconButton>
        </Tooltip>
      </Box>
    </Box>
  );
}

export default function AzurePage() {
  const [expandedCard, setExpandedCard] = useState<string | null>('dce');
  const [workspaceId, setWorkspaceId] = useState('');
  const [tableName, setTableName] = useState('Custom_BibblLogs_CL');
  const [dceName, setDceName] = useState('bibbl-dce');
  const [dcrName, setDcrName] = useState('bibbl-dcr');
  const [dceResponse, setDceResponse] = useState<any>(null);
  const [dcrResponse, setDcrResponse] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState<'dce' | 'dcr' | null>(null);

  const handleCreate = async (kind: 'dce' | 'dcr') => {
    setError(null);
    setLoading(kind);
    try {
      const body = kind === 'dcr' 
        ? { workspaceId, tableName, dcrName } 
        : { dceName };
      
      const res = await fetch(`/api/v1/azure/sentinel/${kind}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      
      const data = await res.json();
      
      if (!res.ok) {
        throw new Error(data.error || `Failed to create ${kind.toUpperCase()}`);
      }
      
      if (kind === 'dce') {
        setDceResponse(data);
      } else {
        setDcrResponse(data);
      }
    } catch (e: any) {
      setError(e?.message || 'An error occurred');
    } finally {
      setLoading(null);
    }
  };

  const toggleCard = (cardId: string) => {
    setExpandedCard((prev) => (prev === cardId ? null : cardId));
  };

  return (
    <Box>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Azure Integration
          </Typography>
          <Chip
            label="Microsoft Sentinel"
            size="small"
            sx={{
              bgcolor: alpha('#0078d4', 0.15),
              color: '#0078d4',
              fontWeight: 600,
            }}
          />
        </Box>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 700 }}>
          Create and manage Azure resources for log ingestion. Set up Data Collection Endpoints (DCE)
          and Data Collection Rules (DCR) to send logs to Microsoft Sentinel.
        </Typography>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Quick Start Guide */}
      <Card
        sx={{
          mb: 4,
          background: `linear-gradient(135deg, ${alpha('#0078d4', 0.1)} 0%, ${alpha('#0078d4', 0.05)} 100%)`,
          borderColor: alpha('#0078d4', 0.3),
        }}
      >
        <CardContent sx={{ p: 3 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
            <Info sx={{ color: '#0078d4' }} />
            <Typography variant="h6" sx={{ fontWeight: 600 }}>
              Quick Start Guide
            </Typography>
          </Box>
          <Stepper orientation="vertical" sx={{ mt: 2 }}>
            <Step active>
              <StepLabel>
                <Typography variant="subtitle2">Create a Data Collection Endpoint (DCE)</Typography>
              </StepLabel>
              <StepContent>
                <Typography variant="body2" color="text.secondary">
                  The DCE provides an endpoint URL where Bibbl will send log data.
                </Typography>
              </StepContent>
            </Step>
            <Step active>
              <StepLabel>
                <Typography variant="subtitle2">Create a Data Collection Rule (DCR)</Typography>
              </StepLabel>
              <StepContent>
                <Typography variant="body2" color="text.secondary">
                  The DCR defines how logs are transformed and which Log Analytics workspace receives them.
                </Typography>
              </StepContent>
            </Step>
            <Step active>
              <StepLabel>
                <Typography variant="subtitle2">Configure Destination in Bibbl</Typography>
              </StepLabel>
              <StepContent>
                <Typography variant="body2" color="text.secondary">
                  Use the DCE endpoint and DCR immutable ID to create an Azure destination.
                </Typography>
              </StepContent>
            </Step>
          </Stepper>
        </CardContent>
      </Card>

      {/* DCE Configuration */}
      <ResourceCard
        title="Data Collection Endpoint (DCE)"
        description="Ingestion endpoint for log data"
        icon={<LinkIcon />}
        color="#0078d4"
        expanded={expandedCard === 'dce'}
        onToggle={() => toggleCard('dce')}
      >
        <Box sx={{ pt: 2 }}>
          <TextField
            fullWidth
            label="DCE Name"
            value={dceName}
            onChange={(e) => setDceName(e.target.value)}
            placeholder="bibbl-dce"
            sx={{ mb: 3 }}
            helperText="A unique name for your Data Collection Endpoint"
          />
          <Button
            variant="contained"
            startIcon={<PlayArrow />}
            onClick={() => handleCreate('dce')}
            disabled={!dceName || loading === 'dce'}
            sx={{ mb: 3 }}
          >
            {loading === 'dce' ? 'Creating...' : 'Create DCE'}
          </Button>

          {dceResponse && (
            <Box
              sx={{
                mt: 2,
                p: 2,
                borderRadius: 2,
                bgcolor: alpha(colors.success[500], 0.1),
                border: `1px solid ${alpha(colors.success[500], 0.3)}`,
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <CloudDone sx={{ color: colors.success[400] }} />
                <Typography variant="subtitle2" sx={{ color: colors.success[400] }}>
                  DCE Created Successfully
                </Typography>
              </Box>
              <CopyableField label="DCE ID" value={dceResponse.id || 'N/A'} />
              <CopyableField label="Ingestion Endpoint" value={dceResponse.logsIngestionEndpoint || 'N/A'} />
            </Box>
          )}
        </Box>
      </ResourceCard>

      {/* DCR Configuration */}
      <ResourceCard
        title="Data Collection Rule (DCR)"
        description="Transformation and routing rules"
        icon={<Cloud />}
        color="#5c2d91"
        expanded={expandedCard === 'dcr'}
        onToggle={() => toggleCard('dcr')}
      >
        <Box sx={{ pt: 2 }}>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Log Analytics Workspace ID"
                value={workspaceId}
                onChange={(e) => setWorkspaceId(e.target.value)}
                placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                helperText="Your Azure Log Analytics workspace resource ID"
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="Table Name"
                value={tableName}
                onChange={(e) => setTableName(e.target.value)}
                placeholder="Custom_BibblLogs_CL"
                helperText="Custom log table name (must end with _CL)"
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                label="DCR Name"
                value={dcrName}
                onChange={(e) => setDcrName(e.target.value)}
                placeholder="bibbl-dcr"
                helperText="A unique name for your Data Collection Rule"
              />
            </Grid>
          </Grid>

          <Button
            variant="contained"
            startIcon={<PlayArrow />}
            onClick={() => handleCreate('dcr')}
            disabled={!workspaceId || !tableName || !dcrName || loading === 'dcr'}
            sx={{ mt: 3, mb: 3 }}
          >
            {loading === 'dcr' ? 'Creating...' : 'Create DCR'}
          </Button>

          {dcrResponse && (
            <Box
              sx={{
                mt: 2,
                p: 2,
                borderRadius: 2,
                bgcolor: alpha(colors.success[500], 0.1),
                border: `1px solid ${alpha(colors.success[500], 0.3)}`,
              }}
            >
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <CloudDone sx={{ color: colors.success[400] }} />
                <Typography variant="subtitle2" sx={{ color: colors.success[400] }}>
                  DCR Created Successfully
                </Typography>
              </Box>
              <CopyableField label="DCR ID" value={dcrResponse.id || 'N/A'} />
              <CopyableField label="Immutable ID" value={dcrResponse.immutableId || 'N/A'} />
              <CopyableField label="Stream Name" value={dcrResponse.streamName || 'N/A'} />
            </Box>
          )}
        </Box>
      </ResourceCard>

      {/* Resources Overview */}
      <Typography variant="h5" sx={{ fontWeight: 600, mb: 2, mt: 4 }}>
        Quick Links
      </Typography>
      <Grid container spacing={2}>
        {[
          {
            title: 'Azure Portal',
            description: 'Manage your Azure resources',
            url: 'https://portal.azure.com',
            color: '#0078d4',
          },
          {
            title: 'Microsoft Sentinel',
            description: 'View security analytics',
            url: 'https://portal.azure.com/#blade/Microsoft_Azure_Security_Insights/MainMenu',
            color: '#5c2d91',
          },
          {
            title: 'Log Analytics',
            description: 'Query your log data',
            url: 'https://portal.azure.com/#blade/Microsoft_Azure_Monitoring_Logs/LogsBlade',
            color: '#00a1f1',
          },
          {
            title: 'Documentation',
            description: 'Azure ingestion setup guide',
            url: 'https://learn.microsoft.com/azure/azure-monitor/logs/',
            color: colors.success[500],
          },
        ].map((link) => (
          <Grid item xs={12} sm={6} md={3} key={link.title}>
            <Card
              component="a"
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              sx={{
                textDecoration: 'none',
                display: 'block',
                transition: 'all 0.3s ease',
                '&:hover': {
                  transform: 'translateY(-2px)',
                  borderColor: alpha(link.color, 0.4),
                },
              }}
            >
              <CardContent sx={{ p: 2.5 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 0.5, color: link.color }}>
                  {link.title}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {link.description}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}
