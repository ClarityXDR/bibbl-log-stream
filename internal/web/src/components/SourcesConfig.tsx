import React, { useState, useEffect, useRef } from 'react'
import './styles.css'
import {
  Box, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Select, MenuItem, FormControl, InputLabel, Grid,
  Table, TableBody, TableCell, TableHead, TableRow, IconButton,
  Paper, Typography, Chip, LinearProgress, Switch, FormControlLabel, Snackbar, Alert, Tooltip,
  Card, CardContent
} from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import { Add, Edit, Delete, PlayArrow, Stop, Visibility, Download, Cloud, Speed, Queue, Security, ArrowBack as BackIcon, Save as SaveIcon } from '@mui/icons-material';
import AkamaiWorkbench from './AkamaiWorkbench';
import LoadTestWorkbench from './LoadTestWorkbench';
import { apiClient } from '../utils/apiClient';
import SourceTemplateGallery from './sources/SourceTemplateGallery';
import { SourceTemplate } from './sources/sourceTemplates';

interface Source {
  id: string;
  name: string;
  type: 'syslog' | 'http' | 'kafka' | 'file' | 'windows_event' | 'akamai_ds2';
  config: Record<string, any>;
  status: 'running' | 'stopped' | 'error';
  enabled: boolean;
  lastUnix?: number;
  flow?: boolean;
  // Optional queue depth metrics (future backend addition)
  queueDepth?: number;
}

