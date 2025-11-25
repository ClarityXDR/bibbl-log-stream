import React, { useState, useEffect } from 'react';
import {
  Box, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Select, MenuItem, FormControl, InputLabel,
  Table, TableBody, TableCell, TableHead, TableRow, IconButton,
  Typography, Chip, Switch, FormControlLabel, LinearProgress, Snackbar, Alert, Tooltip, Paper,
  Grid, Card, CardContent
} from '@mui/material';
import { Add, Edit, Delete, CloudUpload, CloudDone, ArrowBack as BackIcon, Save as SaveIcon } from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';
import DestinationTemplateGallery from './destinations/DestinationTemplateGallery';
import { DestinationTemplate } from './destinations/destinationTemplates';

interface Destination {
  id: string;
  name: string;
  type: 'sentinel' | 'splunk' | 's3' | 'azure_blob' | 'elasticsearch' | 'azure_datalake' | 'azure_loganalytics';
  config: Record<string, any>;
  status: 'connected' | 'disconnected' | 'error';
  enabled: boolean;
}

export default function DestinationsConfig() {
  const [currentView, setCurrentView] = useState<'gallery' | 'list' | 'builder'>('list');
  const [selectedTemplate, setSelectedTemplate] = useState<DestinationTemplate | null>(null);
  const [testMode, setTestMode] = useState<boolean>(true);
  const [destinations, setDestinations] = useState<Destination[]>([]);
  const [open, setOpen] = useState(false);
  const [editingDest, setEditingDest] = useState<Partial<Destination>>({
    type: 'sentinel',
    config: {}
  });
  const [workbenchOpen, setWorkbenchOpen] = useState(false);
  const [workbenchDest, setWorkbenchDest] = useState<Destination | null>(null);
  const [azureModalOpen, setAzureModalOpen] = useState(false);
  const [azureStatus, setAzureStatus] = useState<any>(null);
  const [azureBusy, setAzureBusy] = useState(false);
  const [tenantId, setTenantId] = useState<string>('');
  const [saveOk, setSaveOk] = useState(false);

  useEffect(() => {
    loadDestinations();
  }, []);

  const handleSelectTemplate = (template: DestinationTemplate) => {
    setSelectedTemplate(template);
    setCurrentView('builder');
    setEditingDest({
      name: template.config.name,
      type: template.config.type,
      config: template.config.config
    });
  };

  const handleBack = () => {
    if (confirm('Go back? Your current work will be lost.')) {
      setCurrentView('list');
      setEditingDest({ type: 'sentinel', config: {} });
      setSelectedTemplate(null);
    }
  };

  const getProgress = () => {
    let completed = 0;
    const total = 3;
    if (editingDest.name && editingDest.name.trim().length > 0) completed++;
    if (editingDest.type) completed++;
    if (editingDest.config && Object.keys(editingDest.config).length > 0) completed++;
    return (completed / total) * 100;
  };

  const loadDestinations = async () => {
    try {
  const response = await apiClient.get('/api/v1/destinations');
  const data = response.data;
  const items = Array.isArray(data) ? data : (Array.isArray(data?.items) ? data.items : []);
  setDestinations(items);
    } catch (error) {
      console.error('Failed to load destinations:', error);
    }
  };

  const handleSave = async () => {
    if (testMode && currentView === 'builder') {
      if (confirm('🧪 You are in Test Mode. Switch to Live Mode to save this destination for real?')) {
        setTestMode(false);
      }
      return;
    }
    try {
      if (editingDest.id) {
        await apiClient.put(`/api/v1/destinations/${editingDest.id}`, editingDest);
      } else {
        await apiClient.post('/api/v1/destinations', editingDest);
      }
      setSaveOk(true);
      setTimeout(() => {
        setOpen(false);
        setCurrentView('list');
        setEditingDest({ type: 'sentinel', config: {} });
        setSelectedTemplate(null);
        loadDestinations();
      }, 2000);
    } catch (error) {
      console.error('Failed to save destination:', error);
    }
  };

  const renderConfigFields = () => {
    switch (editingDest.type) {
  case 'sentinel':
        return (
          <>
            <TextField
              fullWidth
              label="Workspace ID"
              value={editingDest.config?.workspaceId || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, workspaceId: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="DCE Endpoint"
              value={editingDest.config?.dceEndpoint || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, dceEndpoint: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="DCR ID"
              value={editingDest.config?.dcrId || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, dcrId: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Table Name"
              value={editingDest.config?.tableName || 'Custom_BibblLogs_CL'}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, tableName: e.target.value }
              })}
              margin="normal"
            />
            <TextField fullWidth type="number" label="Batch Max Events" value={editingDest.config?.batchMaxEvents || 500}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxEvents:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Batch Max Bytes" value={editingDest.config?.batchMaxBytes || 524288}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxBytes:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Flush Interval (s)" value={editingDest.config?.flushIntervalSec || 5}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, flushIntervalSec:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Concurrency" value={editingDest.config?.concurrency || 2}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, concurrency:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth label="Compression" value={editingDest.config?.compression || 'gzip'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, compression:e.target.value}})} margin="normal" />
          </>
        );
      case 'azure_loganalytics':
        return (
          <>
            <TextField
              fullWidth
              label="Workspace ID"
              value={editingDest.config?.workspaceID || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, workspaceID: e.target.value }
              })}
              margin="normal"
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            />
            <TextField
              fullWidth
              label="Shared Key"
              type="password"
              value={editingDest.config?.sharedKey || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, sharedKey: e.target.value }
              })}
              margin="normal"
              placeholder="Base64-encoded primary key"
            />
            <TextField
              fullWidth
              label="Log Type (Table Name)"
              value={editingDest.config?.logType || 'SecurityAlerts'}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, logType: e.target.value }
              })}
              margin="normal"
              helperText="Table will be created as [LogType]_CL automatically"
            />
            <TextField
              fullWidth
              label="Resource Group (optional)"
              value={editingDest.config?.resourceGroup || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, resourceGroup: e.target.value }
              })}
              margin="normal"
            />
            <TextField fullWidth type="number" label="Batch Max Events" value={editingDest.config?.batchMaxEvents || 500}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxEvents:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Batch Max Bytes" value={editingDest.config?.batchMaxBytes || 1048576}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxBytes:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Flush Interval (s)" value={editingDest.config?.flushIntervalSec || 10}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, flushIntervalSec:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Concurrency" value={editingDest.config?.concurrency || 2}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, concurrency:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Max Retries" value={editingDest.config?.maxRetries || 3}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, maxRetries:parseInt(e.target.value)||0}})} margin="normal" />
          </>
        );
      case 'splunk':
        return (
          <>
            <TextField
              fullWidth
              label="HEC URL"
              value={editingDest.config?.hecUrl || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, hecUrl: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="HEC Token"
              type="password"
              value={editingDest.config?.hecToken || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, hecToken: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Index"
              value={editingDest.config?.index || 'main'}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, index: e.target.value }
              })}
              margin="normal"
            />
          </>
        );
  case 's3':
        return (
          <>
            <TextField
              fullWidth
              label="Bucket"
              value={editingDest.config?.bucket || ''}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, bucket: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Region"
              value={editingDest.config?.region || 'us-east-1'}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, region: e.target.value }
              })}
              margin="normal"
            />
            <TextField
              fullWidth
              label="Prefix"
              value={editingDest.config?.prefix || 'logs/'}
              onChange={(e) => setEditingDest({
                ...editingDest,
                config: { ...editingDest.config, prefix: e.target.value }
              })}
              margin="normal"
            />
          </>
        );
      case 'azure_datalake':
        return (
          <>
            <TextField fullWidth label="Storage Account" value={editingDest.config?.storageAccount || ''}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, storageAccount:e.target.value}})} margin="normal" />
            <TextField fullWidth label="Filesystem" value={editingDest.config?.filesystem || 'logs'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, filesystem:e.target.value}})} margin="normal" />
            <TextField fullWidth label="Directory" value={editingDest.config?.directory || 'bibbl/raw/'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, directory:e.target.value}})} margin="normal" />
            <TextField fullWidth label="Path Template" value={editingDest.config?.pathTemplate || 'bibbl/raw/${yyyy}/${MM}/${dd}/${HH}/data-${mm}.jsonl'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, pathTemplate:e.target.value}})} margin="normal" />
            <TextField fullWidth label="Format" value={editingDest.config?.format || 'jsonl'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, format:e.target.value}})} margin="normal" />
            <TextField fullWidth label="Compression" value={editingDest.config?.compression || 'gzip'}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, compression:e.target.value}})} margin="normal" />
            <TextField fullWidth type="number" label="Batch Max Events" value={editingDest.config?.batchMaxEvents || 1000}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxEvents:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Batch Max Bytes" value={editingDest.config?.batchMaxBytes || 1048576}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, batchMaxBytes:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Flush Interval (s)" value={editingDest.config?.flushIntervalSec || 5}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, flushIntervalSec:parseInt(e.target.value)||0}})} margin="normal" />
            <TextField fullWidth type="number" label="Concurrency" value={editingDest.config?.concurrency || 4}
              onChange={(e)=>setEditingDest({...editingDest, config:{...editingDest.config, concurrency:parseInt(e.target.value)||0}})} margin="normal" />
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
        <DestinationTemplateGallery onSelectTemplate={handleSelectTemplate} />
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
                  {selectedTemplate?.icon} {selectedTemplate?.title || 'Custom Destination'}
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
                {selectedTemplate?.description || 'Configure your custom destination'}
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
              <Tooltip title="Back to destinations list">
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
            label="Destination Name"
            value={editingDest.name || ''}
            onChange={(e) => setEditingDest({ ...editingDest, name: e.target.value })}
            margin="normal"
            required
            helperText="Give your destination a descriptive name"
          />
          <FormControl fullWidth margin="normal">
            <InputLabel>Type</InputLabel>
            <Select
              value={editingDest.type}
              onChange={(e) => setEditingDest({
                ...editingDest,
                type: e.target.value as Destination['type'],
                config: {}
              })}
            >
              <MenuItem value="sentinel">Microsoft Sentinel</MenuItem>
              <MenuItem value="azure_loganalytics">Azure Log Analytics</MenuItem>
              <MenuItem value="splunk">Splunk</MenuItem>
              <MenuItem value="s3">Amazon S3</MenuItem>
              <MenuItem value="azure_blob">Azure Blob Storage</MenuItem>
              <MenuItem value="azure_datalake">Azure Data Lake Gen2</MenuItem>
              <MenuItem value="elasticsearch">Elasticsearch</MenuItem>
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
              disabled={!editingDest.name || !editingDest.type}
              sx={{
                py: 1.5,
                px: 4,
                fontSize: '1.1rem',
                fontWeight: 700,
                background: (editingDest.name && editingDest.type)
                  ? 'linear-gradient(45deg, #4CAF50 30%, #8BC34A 90%)'
                  : undefined,
                boxShadow: (editingDest.name && editingDest.type) ? '0 3px 5px 2px rgba(76, 175, 80, .3)' : undefined,
                '&:hover': (editingDest.name && editingDest.type) ? {
                  background: 'linear-gradient(45deg, #45A049 30%, #7CB342 90%)',
                  transform: 'scale(1.05)',
                  boxShadow: '0 6px 10px 4px rgba(76, 175, 80, .4)'
                } : undefined
              }}
            >
              {testMode ? '\ud83e\uddea Test Save' : '\ud83c\udf89 Save Destination!'}
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
            \ud83c\udf89 {testMode ? 'Test completed! Changes not saved.' : 'Success! Destination saved and ready to use!'}
          </Alert>
        </Snackbar>
      </Paper>
    );
  }

  // Helper function to get destination icon and color
  const getDestinationVisuals = (type: string) => {
    switch (type) {
      case 'sentinel': return { icon: '🛡️', color: '#0078D4', bg: '#E6F2FF' };
      case 'azure_loganalytics': return { icon: '📊', color: '#008AD7', bg: '#E0F4FF' };
      case 'azure_datalake': return { icon: '🗄️', color: '#00BCF2', bg: '#E0F9FF' };
      case 'splunk': return { icon: '🔍', color: '#000000', bg: '#F0F0F0' };
      case 's3': return { icon: '☁️', color: '#FF9900', bg: '#FFF4E6' };
      case 'elasticsearch': return { icon: '🔎', color: '#00BFB3', bg: '#E0FBF9' };
      case 'azure_blob': return { icon: '📦', color: '#0089D6', bg: '#E6F4FF' };
      default: return { icon: '📤', color: '#95A5A6', bg: '#ECF0F1' };
    }
  };

  // Render list view (default) - Visual card layout
  return (
    <Box>
      <Box sx={{ mb: 3, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="h5" sx={{ fontWeight: 700 }}>Destinations</Typography>
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
              setEditingDest({ type: 'sentinel', config: {} });
              setOpen(true);
            }}
            sx={{ borderRadius: 2 }}
          >
            Add Destination
          </Button>
        </Box>
      </Box>

      {destinations.length === 0 ? (
        <Paper elevation={3} sx={{ p: 6, textAlign: 'center', borderRadius: 3, bgcolor: 'background.default' }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No destinations configured yet
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Get started by choosing a template or creating a custom destination
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
          {destinations.map((dest) => {
            const visuals = getDestinationVisuals(dest.type);
            const configSummary = 
              dest.type === 'sentinel' ? `Workspace ${dest.config.workspaceId?.slice(0, 8)}...` :
              dest.type === 'splunk' ? `${dest.config.hecUrl}` :
              dest.type === 's3' ? `s3://${dest.config.bucket}/${dest.config.prefix || ''}` :
              dest.type === 'azure_blob' ? `${dest.config.storageAccount}` :
              dest.type === 'azure_datalake' ? `${dest.config.filesystem}` :
              dest.type === 'elasticsearch' ? `${dest.config.url}` :
              'Configuration available';

            return (
              <Grid item xs={12} sm={6} md={4} key={dest.id}>
                <Card 
                  elevation={3}
                  sx={{
                    height: '100%',
                    borderRadius: 3,
                    transition: 'all 0.3s ease',
                    border: dest.enabled ? `2px solid ${visuals.color}` : '2px solid transparent',
                    '&:hover': {
                      transform: 'translateY(-4px)',
                      boxShadow: 6,
                      border: `2px solid ${visuals.color}`,
                    },
                  }}
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
                      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, alignItems: 'flex-end' }}>
                        <Chip
                          label={dest.status || 'unknown'}
                          size="small"
                          color={
                            dest.status === 'connected' ? 'success' : 
                            dest.status === 'error' ? 'error' : 'default'
                          }
                          sx={{ fontWeight: 600, fontSize: 10 }}
                        />
                        <Switch
                          size="small"
                          checked={dest.enabled}
                          onChange={async (e) => {
                            e.stopPropagation();
                            await apiClient.patch(`/api/v1/destinations/${dest.id}`, {
                              enabled: !dest.enabled
                            });
                            loadDestinations();
                          }}
                          sx={{ ml: -1 }}
                        />
                      </Box>
                    </Box>

                    {/* Name and Type */}
                    <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5, color: visuals.color }}>
                      {dest.name}
                    </Typography>
                    <Typography variant="caption" sx={{ color: 'text.secondary', textTransform: 'uppercase', letterSpacing: 1, fontWeight: 600 }}>
                      {dest.type.replace(/_/g, ' ')}
                    </Typography>

                    {/* Configuration */}
                    <Box sx={{ mt: 2, mb: 2, p: 1.5, bgcolor: 'background.default', borderRadius: 1.5 }}>
                      <Typography variant="body2" sx={{ fontSize: 12, color: 'text.secondary', wordBreak: 'break-all' }}>
                        {configSummary}
                      </Typography>
                    </Box>

                    {/* Actions */}
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                      <IconButton
                        size="small"
                        onClick={() => {
                          setEditingDest(dest);
                          setOpen(true);
                        }}
                        sx={{ bgcolor: 'primary.light', '&:hover': { bgcolor: 'primary.main' } }}
                      >
                        <Edit fontSize="small" />
                      </IconButton>
                      <IconButton
                        size="small"
                        onClick={() => handleDelete(dest.id)}
                        sx={{ bgcolor: 'error.light', '&:hover': { bgcolor: 'error.main' } }}
                      >
                        <Delete fontSize="small" />
                      </IconButton>
                      {(dest.type === 'sentinel' || dest.type === 'azure_datalake') && (
                        <>
                          <IconButton
                            size="small"
                            onClick={() => { setWorkbenchDest(dest); setWorkbenchOpen(true); }}
                            title="Workbench"
                            sx={{ bgcolor: 'info.light', '&:hover': { bgcolor: 'info.main' } }}
                          >
                            <CloudUpload fontSize="small" />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => { setAzureModalOpen(true); pollAzureStatus(); }}
                            title="Azure Automate"
                            sx={{ bgcolor: 'success.light', '&:hover': { bgcolor: 'success.main' } }}
                          >
                            <CloudDone fontSize="small" />
                          </IconButton>
                        </>
                      )}
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
          {editingDest.id ? 'Edit Destination' : 'Add Destination'}
        </DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            label="Name"
            value={editingDest.name || ''}
            onChange={(e) => setEditingDest({ ...editingDest, name: e.target.value })}
            margin="normal"
          />
          <FormControl fullWidth margin="normal">
            <InputLabel>Type</InputLabel>
            <Select
              value={editingDest.type}
              onChange={(e) => setEditingDest({
                ...editingDest,
                type: e.target.value as Destination['type'],
                config: {}
              })}
            >
              <MenuItem value="sentinel">Microsoft Sentinel</MenuItem>
              <MenuItem value="azure_loganalytics">Azure Log Analytics</MenuItem>
              <MenuItem value="splunk">Splunk</MenuItem>
              <MenuItem value="s3">Amazon S3</MenuItem>
              <MenuItem value="azure_blob">Azure Blob Storage</MenuItem>
              <MenuItem value="azure_datalake">Azure Data Lake Gen2</MenuItem>
              <MenuItem value="elasticsearch">Elasticsearch</MenuItem>
            </Select>
          </FormControl>
          {renderConfigFields()}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained">Save</Button>
        </DialogActions>
      </Dialog>

      {/* Workbench Modal */}
      <Dialog open={workbenchOpen} onClose={()=>setWorkbenchOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Destination Workbench - {workbenchDest?.name}</DialogTitle>
        <DialogContent dividers>
          {workbenchDest && (
            <Box sx={{display:'flex', flexDirection:'column', gap:2}}>
              <Typography variant="subtitle2" color="text.secondary">Performance & Batching</Typography>
              <TextField type="number" label="Batch Max Events" value={workbenchDest.config.batchMaxEvents || ''}
                onChange={(e)=>updateWorkbenchCfg({batchMaxEvents: parseInt(e.target.value)||0})} />
              <TextField type="number" label="Batch Max Bytes" value={workbenchDest.config.batchMaxBytes || ''}
                onChange={(e)=>updateWorkbenchCfg({batchMaxBytes: parseInt(e.target.value)||0})} />
              <TextField type="number" label="Flush Interval (s)" value={workbenchDest.config.flushIntervalSec || ''}
                onChange={(e)=>updateWorkbenchCfg({flushIntervalSec: parseInt(e.target.value)||0})} />
              <TextField type="number" label="Concurrency" value={workbenchDest.config.concurrency || ''}
                onChange={(e)=>updateWorkbenchCfg({concurrency: parseInt(e.target.value)||0})} />
              <TextField label="Compression" value={workbenchDest.config.compression || ''}
                onChange={(e)=>updateWorkbenchCfg({compression: e.target.value})} />
              {workbenchDest.type === 'azure_datalake' && (
                <>
                  <Typography variant="subtitle2" color="text.secondary">Azure Data Lake Pathing</Typography>
                  <TextField label="Path Template" value={workbenchDest.config.pathTemplate || ''}
                    onChange={(e)=>updateWorkbenchCfg({pathTemplate: e.target.value})} />
                  <TextField label="Directory" value={workbenchDest.config.directory || ''}
                    onChange={(e)=>updateWorkbenchCfg({directory: e.target.value})} />
                  <TextField label="Format" value={workbenchDest.config.format || ''}
                    onChange={(e)=>updateWorkbenchCfg({format: e.target.value})} />
                </>
              )}
              {workbenchDest.type === 'sentinel' && (
                <>
                  <Typography variant="subtitle2" color="text.secondary">Sentinel Mapping</Typography>
                  <TextField label="Table Name" value={workbenchDest.config.tableName || ''}
                    onChange={(e)=>updateWorkbenchCfg({tableName: e.target.value})} />
                </>
              )}
              <Typography variant="body2" color="text.secondary">Changes are applied immediately. Tune batch sizes for throughput vs latency.</Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={()=>setWorkbenchOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Azure Automation Modal */}
      <Dialog open={azureModalOpen} onClose={()=>setAzureModalOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Azure Automation</DialogTitle>
        <DialogContent dividers>
          <Box sx={{display:'flex', flexDirection:'column', gap:2}}>
            {!azureStatus?.authenticated && (
              <>
                <Typography variant="body2">Authenticate with Azure using device code flow.</Typography>
                <TextField label="Tenant ID (optional)" value={tenantId} onChange={(e)=>setTenantId(e.target.value)} fullWidth />
                <Button variant="contained" disabled={azureBusy} onClick={startAzureLogin}>Start Login</Button>
                {azureStatus?.userCode && (
                  <Box sx={{p:1, border:'1px dashed', borderColor:'divider'}}>
                    <Typography variant="body2">Code: <b>{azureStatus.userCode}</b></Typography>
                    <Typography variant="body2">Visit: <a href={azureStatus.verificationUrl} target="_blank" rel="noreferrer">{azureStatus.verificationUrl}</a></Typography>
                    <Typography variant="caption" color="text.secondary">{azureStatus.message}</Typography>
                  </Box>
                )}
              </>
            )}
            {azureStatus?.authenticated && (
              <>
                <Typography variant="body2" color="success.main">Authenticated with Azure.</Typography>
                <Button variant="outlined" disabled={azureBusy} onClick={()=>provision('sentinel')}>Provision Sentinel (stub)</Button>
                <Button variant="outlined" disabled={azureBusy} onClick={()=>provision('datalake')}>Provision Data Lake (stub)</Button>
              </>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={()=>setAzureModalOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );

  async function handleDelete(id: string) {
    if (window.confirm('Delete this destination?')) {
      try {
        await apiClient.delete(`/api/v1/destinations/${id}`);
        loadDestinations();
      } catch (error) {
        console.error('Failed to delete destination:', error);
      }
    }
  }

  function updateWorkbenchCfg(partial: Record<string, any>) {
    if (!workbenchDest) return;
    const updated = { ...workbenchDest, config: { ...workbenchDest.config, ...partial } } as Destination;
    setWorkbenchDest(updated);
    // Patch server (best-effort; ignore failures for now)
    apiClient.patch(`/api/v1/destinations/${workbenchDest.id}`, { config: updated.config })
      .then(()=>loadDestinations())
      .catch(err=>console.error('Workbench patch failed', err));
  }

  async function startAzureLogin() {
    setAzureBusy(true);
    try {
      await apiClient.post('/api/v1/azure/login/start', tenantId? {tenantId}:{ });
      pollAzureStatus();
    } catch (e) { console.error('azure login start failed', e); }
    finally { setAzureBusy(false); }
  }
  async function pollAzureStatus() {
    try {
      const r = await apiClient.get('/api/v1/azure/login/status');
      setAzureStatus(r.data);
      if(!r.data?.authenticated) setTimeout(pollAzureStatus, 2000);
    } catch (e) { console.error('azure status failed', e); }
  }
  async function provision(kind: 'sentinel'|'datalake') {
    setAzureBusy(true);
    try {
      const r = await apiClient.post(`/api/v1/azure/provision/${kind}`, {});
      console.log('provision resp', r.data);
    } catch (e) { console.error('provision failed', e); }
    finally { setAzureBusy(false); }
  }
}
