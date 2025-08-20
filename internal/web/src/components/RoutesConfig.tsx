import React, { useEffect, useState } from 'react';
import {
  Box, Typography, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Table, TableHead, TableRow, TableCell, TableBody, IconButton,
  FormControlLabel, Switch
} from '@mui/material';
import { Add, Edit, Delete } from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';

interface Route {
  id: string;
  name: string;
  filter: string;
  pipelineId: string;
  destination: string;
  final: boolean;
}

export default function RoutesConfig() {
  const [routes, setRoutes] = useState<Route[]>([]);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Partial<Route>>({ final: true });
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    try{
      setError(null);
      const r = await apiClient.get('/api/v1/routes');
      setRoutes(r.data);
    }catch(e:any){ setError(e?.message||'Failed to load routes') }
  };
  useEffect(() => { load(); }, []);

  const save = async () => {
    try{
      if (editing.id) {
        await apiClient.put(`/api/v1/routes/${editing.id}`, editing);
      } else {
        await apiClient.post('/api/v1/routes', editing);
      }
      setOpen(false); setEditing({ final: true }); load();
    }catch(e:any){ setError(e?.message||'Failed to save route') }
  };

  const del = async (id: string) => {
    if (!confirm('Delete this route?')) return;
    try{ await apiClient.delete(`/api/v1/routes/${id}`); load(); }
    catch(e:any){ setError(e?.message||'Failed to delete route') }
  };

  return (
    <Box>
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between' }}>
        <Typography variant="h6">Routes</Typography>
        <Button variant="contained" startIcon={<Add />} onClick={() => { setEditing({ final: true }); setOpen(true); }}>Add Route</Button>
      </Box>
  {error && <div style={{margin:'8px 0', color:'#b91c1c'}}>{error}</div>}
  <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Filter</TableCell>
            <TableCell>Pipeline</TableCell>
            <TableCell>Destination</TableCell>
            <TableCell>Final</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {routes.map(r => (
            <TableRow key={r.id}>
              <TableCell>{r.name}</TableCell>
              <TableCell>{r.filter}</TableCell>
              <TableCell>{r.pipelineId}</TableCell>
              <TableCell>{r.destination}</TableCell>
              <TableCell>{String(r.final)}</TableCell>
              <TableCell>
                <IconButton onClick={() => { setEditing(r); setOpen(true); }}><Edit /></IconButton>
                <IconButton color="error" onClick={() => del(r.id)}><Delete /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editing.id ? 'Edit Route' : 'Add Route'}</DialogTitle>
        <DialogContent>
          <TextField fullWidth margin="normal" label="Name" value={editing.name || ''} onChange={e => setEditing({...editing, name: e.target.value})} />
          <TextField fullWidth margin="normal" label="Filter (JS expression)" value={editing.filter || ''} onChange={e => setEditing({...editing, filter: e.target.value})} />
          <TextField fullWidth margin="normal" label="Pipeline ID" value={editing.pipelineId || ''} onChange={e => setEditing({...editing, pipelineId: e.target.value})} />
          <TextField fullWidth margin="normal" label="Destination ID" value={editing.destination || ''} onChange={e => setEditing({...editing, destination: e.target.value})} />
          <FormControlLabel control={<Switch checked={!!editing.final} onChange={e => setEditing({...editing, final: e.target.checked})} />} label="Final" />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={save}>Save</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