export default function SourcesConfig() {
  const [currentView, setCurrentView] = useState<'gallery' | 'list' | 'builder'>('list');
  const [selectedTemplate, setSelectedTemplate] = useState<SourceTemplate | null>(null);
  const [testMode, setTestMode] = useState<boolean>(true);
  const [sources, setSources] = useState<Source[]>([]);
  const [open, setOpen] = useState(false);
  const [editingSource, setEditingSource] = useState<Partial<Source>>({
    type: 'syslog',
    config: {}
  });
  const [detailFor, setDetailFor] = useState<Source|null>(null);
  const [akamaiFor, setAkamaiFor] = useState<Source|null>(null);
  const [loadTestFor, setLoadTestFor] = useState<Source|null>(null);
  const [actionError, setActionError] = useState<string>('');
  const [saveOk, setSaveOk] = useState(false);

  useEffect(() => {
    loadSources();
  }, []);

  const handleSelectTemplate = (template: SourceTemplate) => {
    setSelectedTemplate(template);
    setCurrentView('builder');
    setEditingSource({
      name: template.config.name,
      type: template.config.type,
      config: template.config.config
    });
  };

  const handleBack = () => {
    if (confirm('Go back? Your current work will be lost.')) {
      setCurrentView('list');
      setEditingSource({ type: 'syslog', config: {} });
      setSelectedTemplate(null);
    }
  };

  const getProgress = () => {
    let completed = 0;
    const total = 3;
    if (editingSource.name && editingSource.name.trim().length > 0) completed++;
    if (editingSource.type) completed++;
    if (editingSource.config && Object.keys(editingSource.config).length > 0) completed++;
    return (completed / total) * 100;
  };

  const loadSources = async () => {
    try {
  const response = await apiClient.get('/api/v1/sources');
  let items: any[] = [];
  const data = response.data;
  if (Array.isArray(data)) items = data; else if (Array.isArray(data?.items)) items = data.items; else if (Array.isArray(data?.sources)) items = data.sources; // legacy
  setSources(items as Source[]);
    } catch (error) {
      console.error('Failed to load sources:', error);
    }
  };

  const handleSave = async () => {
    if (testMode && currentView === 'builder') {
      if (confirm('🧪 You are in Test Mode. Switch to Live Mode to save this source for real?')) {
        setTestMode(false);
      }
      return;
    }
    try {
      if (editingSource.id) {
        await apiClient.put(`/api/v1/sources/${editingSource.id}`, editingSource);
      } else {
        await apiClient.post('/api/v1/sources', editingSource);
      }
      setSaveOk(true);
      setTimeout(() => {
        setOpen(false);
        setCurrentView('list');
        setEditingSource({ type: 'syslog', config: {} });
        setSelectedTemplate(null);
        loadSources();
      }, 2000);
    } catch (error) {
      console.error('Failed to save source:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (window.confirm('Delete this source?')) {
      try {
        await apiClient.delete(`/api/v1/sources/${id}`);
        loadSources();
      } catch (error) {
        console.error('Failed to delete source:', error);
      }
    }
  };

  const handleDownloadVersaCerts = async () => {
    try {
      const response = await apiClient.get('/api/v1/syslog/certs/bundle', { responseType: 'blob' });
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', 'bibbl-versa-certs.zip');
      document.body.appendChild(link);
      link.click();
      link.parentNode?.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to download Versa certificates:', error);
      alert('Failed to download certificates. Make sure the syslog source is configured with TLS.');
    }
  };

  const handleToggle = async (source: Source) => {
    try {
      setActionError('');
      await apiClient.post(`/api/v1/sources/${source.id}/${source.enabled ? 'stop' : 'start'}`);
      loadSources();
    } catch (error:any) {
      console.error('Failed to toggle source:', error);
      const msg = error?.response?.data || error?.message || 'toggle failed';
      setActionError(`Toggle failed for ${source.name}: ${msg}`);
    }
  };

  const renderConfigFields = () => {
  switch (editingSource.type) {
      case 'syslog':
        return (
          <>
            <TextField
              fullWidth
              label="Host"
              value={editingSource.config?.host || '0.0.0.0'}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, host: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Port"
              type="number"
              value={editingSource.config?.port || 514}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, port: parseInt(e.target.value) }
              })}
              margin="normal"
            />
            <FormControl fullWidth margin="normal">
              <InputLabel id="protocol-label">Protocol</InputLabel>
              <Select
                labelId="protocol-label"
                label="Protocol"
                value={editingSource.config?.protocol || 'udp'}
                onChange={(e: SelectChangeEvent<string>) => setEditingSource({
                  ...editingSource,
                  config: { ...editingSource.config, protocol: e.target.value }
                })}
              >
                <MenuItem value="udp">UDP</MenuItem>
                <MenuItem value="tcp">TCP</MenuItem>
                <MenuItem value="tls">TLS</MenuItem>
              </Select>
            </FormControl>
            {editingSource.config?.protocol === 'tls' && (
              <>
                <TextField
                  fullWidth
                  label="TLS Cert File"
                  value={editingSource.config?.certFile || ''}
                  onChange={(e) => setEditingSource({
                    ...editingSource,
                    config: { ...editingSource.config, certFile: e.target.value }
                  })}
                  margin="normal"
                />
                <TextField
                  fullWidth
                  label="TLS Key File"
                  value={editingSource.config?.keyFile || ''}
                  onChange={(e) => setEditingSource({
                    ...editingSource,
                    config: { ...editingSource.config, keyFile: e.target.value }
                  })}
                  margin="normal"
                />
              </>
            )}
            <TextField
              fullWidth
              label="Allowlist (IPs or CIDRs, comma-separated)"
              helperText="Leave empty to allow all"
              value={(editingSource.config?.allow && Array.isArray(editingSource.config.allow) ? editingSource.config.allow.join(', ') : (editingSource.config?.allow || ''))}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, allow: e.target.value.split(',').map((s:string)=>s.trim()).filter(Boolean) }
              })}
              margin="normal"
            />
          </>
        );
      case 'http':
        return (
          <>
            <TextField
              fullWidth
              label="Port"
              type="number"
              value={editingSource.config?.port || 8080}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, port: parseInt(e.target.value) }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Path"
              value={editingSource.config?.path || '/logs'}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, path: e.target.value }
              })}
              margin="normal"
            />
          </>
        );
      case 'kafka':
        return (
          <>
            <TextField
              fullWidth
              label="Brokers"
              value={editingSource.config?.brokers || ''}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, brokers: e.target.value }
              })}
              margin="normal"
              helperText="Comma-separated list"
            />
            <TextField
              fullWidth
              label="Topics"
              value={editingSource.config?.topics || ''}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, topics: e.target.value }
              })}
              margin="normal"
              helperText="Comma-separated list"
            />
            <TextField
              fullWidth
              label="Consumer Group"
              value={editingSource.config?.consumerGroup || 'bibbl-stream'}
              onChange={(e) => setEditingSource({
                ...editingSource,
                config: { ...editingSource.config, consumerGroup: e.target.value }
              })}
              margin="normal"
            />
          </>
        );
      case 'akamai_ds2':
        return (
          <>
            <TextField fullWidth label="Host" helperText="Akamai API host (akab-xxxx.luna.akamaiapis.net)"
              value={editingSource.config?.host || ''}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, host: e.target.value}})}
              margin="normal" />
            <TextField fullWidth label="Client Token"
              value={editingSource.config?.clientToken || ''}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, clientToken: e.target.value}})}
              margin="normal" />
            <TextField fullWidth type="password" label="Client Secret"
              value={editingSource.config?.clientSecret || ''}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, clientSecret: e.target.value}})}
              margin="normal" />
            <TextField fullWidth label="Access Token"
              value={editingSource.config?.accessToken || ''}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, accessToken: e.target.value}})}
              margin="normal" />
            <TextField fullWidth label="Stream IDs (comma separated, blank=all)"
              value={editingSource.config?.streams || ''}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, streams: e.target.value}})}
              margin="normal" />
            <TextField fullWidth type="number" label="Poll Interval Seconds"
              value={editingSource.config?.intervalSeconds || 60}
              onChange={(e)=> setEditingSource({...editingSource, config:{...editingSource.config, intervalSeconds: parseInt(e.target.value)}})}
              margin="normal" />
          </>
        );
      default:
        return null;
    }
  };

  // Render gallery view
  if (currentView === 'gallery') {
    return (
      <Paper elevation={6} sx={{ borderRadius: 3, overflow: 'hidden' }}>
        <SourceTemplateGallery onSelectTemplate={handleSelectTemplate} />
      </Paper>
    );
  }

  // Render builder view
  if (currentView === 'builder') {
    return (
      <Paper elevation={6} sx={{ borderRadius: 3, overflow: 'hidden' }}>
        {/* Header */}
        <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider', bgcolor: 'primary.50' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Box sx={{ flex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 0.5 }}>
                <Typography variant="h5" sx={{ fontWeight: 700 }}>
                  {selectedTemplate?.icon} {selectedTemplate?.title || 'Custom Source'}
                </Typography>
                {testMode && (
                  <Chip 
                    label="\ud83e\uddea Test Mode" 
                    size="small" 
                    color="warning"
                    sx={{ fontWeight: 600 }}
                  />
                )}
              </Box>
              <Typography variant="body2" color="text.secondary">
                {selectedTemplate?.description || 'Configure your custom data source'}
              </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <FormControlLabel
                control={
                  <Switch 
                    checked={testMode} 
                    onChange={(e) => setTestMode(e.target.checked)}
                    color="warning"
                  />
                }
                label={
                  <Typography variant="caption" sx={{ fontWeight: 600 }}>
                    {testMode ? 'Testing' : 'Live'}
                  </Typography>
                }
              />
              <Tooltip title="Back to sources list">
                <IconButton onClick={handleBack} size="large">
                  <BackIcon />
                </IconButton>
              </Tooltip>
            </Box>
          </Box>

          {/* Progress Bar */}
          <Box sx={{ mt: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 600 }}>
                Progress: {Math.round(getProgress())}% Complete
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {Math.round(getProgress()) === 100 ? '\ud83c\udf89 Ready to save!' : 'Keep going...'}
              </Typography>
            </Box>
            <LinearProgress 
              variant="determinate" 
              value={getProgress()} 
              sx={{ 
                height: 8, 
                borderRadius: 1,
                bgcolor: 'grey.200',
                '& .MuiLinearProgress-bar': {
                  borderRadius: 1,
                  background: getProgress() === 100 
                    ? 'linear-gradient(90deg, #4caf50 0%, #8bc34a 100%)'
                    : 'linear-gradient(90deg, #2196f3 0%, #21cbf3 100%)'
                }
              }}
            />
          </Box>
        </Box>

        {/* Builder Content */}
        <Box sx={{ p: 3 }}>
          <TextField
            fullWidth
            label="Source Name"
            value={editingSource.name || ''}
            onChange={(e) => setEditingSource({ ...editingSource, name: e.target.value })}
            margin="normal"
            required
            helperText="Give your source a descriptive name"
          />
          <FormControl fullWidth margin="normal">
            <InputLabel>Type</InputLabel>
            <Select
              value={editingSource.type}
              label="Type"
              onChange={(e: SelectChangeEvent<Source['type']>) => setEditingSource({
                ...editingSource,
                type: e.target.value as Source['type'],
                config: {}
              })}
            >
              <MenuItem value="syslog">Syslog</MenuItem>
              <MenuItem value="http">HTTP</MenuItem>
              <MenuItem value="kafka">Kafka</MenuItem>
              <MenuItem value="file">File</MenuItem>
              <MenuItem value="windows_event">Windows Event Log</MenuItem>
              <MenuItem value="akamai_ds2">Akamai DataStream 2</MenuItem>
            </Select>
          </FormControl>
          {renderConfigFields()}

          <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end', mt: 3 }}>
            <Button variant="outlined" onClick={handleBack}>
              Cancel
            </Button>
            <Button
              variant="contained"
              size="large"
              startIcon={<SaveIcon />}
              onClick={handleSave}
              disabled={!editingSource.name || !editingSource.type}
              sx={{
                py: 1.5,
                px: 4,
                fontSize: '1.1rem',
                fontWeight: 700,
                background: (editingSource.name && editingSource.type)
                  ? 'linear-gradient(45deg, #4CAF50 30%, #8BC34A 90%)'
                  : undefined,
                boxShadow: (editingSource.name && editingSource.type) ? '0 3px 5px 2px rgba(76, 175, 80, .3)' : undefined,
                '&:hover': (editingSource.name && editingSource.type) ? {
                  background: 'linear-gradient(45deg, #45A049 30%, #7CB342 90%)',
                  transform: 'scale(1.05)',
                  boxShadow: '0 6px 10px 4px rgba(76, 175, 80, .4)'
                } : undefined
              }}
            >
              {testMode ? '\ud83e\uddea Test Save' : '\ud83c\udf89 Save Source!'}
            </Button>
          </Box>
        </Box>

        {/* Success Snackbar */}
        <Snackbar 
          open={saveOk} 
          autoHideDuration={4000} 
          onClose={() => setSaveOk(false)}
          anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        >
          <Alert 
            onClose={() => setSaveOk(false)} 
            severity="success" 
            sx={{ 
              width: '100%',
              fontSize: '1.1rem',
              fontWeight: 600
            }}
            variant="filled"
          >
            \ud83c\udf89 {testMode ? 'Test completed! Changes not saved.' : 'Success! Source saved and ready to use!'}
          </Alert>
        </Snackbar>
      </Paper>
    );
  }

  // Helper function to get source icon and color
  const getSourceVisuals = (type: string) => {
    switch (type) {
      case 'syslog': return { icon: '🔥', color: '#FF6B6B', bg: '#FFE8E8' };
      case 'http': return { icon: '🌐', color: '#4ECDC4', bg: '#E0F7F6' };
      case 'kafka': return { icon: '📨', color: '#95E1D3', bg: '#E8F8F5' };
      case 'file': return { icon: '📁', color: '#FFA07A', bg: '#FFF0E6' };
      case 'windows_event': return { icon: '🪟', color: '#00A8E8', bg: '#E0F4FF' };
      case 'akamai_ds2': return { icon: '☁️', color: '#9B59B6', bg: '#F4ECF7' };
      default: return { icon: '⚡', color: '#95A5A6', bg: '#ECF0F1' };
    }
  };

  // Render list view (default) - Visual card layout
  return (
    <Box>
      <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h5" sx={{ fontWeight: 700 }}>Data Sources</Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<Add />}
            onClick={() => setCurrentView('gallery')}
            sx={{ borderRadius: 2 }}
          >
            Use Template
          </Button>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={() => {
              setEditingSource({ type: 'syslog', config: {} });
              setOpen(true);
            }}
            sx={{ borderRadius: 2 }}
          >
            Add Source
          </Button>
        </Box>
      </Box>

      {actionError && (
        <Box sx={{mb:2}}>
          <Paper elevation={0} sx={{p:2, bgcolor:'error.light', color:'error.contrastText', borderRadius: 2, display:'flex', justifyContent:'space-between', alignItems:'center'}}>
            <Typography>{actionError}</Typography>
            <Button size="small" variant="text" sx={{color:'inherit'}} onClick={()=>setActionError('')}>Dismiss</Button>
          </Paper>
        </Box>
      )}

      {sources.length === 0 ? (
        <Paper elevation={3} sx={{ p: 6, textAlign: 'center', borderRadius: 3, bgcolor: 'background.default' }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No sources configured yet
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Get started by choosing a template or creating a custom source
          </Typography>
          <Button
            variant="contained"
            size="large"
            startIcon={<Add />}
            onClick={() => setCurrentView('gallery')}
            sx={{ borderRadius: 2, px: 4 }}
          >
            Browse Templates
          </Button>
        </Paper>
      ) : (
        <Grid container spacing={3}>
          {sources.map((source) => {
            const visuals = getSourceVisuals(source.type);
            const secs = source.lastUnix ? Math.max(0, Math.floor(Date.now()/1000 - source.lastUnix)) : -1;
            const lastActivity = secs < 0 ? 'Never' : (secs === 0 ? 'Just now' : `${secs}s ago`);
            const queueDepth = source.queueDepth ?? 0;

            return (
              <Grid item xs={12} sm={6} md={4} key={source.id}>
                <Card 
                  elevation={3}
                  sx={{
                    height: '100%',
                    borderRadius: 3,
                    transition: 'all 0.3s ease',
                    cursor: 'pointer',
                    '&:hover': {
                      transform: 'translateY(-4px)',
                      boxShadow: 6,
                    },
                  }}
                  onClick={() => setDetailFor(source)}
                >
                  <CardContent sx={{ p: 3 }}>
                    {/* Header with icon and status */}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                      <Box 
                        sx={{ 
                          width: 56, 
                          height: 56, 
                          borderRadius: 2, 
                          bgcolor: visuals.bg,
                          display: 'flex', 
                          alignItems: 'center', 
                          justifyContent: 'center',
                          fontSize: 28,
                        }}
                      >
                        {visuals.icon}
                      </Box>
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        <Chip
                          label={source.enabled ? 'ON' : 'OFF'}
                          size="small"
                          color={source.enabled ? 'success' : 'default'}
                          sx={{ fontWeight: 600, fontSize: 10 }}
                        />
                        {source.flow && (
                          <Chip
                            label="FLOWING"
                            size="small"
                            color="info"
                            sx={{ fontWeight: 600, fontSize: 10 }}
                          />
                        )}
                      </Box>
                    </Box>

                    {/* Name and Type */}
                    <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5, color: visuals.color }}>
                      {source.name}
                    </Typography>
                    <Typography variant="caption" sx={{ color: 'text.secondary', textTransform: 'uppercase', letterSpacing: 1, fontWeight: 600 }}>
                      {source.type}
                    </Typography>

                    {/* Configuration */}
                    <Box sx={{ mt: 2, mb: 2, p: 1.5, bgcolor: 'background.default', borderRadius: 1.5 }}>
                      <Typography variant="body2" sx={{ fontSize: 12, color: 'text.secondary' }}>
                        {source.type === 'syslog' && `📍 Port ${source.config.port}`}
                        {source.type === 'http' && `🔗 ${source.config.path}:${source.config.port}`}
                        {source.type === 'kafka' && `📋 ${source.config.topics || 'No topics'}`}
                        {source.type === 'akamai_ds2' && `🌍 ${source.config.clientId || 'Akamai Stream'}`}
                        {!['syslog', 'http', 'kafka', 'akamai_ds2'].includes(source.type) && 'Configuration available'}
                      </Typography>
                    </Box>

                    {/* Stats */}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
                      <Box>
                        <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                          Last Activity
                        </Typography>
                        <Typography variant="body2" sx={{ fontWeight: 600, color: source.flow ? 'success.main' : 'text.secondary' }}>
                          {lastActivity}
                        </Typography>
                      </Box>
                      {queueDepth > 0 && (
                        <Box>
                          <Typography variant="caption" sx={{ color: 'text.secondary', display: 'block' }}>
                            Queue Depth
                          </Typography>
                          <Typography 
                            variant="body2" 
                            sx={{ 
                              fontWeight: 600, 
                              color: queueDepth > 1000 ? 'error.main' : 'warning.main'
                            }}
                          >
                            {queueDepth.toLocaleString()}
                          </Typography>
                        </Box>
                      )}
                    </Box>

                    {/* Actions */}
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <IconButton
                        size="small"
                        onClick={(e) => { e.stopPropagation(); handleToggle(source); }}
                        sx={{ 
                          bgcolor: source.enabled ? 'error.light' : 'success.light',
                          '&:hover': { bgcolor: source.enabled ? 'error.main' : 'success.main' }
                        }}
                      >
                        {source.enabled ? <Stop fontSize="small" /> : <PlayArrow fontSize="small" />}
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={(e) => { e.stopPropagation(); setDetailFor(source); }}
                        sx={{ bgcolor: 'info.light', '&:hover': { bgcolor: 'info.main' } }}
                      >
                        <Visibility fontSize="small" />
                      </IconButton>
                      {source.type === 'syslog' && (
                        <IconButton
                          size="small"
                          onClick={(e) => { e.stopPropagation(); handleDownloadVersaCerts(); }}
                          title="Download Versa SD-WAN TLS Certificates"
                          sx={{ bgcolor: 'secondary.light', '&:hover': { bgcolor: 'secondary.main' } }}
                        >
                          <Security fontSize="small" />
                        </IconButton>
                      )}
                      {source.type === 'akamai_ds2' && (
                        <IconButton
                          size="small"
                          onClick={(e) => { e.stopPropagation(); setAkamaiFor(source); }}
                          title="Akamai Tool Workbench"
                          sx={{ bgcolor: 'primary.light', '&:hover': { bgcolor: 'primary.main' } }}
                        >
                          <Cloud fontSize="small" />
                        </IconButton>
                      )}
                      <IconButton
                        size="small"
                        onClick={(e) => { e.stopPropagation(); setLoadTestFor(source); }}
                        title="Load test"
                        sx={{ bgcolor: 'warning.light', '&:hover': { bgcolor: 'warning.main' } }}
                      >
                        <Speed fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingSource(source);
                          setOpen(true);
                        }}
                        sx={{ bgcolor: 'grey.200', '&:hover': { bgcolor: 'grey.400' } }}
                      >
                        <Edit fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={(e) => { e.stopPropagation(); handleDelete(source.id); }}
                        sx={{ bgcolor: 'error.light', '&:hover': { bgcolor: 'error.main' } }}
                      >
                        <Delete fontSize="small" />
                      </IconButton>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            );
          })}
        </Grid>
      )}

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          {editingSource.id ? 'Edit Source' : 'Add Source'}
        </DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="Name"
            value={editingSource.name || ''}
            onChange={(e) => setEditingSource({ ...editingSource, name: e.target.value })}
            margin="normal"
          />
          <FormControl fullWidth margin="normal">
            <InputLabel>Type</InputLabel>
            <Select
              value={editingSource.type}
              label="Type"
              onChange={(e: SelectChangeEvent<Source['type']>) => setEditingSource({
                ...editingSource,
                type: e.target.value as Source['type'],
                config: {}
              })}
            >
              <MenuItem value="syslog">Syslog</MenuItem>
              <MenuItem value="http">HTTP</MenuItem>
              <MenuItem value="kafka">Kafka</MenuItem>
              <MenuItem value="file">File</MenuItem>
              <MenuItem value="windows_event">Windows Event Log</MenuItem>
              <MenuItem value="akamai_ds2">Akamai DataStream 2</MenuItem>
            </Select>
          </FormControl>
          {renderConfigFields()}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained">Save</Button>
        </DialogActions>
      </Dialog>

      <SourceDetailModal source={detailFor} onClose={()=>setDetailFor(null)} />
  {akamaiFor && <AkamaiWorkbench sourceId={akamaiFor.id} sourceName={akamaiFor.name} onClose={()=>setAkamaiFor(null)} />}
  {loadTestFor && <LoadTestWorkbench sourceId={loadTestFor.id} sourceName={loadTestFor.name} sourceConfig={loadTestFor.config} onClose={()=>setLoadTestFor(null)} />}
    </Box>
  );
}

