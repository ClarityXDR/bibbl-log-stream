import React, { useState, useEffect, useRef } from 'react'
import './styles.css'
import {
  Box, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Select, MenuItem, FormControl, InputLabel, Grid,
  Table, TableBody, TableCell, TableHead, TableRow, IconButton,
  Paper, Typography, Chip
} from '@mui/material';
import { SelectChangeEvent } from '@mui/material/Select';
import { Add, Edit, Delete, PlayArrow, Stop, Visibility, Download, Cloud, Speed, Queue } from '@mui/icons-material';
import AkamaiWorkbench from './AkamaiWorkbench';
import LoadTestWorkbench from './LoadTestWorkbench';
import { apiClient } from '../utils/apiClient';

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

  useEffect(() => {
    loadSources();
  }, []);

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
    try {
      if (editingSource.id) {
        await apiClient.put(`/api/v1/sources/${editingSource.id}`, editingSource);
      } else {
        await apiClient.post('/api/v1/sources', editingSource);
      }
      setOpen(false);
      setEditingSource({ type: 'syslog', config: {} });
      loadSources();
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

  return (
    <Box>
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between' }}>
        <Typography variant="h6">Data Sources</Typography>
        <Button
          variant="contained"
          startIcon={<Add />}
          onClick={() => {
            setEditingSource({ type: 'syslog', config: {} });
            setOpen(true);
          }}
        >
          Add Source
        </Button>
      </Box>

      <Table>
        {actionError && (
          <Box sx={{mb:1}}>
            <Paper elevation={0} sx={{p:1, bgcolor:'#4b1d1d', color:'#fff', fontSize:13, display:'flex', justifyContent:'space-between', alignItems:'center'}}>
              <span>{actionError}</span>
              <Button size="small" variant="text" sx={{color:'#fff'}} onClick={()=>setActionError('')}>Dismiss</Button>
            </Paper>
          </Box>
        )}
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Type</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Flow</TableCell>
            <TableCell>Queue</TableCell>
            <TableCell>Configuration</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {sources.map((source) => (
            <TableRow key={source.id}>
              <TableCell>{source.name}</TableCell>
              <TableCell>{source.type}</TableCell>
              <TableCell>
                <Chip
                  label={source.enabled ? 'On' : 'Off'}
                  color={source.enabled ? 'success' : 'default'}
                  size="small"
                />
              </TableCell>
              <TableCell>
                {(() => {
                  const secs = source.lastUnix ? Math.max(0, Math.floor(Date.now()/1000 - source.lastUnix)) : -1;
                  const hint = secs < 0 ? 'never' : (secs === 0 ? 'just now' : `${secs}s ago`);
                  return (
                    <span title={`Last log: ${hint}`}>
                      <Chip
                        label={source.flow ? 'On' : 'Off'}
                        color={source.flow ? 'success' : 'default'}
                        size="small"
                      />
                    </span>
                  )
                })()}
              </TableCell>
              <TableCell>
                {(() => {
                  const depth = source.queueDepth ?? 0;
                  // Color logic: 0=default, >0=warning, very large (>1000)=error
                  let color: 'default' | 'warning' | 'error' = 'default';
                  if (depth > 0) color = depth > 1000 ? 'error' : 'warning';
                  return (
                    <span title={`Queue depth: ${depth}`}> <Queue fontSize="small" color={color === 'default' ? 'inherit' : (color as any)} /> </span>
                  )
                })()}
              </TableCell>
              <TableCell>
                {source.type === 'syslog' && `Port: ${source.config.port}`}
                {source.type === 'http' && `${source.config.path}:${source.config.port}`}
                {source.type === 'kafka' && `Topics: ${source.config.topics}`}
              </TableCell>
              <TableCell>
                <IconButton
                  onClick={() => handleToggle(source)}
                  color={source.enabled ? 'error' : 'success'}
                >
                  {source.enabled ? <Stop /> : <PlayArrow />}
                </IconButton>
                <IconButton onClick={() => setDetailFor(source)}>
                  <Visibility />
                </IconButton>
                {source.type === 'akamai_ds2' && (
                  <IconButton onClick={()=> setAkamaiFor(source)} title="Akamai Tool Workbench">
                    <Cloud />
                  </IconButton>
                )}
                <IconButton onClick={()=> setLoadTestFor(source)} title="Load test from this source">
                  <Speed />
                </IconButton>
                <IconButton
                  onClick={() => {
                    setEditingSource(source);
                    setOpen(true);
                  }}
                >
                  <Edit />
                </IconButton>
                <IconButton onClick={() => handleDelete(source.id)} color="error">
                  <Delete />
                </IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

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
