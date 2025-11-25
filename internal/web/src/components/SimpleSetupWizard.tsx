import React, { useState, useEffect, useCallback } from 'react';
import {
  Box, Button, Paper, Typography, Stepper, Step, StepLabel, StepContent,
  Card, CardContent, CardActionArea, TextField, Collapse,
  Alert, Chip, LinearProgress, Divider, IconButton, Tooltip, Dialog, DialogTitle, DialogContent, DialogActions,
  alpha
} from '@mui/material';
import {
  CheckCircle, RadioButtonUnchecked, ArrowForward, ArrowBack,
  Celebration, Refresh, PlayArrow, Add, Visibility, ExpandLess,
  AttachMoney
} from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';
import { colors } from '../theme/theme2026';

// Types
interface Source {
  id: string;
  name: string;
  type: string;
  status: string;
  enabled: boolean;
  flow?: boolean;
}

interface Destination {
  id: string;
  name: string;
  type: string;
  status: string;
  enabled: boolean;
}

// Preset routing scenarios - the most common patterns
const ROUTING_PRESETS = [
  {
    id: 'security-focus',
    name: '🛡️ Security Focus',
    description: 'Critical & High severity to Sentinel for alerting, everything to Data Lake for compliance',
    severities: ['critical', 'high'],
    priorityDestType: 'sentinel',
    archiveDestType: 'azure_datalake',
    icon: '🛡️',
    color: '#d32f2f'
  },
  {
    id: 'balanced',
    name: '⚖️ Balanced',
    description: 'Critical, High & Medium to Sentinel, full stream to archive',
    severities: ['critical', 'high', 'medium'],
    priorityDestType: 'sentinel',
    archiveDestType: 'azure_datalake',
    icon: '⚖️',
    color: '#1976d2'
  },
  {
    id: 'cost-optimized',
    name: '💰 Cost Optimized',
    description: 'Only Critical events to Sentinel, everything else to S3 for cheap storage',
    severities: ['critical'],
    priorityDestType: 'sentinel',
    archiveDestType: 's3',
    icon: '💰',
    color: '#388e3c'
  },
  {
    id: 'full-visibility',
    name: '👁️ Full Visibility',
    description: 'Send everything to Sentinel (higher cost, maximum visibility)',
    severities: ['critical', 'high', 'medium', 'low', 'debug'],
    priorityDestType: 'sentinel',
    archiveDestType: null,
    icon: '👁️',
    color: '#7b1fa2'
  }
];

// Severity levels for routing
const SEVERITY_LEVELS = [
  { id: 'critical', label: 'Critical', color: '#d32f2f', description: 'System failures, security breaches', costMultiplier: 1.0 },
  { id: 'high', label: 'High', color: '#f57c00', description: 'Major issues requiring attention', costMultiplier: 0.15 },
  { id: 'medium', label: 'Medium', color: '#fbc02d', description: 'Warnings and notable events', costMultiplier: 0.25 },
  { id: 'low', label: 'Low', color: '#388e3c', description: 'Informational messages', costMultiplier: 0.40 },
  { id: 'debug', label: 'Debug', color: '#1976d2', description: 'Debugging and trace logs', costMultiplier: 0.20 }
];

// Destination type info
const DEST_TYPES: Record<string, { icon: string; label: string; color: string; description: string; costPerGB: number }> = {
  sentinel: { 
    icon: '🛡️', 
    label: 'Microsoft Sentinel', 
    color: '#0078d4',
    description: 'Azure SIEM for security analytics',
    costPerGB: 2.76 // Approximate ingestion cost
  },
  azure_datalake: { 
    icon: '🌊', 
    label: 'Azure Data Lake', 
    color: '#00897b',
    description: 'Scalable storage for analytics',
    costPerGB: 0.023
  },
  azure_blob: { 
    icon: '📦', 
    label: 'Azure Blob Storage', 
    color: '#00897b',
    description: 'General purpose cloud storage',
    costPerGB: 0.018
  },
  s3: { 
    icon: '☁️', 
    label: 'AWS S3', 
    color: '#ff9900',
    description: 'Amazon cloud storage',
    costPerGB: 0.023
  },
  azure_loganalytics: {
    icon: '📊',
    label: 'Azure Log Analytics',
    color: '#0078d4',
    description: 'Azure monitoring workspace',
    costPerGB: 2.76
  },
  splunk: { 
    icon: '🔍', 
    label: 'Splunk', 
    color: '#000000',
    description: 'Enterprise data platform',
    costPerGB: 3.50
  }
};