function SourceDetailModal({source, onClose}:{source: Source|null; onClose: ()=>void}){
  const [lines, setLines] = useState<string[]>([])
  const [capId, setCapId] = useState<string|undefined>()
  const [capPath, setCapPath] = useState<string|undefined>()
  const logBoxRef = useRef<HTMLDivElement|null>(null)
  const [connected, setConnected] = useState<boolean>(false)
  const [connErr, setConnErr] = useState<string|undefined>()
  useEffect(()=>{
    if(!source) return
    setLines([])
    setConnected(false)
    setConnErr(undefined)
    const ev = new EventSource(`/api/v1/sources/${source.id}/stream?tail=15`)
    ev.onmessage = (e)=>{
      setLines(prev => [...prev.slice(-14), e.data])
      if (!connected) setConnected(true)
    }
    ev.onerror = (e)=>{
      setConnErr('stream error')
    }
    return ()=>{ ev.close() }
  }, [source?.id])
  useEffect(()=>{
    const el = logBoxRef.current
    if (!el) return
    el.scrollTop = el.scrollHeight
  }, [lines])
  const startCap = async (format: 'log'|'json') => {
    const r = await apiClient.post(`/api/v1/sources/${source?.id}/capture/start`, {format})
    setCapId(r.data.captureId)
    if (r.data.path) setCapPath(r.data.path.split('/')?.pop?.() || r.data.path)
  }
  const stopCap = async () => {
    if(!capId || !source) return
    await apiClient.post(`/api/v1/sources/${source.id}/capture/stop/${capId}`)
    setCapId(undefined)
    // If we have a captured file, signal the Filters page to open and select it
    if (capPath) {
      const file = capPath.includes('sandbox/library/') ? capPath.split('sandbox/library/')[1] : capPath
      window.dispatchEvent(new CustomEvent('open-filters', { detail: { file } }))
    }
  }
  return (
    <Dialog open={!!source} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Source: {source?.name}</DialogTitle>
      <DialogContent>
        <Box sx={{display:'flex', gap:2, alignItems:'center', mb:1}}>
          <Chip size="small" label={source?.type} />
          <Chip size="small" color={source?.status==='running'?'success':'default'} label={source?.status} />
          <Chip size="small" color={connected?'success':'default'} label={connected? 'connected' : 'connecting…'} />
          <Box sx={{flex:1}} />
          {!capId ? (
            <>
              <Button startIcon={<Download/>} onClick={()=>startCap('log')}>Start .log capture</Button>
              <Button startIcon={<Download/>} onClick={()=>startCap('json')}>Start .json capture</Button>
            </>
          ) : (
            <Button color="error" onClick={stopCap}>Stop Capture</Button>
          )}
        </Box>
        <Paper variant="outlined" sx={{p:1, maxHeight:400, overflow:'auto', bgcolor:'#0b1020'}} ref={logBoxRef}>
          <pre className="light-text">
            {lines.map((l,i)=> <div key={i}>{l}</div>)}
          </pre>
        </Paper>
  {connErr && <Typography variant="caption" color="error">{connErr}</Typography>}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  )
}
