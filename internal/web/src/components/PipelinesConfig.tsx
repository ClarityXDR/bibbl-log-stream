import React, { useEffect, useState } from 'react';
import {
  Box, Typography, Button, Dialog, DialogTitle, DialogContent, DialogActions,
  TextField, Table, TableHead, TableRow, TableCell, TableBody, IconButton,
  Stack, Select, MenuItem
} from '@mui/material';
import { Add, Edit, Delete, Clear } from '@mui/icons-material';
import { apiClient } from '../utils/apiClient';

interface Pipeline {
  id: string;
  name: string;
  description: string;
  functions: string[];
  filters?: PipelineFilter[];
}

interface PipelineFilter {
  field: string;
  values: string[];
  mode?: 'include' | 'exclude';
}

interface PipelineStat {
  id: string;
  name: string;
  filtered: number;
}

export default function PipelinesConfig() {
  const [pipelines, setPipelines] = useState<Pipeline[]>([]);
  const [open, setOpen] = useState(false);
  const blankPipeline = () => ({ functions: [], filters: [] as PipelineFilter[] });
  const [editing, setEditing] = useState<Partial<Pipeline>>(blankPipeline());
  const [stats, setStats] = useState<Record<string, number>>({});
  const addFilterRow = () => {
    const next = [...(editing.filters || [])];
    next.push({ field: '', values: [], mode: 'include' });
    setEditing({ ...editing, filters: next });
  };
  const updateFilter = (index: number, patch: Partial<PipelineFilter>) => {
    const next = [...(editing.filters || [])];
    next[index] = { ...next[index], ...patch };
    setEditing({ ...editing, filters: next });
  };
  const removeFilterRow = (index: number) => {
    const next = [...(editing.filters || [])];
    next.splice(index, 1);
    setEditing({ ...editing, filters: next });
  };

  const load = async () => {
    const [pipelinesRes, statsRes] = await Promise.all([
      apiClient.get('/api/v1/pipelines'),
      apiClient.get('/api/v1/pipelines/stats')
    ]);
    const pipelineItems = unwrapItems<Pipeline>(pipelinesRes.data);
    setPipelines(pipelineItems);
    const statMap: Record<string, number> = {};
    unwrapItems<PipelineStat>(statsRes.data).forEach(s => {
      statMap[s.id] = s.filtered ?? 0;
    });
    setStats(statMap);
  };
  useEffect(() => { load(); }, []);

  const save = async () => {
    if (editing.id) {
      await apiClient.put(`/api/v1/pipelines/${editing.id}`, editing);
    } else {
      await apiClient.post('/api/v1/pipelines', editing);
    }
    setOpen(false); setEditing(blankPipeline()); load();
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
        <Button variant="contained" startIcon={<Add />} onClick={() => { setEditing(blankPipeline()); setOpen(true); }}>Add Pipeline</Button>
      </Box>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Name</TableCell>
            <TableCell>Description</TableCell>
            <TableCell>Filters</TableCell>
            <TableCell>Filtered Events</TableCell>
            <TableCell>Functions</TableCell>
            <TableCell>Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {pipelines.map(p => (
            <TableRow key={p.id}>
              <TableCell>{p.name}</TableCell>
              <TableCell>{p.description}</TableCell>
              <TableCell>{renderFilters(p.filters)}</TableCell>
              <TableCell>{stats[p.id] ?? 0}</TableCell>
              <TableCell>{p.functions?.join(', ')}</TableCell>
              <TableCell>
                <IconButton onClick={() => { setEditing({ ...p, filters: p.filters || [] }); setOpen(true); }}><Edit /></IconButton>
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
          <Box sx={{ mt: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
              <Typography variant="subtitle2">Filters</Typography>
              <Button size="small" startIcon={<Add />} onClick={addFilterRow}>Add Filter</Button>
            </Box>
            {(editing.filters || []).length === 0 && (
              <Typography variant="body2" color="text.secondary">No filters configured.</Typography>
            )}
            <Stack spacing={2}>
              {(editing.filters || []).map((filter, idx) => (
                <Box key={`filter-${idx}`} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 2 }}>
                  <Stack spacing={2} direction={{ xs: 'column', sm: 'row' }}>
                    <TextField fullWidth label="Field" value={filter.field || ''} onChange={e => updateFilter(idx, { field: e.target.value })} />
                    <Select
                      fullWidth
                      value={filter.mode || 'include'}
                      onChange={e => updateFilter(idx, { mode: e.target.value as PipelineFilter['mode'] })}
                    >
                      <MenuItem value="include">Include</MenuItem>
                      <MenuItem value="exclude">Exclude</MenuItem>
                    </Select>
                    <TextField
                      fullWidth
                      label="Values (comma-separated)"
                      value={(filter.values || []).join(', ')}
                      onChange={e => updateFilter(idx, { values: e.target.value.split(',').map(v => v.trim()).filter(Boolean) })}
                    />
                    <IconButton color="error" onClick={() => removeFilterRow(idx)}><Clear /></IconButton>
                  </Stack>
                </Box>
              ))}
            </Stack>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={save}>Save</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

function renderFilters(filters?: PipelineFilter[]) {
  if (!filters || filters.length === 0) {
    return '—';
  }
  return filters
    .map((f) => {
      const modeLabel = (f.mode || 'include') === 'include' ? 'include' : 'exclude';
      const vals = (f.values || []).join(', ');
      return `${f.field || '(field)'} ${modeLabel}: ${vals}`;
    })
    .join('; ');
}

function unwrapItems<T = any>(payload: any): T[] {
  if (Array.isArray(payload)) {
    return payload as T[];
  }
  if (payload?.items && Array.isArray(payload.items)) {
    return payload.items as T[];
  }
  return [];
}