export default function SimpleSetupWizard() {
  const [activeStep, setActiveStep] = useState(0);
  const [sources, setSources] = useState<Source[]>([]);
  const [destinations, setDestinations] = useState<Destination[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Wizard state
  const [selectedSource, setSelectedSource] = useState<string | null>(null);
  const [enableKVParsing, setEnableKVParsing] = useState(true);
  const [selectedSeverities, setSelectedSeverities] = useState<string[]>(['critical', 'high', 'medium']);
  const [priorityDest, setPriorityDest] = useState<string | null>(null);
  const [fullStreamDest, setFullStreamDest] = useState<string | null>(null);
  const [routingMode, setRoutingMode] = useState<'preset' | 'custom'>('preset');
  const [selectedPreset, setSelectedPreset] = useState<string>('balanced');

  // Quick create states
  const [showQuickSource, setShowQuickSource] = useState(false);
  const [showQuickDest, setShowQuickDest] = useState(false);
  const [quickSourceName, setQuickSourceName] = useState('');
  const [quickSourcePort, setQuickSourcePort] = useState('6514');
  const [quickDestName, setQuickDestName] = useState('');
  const [quickDestType, setQuickDestType] = useState<string>('sentinel');
  const [quickDestWorkspaceId, setQuickDestWorkspaceId] = useState('');
  const [quickDestSharedKey, setQuickDestSharedKey] = useState('');

  // Live log preview
  const [showLogPreview, setShowLogPreview] = useState(false);
  const [logPreviewLines, setLogPreviewLines] = useState<string[]>([]);

  // Cost estimation
  const [estimatedGBPerDay, setEstimatedGBPerDay] = useState(1);

  // Load data
  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [srcRes, destRes] = await Promise.all([
        apiClient.get('/api/v1/sources'),
        apiClient.get('/api/v1/destinations')
      ]);
      
      const srcData = srcRes.data;
      const destData = destRes.data;
      
      setSources(Array.isArray(srcData) ? srcData : (srcData?.items || []));
      setDestinations(Array.isArray(destData) ? destData : (destData?.items || []));
    } catch (e: any) {
      setError('Failed to load configuration');
    } finally {
      setLoading(false);
    }
  };

  // Fetch live logs for preview
  const fetchLogPreview = useCallback(async () => {
    if (!selectedSource) return;
    try {
      const res = await fetch(`/api/v1/sources/${selectedSource}/stream?limit=5`);
      if (res.ok) {
        const text = await res.text();
        const lines = text.split(/\r?\n/).filter(Boolean).slice(-5);
        setLogPreviewLines(lines);
      }
    } catch (e) {
      console.error('Failed to fetch log preview:', e);
    }
  }, [selectedSource]);

  useEffect(() => {
    if (showLogPreview && selectedSource) {
      fetchLogPreview();
      const interval = setInterval(fetchLogPreview, 3000);
      return () => clearInterval(interval);
    }
  }, [showLogPreview, selectedSource, fetchLogPreview]);

  // Quick create source
  const createQuickSource = async () => {
    if (!quickSourceName.trim()) return;
    setLoading(true);
    try {
      const res = await apiClient.post('/api/v1/sources', {
        name: quickSourceName,
        type: 'syslog',
        config: {
          port: parseInt(quickSourcePort) || 6514,
          protocol: 'tcp',
          tls: true
        }
      });
      await loadData();
      setSelectedSource(res.data?.id);
      setShowQuickSource(false);
      setQuickSourceName('');
    } catch (e: any) {
      setError(`Failed to create source: ${e.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Quick create destination
  const createQuickDest = async () => {
    if (!quickDestName.trim()) return;
    setLoading(true);
    try {
      const config: Record<string, any> = {};
      if (quickDestType === 'sentinel' || quickDestType === 'azure_loganalytics') {
        config.workspace_id = quickDestWorkspaceId;
        config.shared_key = quickDestSharedKey;
        config.log_type = 'BibblLogs';
      }
      
      const res = await apiClient.post('/api/v1/destinations', {
        name: quickDestName,
        type: quickDestType,
        config
      });
      await loadData();
      // Auto-select based on context
      const newDestId = res.data?.id;
      if (newDestId) {
        if (quickDestType === 'sentinel' || quickDestType === 'azure_loganalytics') {
          setPriorityDest(newDestId);
        } else {
          setFullStreamDest(newDestId);
        }
      }
      setShowQuickDest(false);
      setQuickDestName('');
      setQuickDestWorkspaceId('');
      setQuickDestSharedKey('');
    } catch (e: any) {
      setError(`Failed to create destination: ${e.message}`);
    } finally {
      setLoading(false);
    }
  };

  // Apply preset
  const applyPreset = (presetId: string) => {
    const preset = ROUTING_PRESETS.find(p => p.id === presetId);
    if (preset) {
      setSelectedPreset(presetId);
      setSelectedSeverities(preset.severities);
      
      // Auto-select destinations based on preset type preferences
      const sentinelDest = destinations.find(d => d.type === 'sentinel' || d.type === 'azure_loganalytics');
      const archiveDest = destinations.find(d => 
        d.type === preset.archiveDestType || 
        d.type === 'azure_datalake' || 
        d.type === 'azure_blob' || 
        d.type === 's3'
      );
      
      if (sentinelDest && preset.priorityDestType) {
        setPriorityDest(sentinelDest.id);
      }
      if (archiveDest && preset.archiveDestType) {
        setFullStreamDest(archiveDest.id);
      }
    }
  };

  const handleNext = () => {
    setActiveStep((prev) => prev + 1);
  };

  const handleBack = () => {
    setActiveStep((prev) => prev - 1);
  };

  const canProceed = () => {
    switch (activeStep) {
      case 0: return selectedSource !== null;
      case 1: return true; // KV parsing is optional
      case 2: return selectedSeverities.length > 0;
      case 3: return priorityDest !== null || fullStreamDest !== null;
      default: return true;
    }
  };

  // Calculate estimated monthly cost
  const calculateMonthlyCost = () => {
    let cost = 0;
    const gbPerMonth = estimatedGBPerDay * 30;
    
    if (priorityDest) {
      const dest = destinations.find(d => d.id === priorityDest);
      if (dest) {
        const destInfo = DEST_TYPES[dest.type];
        // Only selected severities go to priority dest
        const severityRatio = selectedSeverities.reduce((sum, sev) => {
          const level = SEVERITY_LEVELS.find(l => l.id === sev);
          return sum + (level?.costMultiplier || 0);
        }, 0);
        cost += gbPerMonth * severityRatio * (destInfo?.costPerGB || 0);
      }
    }
    
    if (fullStreamDest) {
      const dest = destinations.find(d => d.id === fullStreamDest);
      if (dest) {
        const destInfo = DEST_TYPES[dest.type];
        cost += gbPerMonth * (destInfo?.costPerGB || 0);
      }
    }
    
    return cost;
  };

  const applyConfiguration = async () => {
    setLoading(true);
    setError(null);
    
    try {
      // Step 1: Create/update pipeline with KV parsing if enabled
      const pipelineName = enableKVParsing ? 'auto-kv-parse' : 'passthrough';
      const pipelineResp = await apiClient.post('/api/v1/pipelines', {
        name: pipelineName,
        description: enableKVParsing 
          ? 'Auto-parse key:value pairs from syslog' 
          : 'Pass through without transformation',
        functions: enableKVParsing ? ['kv_parse'] : []
      });
      const pipelineId = pipelineResp.data?.id;

      // Step 2: Create severity routing rules
      if (priorityDest && selectedSeverities.length > 0) {
        // Create filter for selected severities
        const severityFilter = selectedSeverities
          .map(s => `severity == "${s}"`)
          .join(' || ');
        
        await apiClient.post('/api/v1/routes', {
          name: 'Priority Events → Sentinel',
          filter: severityFilter || 'true',
          pipelineID: pipelineId,
          destination: priorityDest,
          final: false
        });
      }

      // Step 3: Create full stream route
      if (fullStreamDest) {
        await apiClient.post('/api/v1/routes', {
          name: 'Full Stream → Archive',
          filter: 'true', // Match all
          pipelineID: pipelineId,
          destination: fullStreamDest,
          final: true
        });
      }

      setSuccess('🎉 Configuration applied successfully! Your log pipeline is now active.');
      setActiveStep(4); // Move to success step
    } catch (e: any) {
      setError(`Failed to apply configuration: ${e.message}`);
    } finally {
      setLoading(false);
    }
  };

  const toggleSeverity = (severity: string) => {
    setSelectedSeverities(prev => 
      prev.includes(severity) 
        ? prev.filter(s => s !== severity)
        : [...prev, severity]
    );
    setRoutingMode('custom');
  };

  const getDestInfo = (type: string) => {
    return DEST_TYPES[type] || { icon: '📤', label: type, color: '#666', costPerGB: 0 };
  };

  // Render step content
  const renderStepContent = () => {
    switch (activeStep) {
      case 0:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              📥 Select Your Log Source
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Choose which source is sending logs to Bibbl.
            </Typography>
            
            {sources.length === 0 ? (
              <Alert severity="info" sx={{ mb: 2 }}>
                No sources configured yet. Create one below to get started!
              </Alert>
            ) : (
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mb: 3 }}>
                {sources.map((src) => (
                  <Card 
                    key={src.id}
                    sx={{ 
                      border: selectedSource === src.id ? '2px solid #1976d2' : '1px solid #e0e0e0',
                      bgcolor: selectedSource === src.id ? 'rgba(25, 118, 210, 0.04)' : 'white',
                      transition: 'all 0.2s'
                    }}
                  >
                    <CardActionArea onClick={() => setSelectedSource(src.id)} sx={{ p: 2 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                        <Box sx={{ 
                          width: 48, height: 48, borderRadius: 2, 
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          fontSize: 24,
                          bgcolor: src.type === 'syslog' ? '#FF6B6B' : src.type === 'http' ? '#4ECDC4' : '#95E1D3',
                          color: 'white'
                        }}>
                          {src.type === 'syslog' ? '🔥' : src.type === 'http' ? '🌐' : '📨'}
                        </Box>
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="h6">{src.name}</Typography>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                            <Chip size="small" label={src.type.toUpperCase()} sx={{ bgcolor: '#f0f0f0' }} />
                            <Chip size="small" label={src.enabled ? 'ENABLED' : 'DISABLED'} color={src.enabled ? 'success' : 'default'} />
                            {src.flow && <Chip size="small" label="FLOWING" color="info" />}
                          </Box>
                        </Box>
                        {selectedSource === src.id ? (
                          <CheckCircle color="primary" sx={{ fontSize: 32 }} />
                        ) : (
                          <RadioButtonUnchecked sx={{ fontSize: 32, color: '#ccc' }} />
                        )}
                      </Box>
                    </CardActionArea>
                  </Card>
                ))}
              </Box>
            )}

            {/* Quick Create Source */}
            <Button 
              startIcon={<Add />} 
              onClick={() => setShowQuickSource(!showQuickSource)}
              variant="outlined"
              sx={{ mb: 2 }}
            >
              Quick Create Syslog Source
            </Button>
            
            <Collapse in={showQuickSource}>
              <Paper sx={{ p: 2, mb: 2, bgcolor: '#f9f9f9' }}>
                <Typography variant="subtitle2" gutterBottom>Create a new Syslog TLS source:</Typography>
                <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'flex-end' }}>
                  <TextField
                    size="small"
                    label="Source Name"
                    value={quickSourceName}
                    onChange={(e) => setQuickSourceName(e.target.value)}
                    placeholder="e.g., Firewall Logs"
                    sx={{ minWidth: 200 }}
                  />
                  <TextField
                    size="small"
                    label="Port"
                    value={quickSourcePort}
                    onChange={(e) => setQuickSourcePort(e.target.value)}
                    type="number"
                    sx={{ width: 100 }}
                  />
                  <Button variant="contained" size="small" onClick={createQuickSource} disabled={!quickSourceName.trim()}>
                    Create
                  </Button>
                </Box>
                <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                  Creates a TLS-enabled syslog listener. Your devices should send logs to this port.
                </Typography>
              </Paper>
            </Collapse>

            {/* Live Log Preview */}
            {selectedSource && (
              <Box sx={{ mt: 2 }}>
                <Button 
                  startIcon={showLogPreview ? <ExpandLess /> : <Visibility />}
                  onClick={() => setShowLogPreview(!showLogPreview)}
                  size="small"
                  color="info"
                >
                  {showLogPreview ? 'Hide Log Preview' : 'Show Live Logs'}
                </Button>
                <Collapse in={showLogPreview}>
                  <Paper sx={{ p: 2, mt: 1, bgcolor: '#1e1e1e', color: '#d4d4d4', fontFamily: 'monospace', fontSize: 12, maxHeight: 200, overflow: 'auto' }}>
                    {logPreviewLines.length > 0 ? (
                      logPreviewLines.map((line, i) => (
                        <Box key={i} sx={{ mb: 0.5, wordBreak: 'break-all' }}>{line}</Box>
                      ))
                    ) : (
                      <Box sx={{ color: '#888' }}>Waiting for logs...</Box>
                    )}
                  </Paper>
                </Collapse>
              </Box>
            )}
          </Box>
        );

      case 1:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              🔧 Log Parsing
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Universal KV parsing works great for most syslog formats (Palo Alto, FortiGate, Cisco, etc.)
            </Typography>
            
            <Card sx={{ 
              border: enableKVParsing ? '2px solid #4CAF50' : '1px solid #e0e0e0',
              bgcolor: enableKVParsing ? 'rgba(76, 175, 80, 0.04)' : 'white',
              mb: 2
            }}>
              <CardActionArea onClick={() => setEnableKVParsing(true)} sx={{ p: 3 }}>
                <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                  <Box sx={{ 
                    width: 56, height: 56, borderRadius: 2, 
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 28, bgcolor: '#4CAF50', color: 'white'
                  }}>
                    ⚡
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="h6">Universal Key:Value Parsing</Typography>
                      <Chip label="RECOMMENDED" size="small" color="success" />
                    </Box>
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                      Automatically extracts all key=value pairs and normalizes severity for routing.
                    </Typography>
                    <Paper sx={{ mt: 2, p: 2, bgcolor: '#f5f5f5', fontFamily: 'monospace', fontSize: 11 }}>
                      <Box sx={{ color: '#666' }}>Input:</Box>
                      <Box>src=10.0.0.1 dst=192.168.1.100 action=allow severity=high</Box>
                      <Box sx={{ color: '#666', mt: 1 }}>Output:</Box>
                      <Box sx={{ color: '#1976d2' }}>{'{ "src": "10.0.0.1", "severity": "high", ... }'}</Box>
                    </Paper>
                  </Box>
                  {enableKVParsing ? <CheckCircle color="success" sx={{ fontSize: 32 }} /> : <RadioButtonUnchecked sx={{ fontSize: 32, color: '#ccc' }} />}
                </Box>
              </CardActionArea>
            </Card>

            <Card sx={{ 
              border: !enableKVParsing ? '2px solid #9e9e9e' : '1px solid #e0e0e0',
              bgcolor: !enableKVParsing ? 'rgba(158, 158, 158, 0.04)' : 'white'
            }}>
              <CardActionArea onClick={() => setEnableKVParsing(false)} sx={{ p: 3 }}>
                <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                  <Box sx={{ 
                    width: 56, height: 56, borderRadius: 2, 
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 28, bgcolor: '#9e9e9e', color: 'white'
                  }}>
                    ➡️
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="h6">Pass Through (Raw)</Typography>
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                      Send logs as-is. Parse everything in Sentinel using KQL functions.
                    </Typography>
                  </Box>
                  {!enableKVParsing ? <CheckCircle sx={{ fontSize: 32, color: '#9e9e9e' }} /> : <RadioButtonUnchecked sx={{ fontSize: 32, color: '#ccc' }} />}
                </Box>
              </CardActionArea>
            </Card>
          </Box>
        );

      case 2:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              🎯 Choose a Routing Strategy
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Select a preset that matches your needs, or customize severity levels.
            </Typography>

            {/* Preset Cards */}
            <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2, mb: 3 }}>
              {ROUTING_PRESETS.map((preset) => (
                <Card 
                  key={preset.id}
                  sx={{ 
                    border: selectedPreset === preset.id && routingMode === 'preset' ? `2px solid ${preset.color}` : '1px solid #e0e0e0',
                    bgcolor: selectedPreset === preset.id && routingMode === 'preset' ? `${preset.color}08` : 'white',
                    cursor: 'pointer',
                    transition: 'all 0.2s',
                    '&:hover': { boxShadow: 2 }
                  }}
                  onClick={() => { setRoutingMode('preset'); applyPreset(preset.id); }}
                >
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                      <Typography variant="h5">{preset.icon}</Typography>
                      <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>{preset.name}</Typography>
                    </Box>
                    <Typography variant="body2" color="text.secondary" sx={{ fontSize: 12 }}>
                      {preset.description}
                    </Typography>
                    <Box sx={{ mt: 1.5, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                      {preset.severities.map(sev => {
                        const level = SEVERITY_LEVELS.find(l => l.id === sev);
                        return (
                          <Chip 
                            key={sev} 
                            label={level?.label} 
                            size="small" 
                            sx={{ 
                              bgcolor: level?.color, 
                              color: 'white',
                              fontSize: 10,
                              height: 20
                            }} 
                          />
                        );
                      })}
                    </Box>
                  </CardContent>
                </Card>
              ))}
            </Box>

            <Divider sx={{ my: 3 }}>
              <Chip label="OR CUSTOMIZE" size="small" />
            </Divider>

            {/* Custom Severity Selection */}
            <Typography variant="subtitle2" gutterBottom sx={{ fontWeight: 600 }}>
              Custom: Select which severities go to Sentinel
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
              {SEVERITY_LEVELS.map((sev) => (
                <Chip
                  key={sev.id}
                  label={`${sev.label} (~${Math.round(sev.costMultiplier * 100)}%)`}
                  onClick={() => toggleSeverity(sev.id)}
                  icon={selectedSeverities.includes(sev.id) ? <CheckCircle /> : undefined}
                  sx={{
                    fontSize: 14,
                    py: 2,
                    bgcolor: selectedSeverities.includes(sev.id) ? sev.color : '#f0f0f0',
                    color: selectedSeverities.includes(sev.id) ? 'white' : 'text.primary',
                    '&:hover': { opacity: 0.9 }
                  }}
                />
              ))}
            </Box>
            <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
              Percentages show typical log volume by severity level
            </Typography>
          </Box>
        );

      case 3:
        return (
          <Box sx={{ mt: 2 }}>
            <Typography variant="h6" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              📍 Choose Destinations
            </Typography>

            {destinations.length === 0 ? (
              <Alert severity="warning" sx={{ mb: 2 }}>
                No destinations configured yet. Create one below!
              </Alert>
            ) : (
              <>
                {/* Priority Destination */}
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  🎯 Priority Events ({selectedSeverities.map(s => SEVERITY_LEVELS.find(l => l.id === s)?.label).join(', ')}):
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mb: 3 }}>
                  {destinations.filter(d => ['sentinel', 'azure_loganalytics', 'splunk'].includes(d.type)).map((dest) => {
                    const info = getDestInfo(dest.type);
                    return (
                      <Card 
                        key={dest.id}
                        sx={{ 
                          border: priorityDest === dest.id ? `2px solid ${info.color}` : '1px solid #e0e0e0',
                          cursor: 'pointer'
                        }}
                        onClick={() => setPriorityDest(dest.id)}
                      >
                        <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 1.5 }}>
                          <Box sx={{ width: 40, height: 40, borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, bgcolor: info.color, color: 'white' }}>
                            {info.icon}
                          </Box>
                          <Box sx={{ flex: 1 }}>
                            <Typography variant="subtitle1">{dest.name}</Typography>
                            <Typography variant="caption" color="text.secondary">{info.label} • ~${info.costPerGB}/GB</Typography>
                          </Box>
                          {priorityDest === dest.id ? <CheckCircle sx={{ color: info.color }} /> : <RadioButtonUnchecked sx={{ color: '#ccc' }} />}
                        </CardContent>
                      </Card>
                    );
                  })}
                </Box>

                <Divider sx={{ my: 2 }} />

                {/* Archive Destination */}
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                  📦 Full Stream Archive (all events):
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mb: 3 }}>
                  {destinations.filter(d => ['azure_datalake', 'azure_blob', 's3'].includes(d.type)).map((dest) => {
                    const info = getDestInfo(dest.type);
                    return (
                      <Card 
                        key={dest.id}
                        sx={{ 
                          border: fullStreamDest === dest.id ? `2px solid ${info.color}` : '1px solid #e0e0e0',
                          cursor: 'pointer'
                        }}
                        onClick={() => setFullStreamDest(dest.id)}
                      >
                        <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 1.5 }}>
                          <Box sx={{ width: 40, height: 40, borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 20, bgcolor: info.color, color: 'white' }}>
                            {info.icon}
                          </Box>
                          <Box sx={{ flex: 1 }}>
                            <Typography variant="subtitle1">{dest.name}</Typography>
                            <Typography variant="caption" color="text.secondary">{info.label} • ~${info.costPerGB}/GB</Typography>
                          </Box>
                          {fullStreamDest === dest.id ? <CheckCircle sx={{ color: info.color }} /> : <RadioButtonUnchecked sx={{ color: '#ccc' }} />}
                        </CardContent>
                      </Card>
                    );
                  })}
                </Box>
              </>
            )}

            {/* Quick Create Destination */}
            <Button startIcon={<Add />} onClick={() => setShowQuickDest(true)} variant="outlined" size="small">
              Quick Add Destination
            </Button>

            {/* Cost Estimation */}
            {(priorityDest || fullStreamDest) && (
              <Paper sx={{ mt: 3, p: 2, bgcolor: '#fff8e1' }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                  <AttachMoney color="warning" />
                  <Typography variant="subtitle2">Estimated Monthly Cost</Typography>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
                  <Typography variant="body2">Daily data volume:</Typography>
                  <TextField
                    size="small"
                    type="number"
                    value={estimatedGBPerDay}
                    onChange={(e) => setEstimatedGBPerDay(Math.max(0.1, parseFloat(e.target.value) || 1))}
                    InputProps={{ endAdornment: <Typography variant="caption">GB/day</Typography> }}
                    sx={{ width: 120 }}
                  />
                </Box>
                <Typography variant="h5" sx={{ color: '#f57c00' }}>
                  ~${calculateMonthlyCost().toFixed(2)}/month
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Based on selected severities and destinations. Actual costs may vary.
                </Typography>
              </Paper>
            )}
          </Box>
        );

      case 4:
        return (
          <Box sx={{ mt: 2, textAlign: 'center', py: 4 }}>
            <Celebration sx={{ fontSize: 80, color: '#4CAF50', mb: 2 }} />
            <Typography variant="h4" gutterBottom sx={{ color: '#4CAF50' }}>
              All Done! 🎉
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ mb: 4, maxWidth: 500, mx: 'auto' }}>
              Your log pipeline is now configured and running.
            </Typography>
            
            <Paper sx={{ p: 3, maxWidth: 450, mx: 'auto', textAlign: 'left', bgcolor: '#f5f5f5' }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2 }}>Configuration Summary:</Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">Source:</Typography>
                  <Chip size="small" label={sources.find(s => s.id === selectedSource)?.name} />
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">Parsing:</Typography>
                  <Chip size="small" label={enableKVParsing ? 'Universal KV' : 'Pass Through'} color={enableKVParsing ? 'success' : 'default'} />
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Typography variant="body2" color="text.secondary">Severities to Sentinel:</Typography>
                  <Box sx={{ display: 'flex', gap: 0.5 }}>
                    {selectedSeverities.slice(0, 3).map(s => (
                      <Chip key={s} size="small" label={SEVERITY_LEVELS.find(l => l.id === s)?.label} sx={{ bgcolor: SEVERITY_LEVELS.find(l => l.id === s)?.color, color: 'white', height: 20, fontSize: 10 }} />
                    ))}
                    {selectedSeverities.length > 3 && <Chip size="small" label={`+${selectedSeverities.length - 3}`} />}
                  </Box>
                </Box>
                {priorityDest && (
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="body2" color="text.secondary">Priority →</Typography>
                    <Chip size="small" icon={<span>{getDestInfo(destinations.find(d => d.id === priorityDest)?.type || '')?.icon}</span>} label={destinations.find(d => d.id === priorityDest)?.name} />
                  </Box>
                )}
                {fullStreamDest && (
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Typography variant="body2" color="text.secondary">Archive →</Typography>
                    <Chip size="small" icon={<span>{getDestInfo(destinations.find(d => d.id === fullStreamDest)?.type || '')?.icon}</span>} label={destinations.find(d => d.id === fullStreamDest)?.name} />
                  </Box>
                )}
              </Box>
            </Paper>

            <Box sx={{ mt: 4, display: 'flex', gap: 2, justifyContent: 'center' }}>
              <Button variant="outlined" onClick={() => { setActiveStep(0); setSuccess(null); }}>
                Configure Another
              </Button>
              <Button variant="contained" color="primary" onClick={() => window.location.hash = '#home'}>
                Go to Dashboard
              </Button>
            </Box>
          </Box>
        );

      default:
        return null;
    }
  };

  const steps = [
    { label: 'Select Source', description: 'Choose your log input' },
    { label: 'Configure Parsing', description: 'Set up log parsing' },
    { label: 'Set Routing', description: 'Define routing rules' },
    { label: 'Choose Destinations', description: 'Select where logs go' }
  ];

  return (
    <Box sx={{ maxWidth: 900, mx: 'auto', p: 3 }}>
      {/* Header */}
      <Paper sx={{ 
        p: 4, 
        mb: 3, 
        background: `linear-gradient(135deg, ${colors.primary[600]} 0%, ${colors.primary[800]} 100%)`, 
        color: 'white',
        borderRadius: 4,
        border: `1px solid ${alpha(colors.primary[400], 0.3)}`,
      }}>
        <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>
          🚀 Quick Setup Wizard
        </Typography>
        <Typography variant="body1" sx={{ opacity: 0.9 }}>
          Get your log pipeline running in 4 simple steps. No complex configuration needed.
        </Typography>
      </Paper>

      {loading && <LinearProgress sx={{ mb: 2 }} />}
      {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>{error}</Alert>}
      {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

      {/* Stepper */}
      {activeStep < 4 && (
        <Stepper activeStep={activeStep} orientation="vertical" sx={{ mb: 3 }}>
          {steps.map((step, index) => (
            <Step key={step.label}>
              <StepLabel>
                <Typography variant="subtitle1" sx={{ fontWeight: activeStep === index ? 700 : 400 }}>
                  {step.label}
                </Typography>
                <Typography variant="caption" color="text.secondary">{step.description}</Typography>
              </StepLabel>
              <StepContent>
                <Box sx={{ mb: 2 }}>
                  {renderStepContent()}
                </Box>
                <Box sx={{ display: 'flex', gap: 2, mt: 3 }}>
                  <Button disabled={activeStep === 0} onClick={handleBack} startIcon={<ArrowBack />}>
                    Back
                  </Button>
                  {activeStep < 3 ? (
                    <Button variant="contained" onClick={handleNext} disabled={!canProceed()} endIcon={<ArrowForward />}>
                      Continue
                    </Button>
                  ) : (
                    <Button variant="contained" color="success" onClick={applyConfiguration} disabled={!canProceed() || loading} startIcon={<PlayArrow />}>
                      Apply Configuration
                    </Button>
                  )}
                </Box>
              </StepContent>
            </Step>
          ))}
        </Stepper>
      )}

      {/* Success state */}
      {activeStep === 4 && renderStepContent()}

      {/* Refresh button */}
      {activeStep < 4 && (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
          <Tooltip title="Refresh sources and destinations">
            <IconButton onClick={loadData} disabled={loading}>
              <Refresh />
            </IconButton>
          </Tooltip>
        </Box>
      )}

      {/* Quick Create Destination Dialog */}
      <Dialog open={showQuickDest} onClose={() => setShowQuickDest(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Quick Add Destination</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Destination Name"
              value={quickDestName}
              onChange={(e) => setQuickDestName(e.target.value)}
              placeholder="e.g., Production Sentinel"
              fullWidth
            />
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              {['sentinel', 'azure_datalake', 'azure_blob', 's3'].map(type => {
                const info = DEST_TYPES[type];
                return (
                  <Chip
                    key={type}
                    label={`${info.icon} ${info.label}`}
                    onClick={() => setQuickDestType(type)}
                    sx={{
                      bgcolor: quickDestType === type ? info.color : '#f0f0f0',
                      color: quickDestType === type ? 'white' : 'text.primary'
                    }}
                  />
                );
              })}
            </Box>
            {(quickDestType === 'sentinel' || quickDestType === 'azure_loganalytics') && (
              <>
                <TextField
                  label="Workspace ID"
                  value={quickDestWorkspaceId}
                  onChange={(e) => setQuickDestWorkspaceId(e.target.value)}
                  placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                  fullWidth
                />
                <TextField
                  label="Shared Key"
                  type="password"
                  value={quickDestSharedKey}
                  onChange={(e) => setQuickDestSharedKey(e.target.value)}
                  placeholder="Your workspace shared key"
                  fullWidth
                />
              </>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowQuickDest(false)}>Cancel</Button>
          <Button variant="contained" onClick={createQuickDest} disabled={!quickDestName.trim()}>
            Create Destination
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
