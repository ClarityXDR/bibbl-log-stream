import React, { useEffect, useState } from 'react';
import {
  Box, Typography, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Table, TableHead, TableRow, TableCell, TableBody, IconButton
} from '@mui/material';
import { Add, Edit, Delete } from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';

interface Pipeline {
  id: string;
  name: string;
  description: string;
  functions: string[];
}

export default function PipelinesConfig() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Partial<Pipeline>>({ functions: [] });

  const load = async () => {
    const r = await apiClient.get('/api/v1/pipelines');
    setPipelines(r.data);
  };
  useEffect(() => { load(); }, []);

  const save = async () => {
    if (editing.id) {
      await apiClient.put(`/api/v1/pipelines/${editing.id}`, editing);
    } else {
      await apiClient.post('/api/v1/pipelines', editing);
    }
    setOpen(false); setEditing({ functions: [] }); load();
  };

  const del = async (id: string) => {
    if (!confirm('Delete this pipeline?')) return;
    await apiClient.delete(`/api/v1/pipelines/${id}`);
    load();
  };

  return (
    <Box>
      <Box sx={{ mb: 2, display: 'flex', justifyContent: 'space-between' }}>
        <Typography variant="h6">Pipelines</Typography>
        <Button variant="contained" startIcon={<Add />} onClick={() => { setEditing({ functions: [] }); setOpen(true); }}>Add Pipeline</Button>
      </Box>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Description</TableCell>
            <TableCell>Functions</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {pipelines.map(p => (
            <TableRow key={p.id}>
              <TableCell>{p.name}</TableCell>
              <TableCell>{p.description}</TableCell>
              <TableCell>{p.functions?.join(', ')}</TableCell>
              <TableCell>
                <IconButton onClick={() => { setEditing(p); setOpen(true); }}><Edit /></IconButton>
                <IconButton color="error" onClick={() => del(p.id)}><Delete /></IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editing.id ? 'Edit Pipeline' : 'Add Pipeline'}</DialogTitle>
        <DialogContent>
          <TextField fullWidth margin="normal" label="Name" value={editing.name || ''} onChange={e => setEditing({...editing, name: e.target.value})} />
          <TextField fullWidth margin="normal" label="Description" value={editing.description || ''} onChange={e => setEditing({...editing, description: e.target.value})} />
          <TextField fullWidth margin="normal" label="Functions (comma-separated)" value={(editing.functions||[]).join(', ')} onChange={e => setEditing({...editing, functions: e.target.value.split(',').map(s=>s.trim()).filter(Boolean)})} />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={save}>Save</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
