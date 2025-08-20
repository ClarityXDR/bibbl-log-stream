import React, { useState, useEffect } from 'react';
import {
  Box, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Select, MenuItem, FormControl, InputLabel,
  Table, TableBody, TableCell, TableHead, TableRow, IconButton,
  Typography, Chip, Switch, FormControlLabel
} from '@mui/material';
import { Add, Edit, Delete, CloudUpload, CloudDone } from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';

interface Destination {
  id: string;
  name: string;
  type: 'sentinel' | 'splunk' | 's3' | 'azure_blob' | 'elasticsearch' | 'azure_datalake';
  config: Record<string, any>;
  status: 'connected' | 'disconnected' | 'error';
  enabled: boolean;
}

export default function DestinationsConfig() {
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

  useEffect(() => {
    loadDestinations();
  }, []);

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
    try {
      if (editingDest.id) {
        await apiClient.put(`/api/v1/destinations/${editingDest.id}`, editingDest);
      } else {
        await apiClient.post('/api/v1/destinations', editingDest);
      }
      setOpen(false);
      setEditingDest({ type: 'sentinel', config: {} });
      loadDestinations();
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

  return (
    <Box>
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between' }}>
        <Typography variant="h6">Destinations</Typography>
        <Button
          variant="contained"
          startIcon={<Add />}
          onClick={() => {
            setEditingDest({ type: 'sentinel', config: {} });
            setOpen(true);
          }}
        >
          Add Destination
        </Button>
      </Box>

      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Type</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Configuration</TableCell>
            <TableCell>Enabled</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {destinations.map((dest) => (
            <TableRow key={dest.id}>
              <TableCell>{dest.name}</TableCell>
              <TableCell>{dest.type}</TableCell>
              <TableCell>
                <Chip
                  label={dest.status}
                  color={dest.status === 'connected' ? 'success' : 
                         dest.status === 'error' ? 'error' : 'default'}
                  size="small"
                />
              </TableCell>
              <TableCell>
                {dest.type === 'sentinel' && `Workspace: ${dest.config.workspaceId?.slice(0, 8)}...`}
                {dest.type === 'splunk' && `${dest.config.hecUrl}`}
                {dest.type === 's3' && `s3://${dest.config.bucket}/${dest.config.prefix}`}
              </TableCell>
              <TableCell>
                <Switch
                  checked={dest.enabled}
                  onChange={async () => {
                    await apiClient.patch(`/api/v1/destinations/${dest.id}`, {
                      enabled: !dest.enabled
                    });
                    loadDestinations();
                  }}
                />
              </TableCell>
              <TableCell>
                <IconButton
                  onClick={() => {
                    setEditingDest(dest);
                    setOpen(true);
                  }}
                >
                  <Edit />
                </IconButton>
                <IconButton onClick={() => handleDelete(dest.id)} color="error">
                  <Delete />
                </IconButton>
                {(dest.type === 'sentinel' || dest.type === 'azure_datalake') && (
                  <>
                    <IconButton onClick={()=>{ setWorkbenchDest(dest); setWorkbenchOpen(true); }} title="Workbench">
                      <CloudUpload />
                    </IconButton>
                    <IconButton onClick={()=>{ setAzureModalOpen(true); pollAzureStatus(); }} title="Azure Automate">
                      <CloudDone />
                    </IconButton>
                  </>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

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
