import React, { useEffect, useMemo, useRef, useState } from 'react'
import './styles.css'
import {
  Box,
  Paper,
  Typography,
  TextField,
  Select,
  MenuItem,
  Button,
  Chip,
  Divider,
  IconButton,
  Tooltip,
  Stepper,
  Step,
  StepLabel,
  ToggleButtonGroup,
  ToggleButton,
  LinearProgress,
  Snackbar,
  Alert,
  Checkbox,
  Stack
} from '@mui/material'
import { SelectChangeEvent, TextFieldProps } from '@mui/material'
import { apiClient } from '../utils/apiClient'
import {
  AltRoute as AltRouteIcon,
  FilterAlt as FilterAltIcon,
  RocketLaunch as RocketLaunchIcon,
  AutoFixHigh as AutoFixHighIcon,
  AddCircle as AddCircleIcon,
  Save as SaveIcon,
  PlayArrow as PlayArrowIcon,
  Refresh as RefreshIcon,
  Code as CodeIcon,
  Visibility as VisibilityIcon,
  ArrowUpward as ArrowUpwardIcon,
  ArrowDownward as ArrowDownwardIcon
} from '@mui/icons-material'

function useFetcher<T>(url: string, intervalMs?: number) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState<boolean>(false)
  const fetcher = useMemo(() => async () => {
    try {
      setLoading(true)
      setError(null)
      const r = await fetch(url)
      if (!r.ok) throw new Error(`${r.status}`)
      const json = await r.json()
      // Check if response has paginated format (items field) and extract the items array
      if (json && typeof json === 'object' && 'items' in json && Array.isArray(json.items)) {
        setData(json.items as T)
      } else {
        setData(json)
      }
    } catch (e: any) {
      setError(e?.message || 'error')
    } finally {
      setLoading(false)
    }
  }, [url])
  useEffect(() => {
    fetcher()
    if (!intervalMs) return
    const id = setInterval(fetcher, intervalMs)
    return () => clearInterval(id)
  }, [fetcher, intervalMs])
  return { data, error, loading, refresh: fetcher }
}

// Types for API data
type Pipeline = { id: string; name: string; description?: string; functions?: string[] }
type Destination = { id: string; name: string; type: string }
type Route = { id: string; name: string; filter: string; pipelineID: string; destination: string; final?: boolean }

// Debounce helper
function useDebouncedEffect(effect: () => void, deps: any[], delay: number) {
  useEffect(() => {
    const h = setTimeout(effect, delay)
    return () => clearTimeout(h)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, delay])
}

export default function TransformWorkbench({filtersInitialSelected}: {filtersInitialSelected?: string}) {
  // Data
  const routes = useFetcher<Route[]>('/api/v1/routes', 8000)
  const pipelines = useFetcher<Pipeline[]>('/api/v1/pipelines', 10000)
  const dests = useFetcher<Destination[]>('/api/v1/destinations', 10000)

  // Wizard + mode
  const steps = ['Route', 'Filter', 'Enrichment', 'Pipeline', 'Destination', 'Preview', 'Save']
  const [activeStep, setActiveStep] = useState(0)
  const [mode, setMode] = useState<'visual' | 'code'>('visual')

  // Selection and editor state
  const [selectedRouteId, setSelectedRouteId] = useState<string>('')
  const [routeName, setRouteName] = useState('New Route')
  const [pattern, setPattern] = useState('(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)')
  const [pipeId, setPipeId] = useState<string>('')
  const [destId, setDestId] = useState<string>('')

  // Sample editor
  const [before, setBefore] = useState<string>('10.0.0.1 app - - demo message 1')
  const [after, setAfter] = useState<string>('')

  // Run/eval
  const [err, setErr] = useState<string | undefined>()
  const [evaluating, setEvaluating] = useState(false)

  // Pipeline builder (visual + code JSON)
  type FnItem = { name: string; enabled: boolean }
  const [builderFns, setBuilderFns] = useState<FnItem[]>([])
  const [showPipelineJSON, setShowPipelineJSON] = useState(false)

  // Library helpers
  const [lib, setLib] = useState<{ name: string }[]>([])
  const [libSel, setLibSel] = useState('')
  const beforeRef = useRef<HTMLTextAreaElement | null>(null)

  const loadLib = async () => {
    try {
      const r = await fetch('/api/v1/library')
      setLib(await r.json())
    } catch {
      /* ignore */
    }
  }
  const loadLibFile = async (name: string) => {
    if (!name) return
    try {
      const r = await fetch(`/api/v1/library/${encodeURIComponent(name)}`)
      setBefore(await r.text())
    } catch (e: any) {
      setErr(String(e?.message || e))
    }
  }

  useEffect(() => { loadLib() }, [])
  useEffect(() => {
    if (filtersInitialSelected) { setLibSel(filtersInitialSelected); loadLibFile(filtersInitialSelected) }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtersInitialSelected])

  // Load selection if user picked a route
  useEffect(() => {
    if (!selectedRouteId || !routes.data || !Array.isArray(routes.data)) return
    const r = routes.data.find((x: Route) => x.id === selectedRouteId)
    if (!r) return
    setRouteName(r.name)
    setPattern(r.filter || pattern)
    setPipeId(r.pipelineID || '')
    setDestId(r.destination || '')
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedRouteId])

  // Sync pipeline builder when pipeline changes
  useEffect(() => {
    const fns = Array.isArray(pipelines.data) ? pipelines.data.find((p: Pipeline) => p.id === pipeId)?.functions || [] : []
    setBuilderFns(fns.map(n => ({ name: n, enabled: true })))
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pipeId, pipelines.data])

  // Live preview run (debounced)
  const run = async () => {
    setErr(undefined)
    setEvaluating(true)
    try {
      const lines = before.split(/\r?\n/).filter(Boolean).slice(0, 20)
      const outs: string[] = []
      // check if enrichment available
      let geoLoaded = false
      try { const s = await fetch('/api/v1/enrich/geoip/status'); const j = await s.json(); geoLoaded = !!j?.loaded } catch {}
      for (const line of lines) {
        try {
          const r = await fetch('/api/v1/preview/regex', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sample: line, pattern })
          })
          const j = await r.json()
          let obj: any = j.captures || {}
          if (geoLoaded) {
            try {
              const er = await fetch('/api/v1/preview/enrich', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ sample: line, pattern }) })
              const ej = await er.json(); if (ej?.enriched && ej.geo) { obj = { ...obj, geo: ej.geo } }
            } catch {}
          }
          outs.push(JSON.stringify(obj, null, 0))
        } catch (e: any) {
          outs.push('{}'); setErr(String(e?.message || e))
        }
      }
      setAfter(outs.join('\n'))
    } finally {
      setEvaluating(false)
    }
  }
  useEffect(() => { run() }, []) // initial
  useDebouncedEffect(() => { run() }, [pattern, before], 400)

  // Save / Apply route
  const [saveOk, setSaveOk] = useState(false)
  const saveRoute = async () => {
    const body = { name: routeName, filter: pattern, pipelineID: pipeId, destination: destId, final: true }
    if (selectedRouteId) {
      await apiClient.put(`/api/v1/routes/${selectedRouteId}`, body)
    } else {
      const r = await apiClient.post('/api/v1/routes', body)
      setSelectedRouteId(r.data?.id || '')
    }
    await routes.refresh()
    setSaveOk(true)
  }

  // Quick add helpers
  const quickAddRoute = () => {
    setSelectedRouteId('')
    setRouteName('New Route')
    setPattern('true')
    setPipeId(pipelines.data?.[0]?.id || '')
    setDestId(dests.data?.[0]?.id || '')
  }

  // Gating
  const gates = [
    routeName.trim().length > 0,
    pattern.trim().length > 0,
    true, // enrichment optional
    !!pipeId,
    !!destId,
    true,
    true
  ]
  const canNext = gates[activeStep]

  // Pipeline builder handlers
  const moveFn = (idx: number, dir: -1 | 1) => {
    setBuilderFns(prev => {
      const arr = [...prev]
      const ni = idx + dir
      if (ni < 0 || ni >= arr.length) return prev
      const [it] = arr.splice(idx, 1)
      arr.splice(ni, 0, it)
      return arr
    })
  }
  const toggleFn = (idx: number) => {
    setBuilderFns(prev => {
      const arr = [...prev]
      arr[idx] = { ...arr[idx], enabled: !arr[idx].enabled }
      return arr
    })
  }

  // Mode toggle
  const handleMode = (_: any, val: 'visual' | 'code' | null) => { if (val) setMode(val) }

  // Step content
  const StepContent = () => {
    switch (activeStep) {
      case 0: // Route
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <Select size="small" value={selectedRouteId} onChange={(e: SelectChangeEvent<string>) => setSelectedRouteId(e.target.value as string)} displayEmpty sx={{ minWidth: 220 }}>
                <MenuItem value=""><em>New route…</em></MenuItem>
                {(Array.isArray(routes.data) ? routes.data : []).map((r: Route) => <MenuItem key={r.id} value={r.id}>{r.name}</MenuItem>)}
              </Select>
              <Tooltip title="Quick New"><IconButton onClick={quickAddRoute} color="primary"><AddCircleIcon /></IconButton></Tooltip>
              <Tooltip title="Refresh"><IconButton onClick={() => routes.refresh()} color="primary"><RefreshIcon /></IconButton></Tooltip>
            </Box>
            <TextField fullWidth label="Route name" value={routeName} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRouteName(e.target.value)} />
          </Box>
        )
      case 1: // Filter
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {mode === 'visual' ? (
              <>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Chip icon={<AutoFixHighIcon />} color="info" label="IP + rest" onClick={() => setPattern('(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)\\n?(?P<rest>.*)')} />
                  <Chip color="secondary" label="Reset to IP" onClick={() => { setPattern('(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)'); }} />
                  <Chip color="success" label="Show everything" onClick={() => setPattern('true')} />
                  <Chip color="warning" label="CEF only" onClick={() => setPattern('(?P<cef>CEF:.*)')} />
                </Box>
                <Box sx={{ display: 'flex', gap: 1 }}>
                  <Select size="small" value={libSel} onChange={(e: SelectChangeEvent<string>) => { const v = e.target.value as string; setLibSel(v); loadLibFile(v) }} displayEmpty sx={{ minWidth: 220 }}>
                    <MenuItem value=""><em>Sample library…</em></MenuItem>
                    {(Array.isArray(lib) ? lib : []).map(i => <MenuItem key={i.name} value={i.name}>{i.name}</MenuItem>)}
                  </Select>
                  <IconButton onClick={loadLib}><RefreshIcon /></IconButton>
                </Box>
                <TextField label="Filter (regex or expression)" value={pattern} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPattern(e.target.value)} fullWidth />
              </>
            ) : (
              <TextField label="Filter (raw)" value={pattern} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setPattern(e.target.value)} fullWidth multiline minRows={6} />
            )}
            {!!err && <Typography variant="caption" color="error">{err}</Typography>}
            {evaluating && <LinearProgress sx={{ mt: .5 }} />}
          </Box>
        )
      case 2: // Enrichment
        return (
          <GeoIPEnrichment pattern={pattern} before={before} onSuggestIp={() => {
            // put caret to end to keep typing smooth
            beforeRef.current?.focus()
          }} />
        )
      case 3: // Pipeline
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Select fullWidth size="small" value={pipeId} onChange={(e: SelectChangeEvent<string>) => setPipeId(e.target.value as string)} displayEmpty>
              <MenuItem value=""><em>Choose pipeline</em></MenuItem>
              {(Array.isArray(pipelines.data) ? pipelines.data : []).map(p => <MenuItem key={p.id} value={p.id}>{p.name}</MenuItem>)}
            </Select>
            {!pipeId ? <Chip label="Choose a pipeline to configure functions" /> : (
              <>
                <Typography variant="subtitle2">Functions</Typography>
                {mode === 'visual' && (
                  <Stack spacing={.5}>
                    {builderFns.map((fn, idx) => (
                      <Paper variant="outlined" key={fn.name} sx={{ p: .5, display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Checkbox checked={fn.enabled} onChange={() => toggleFn(idx)} />
                        <Typography sx={{ flex: 1 }}>{fn.name}</Typography>
                        <IconButton size="small" onClick={() => moveFn(idx, -1)} disabled={idx === 0}><ArrowUpwardIcon fontSize="small" /></IconButton>
                        <IconButton size="small" onClick={() => moveFn(idx, 1)} disabled={idx === builderFns.length - 1}><ArrowDownwardIcon fontSize="small" /></IconButton>
                      </Paper>
                    ))}
                  </Stack>
                )}
                {mode === 'code' && (
                  <TextField
                    label="Functions (JSON)"
                    value={JSON.stringify(builderFns, null, 2)}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      try {
                        const v = JSON.parse(e.target.value) as FnItem[]
                        if (Array.isArray(v)) setBuilderFns(v.map(x => ({ name: String(x.name), enabled: !!x.enabled })))
                        setErr(undefined)
                      } catch (ex: any) {
                        setErr('Invalid JSON')
                      }
                    }}
                    multiline minRows={8} fullWidth
                  />
                )}
              </>
            )}
          </Box>
        )
  case 4: // Destination
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Select fullWidth size="small" value={destId} onChange={(e: SelectChangeEvent<string>) => setDestId(e.target.value as string)} displayEmpty>
              <MenuItem value=""><em>Choose destination</em></MenuItem>
              {(Array.isArray(dests.data) ? dests.data : []).map(d => <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>)}
            </Select>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: .5 }}>
              {destId ? <Chip label={Array.isArray(dests.data) ? dests.data.find(d => d.id === destId)?.type || 'selected' : 'selected'} color="primary" variant="outlined" /> : <Chip label="select a destination" />}
            </Box>
          </Box>
        )
  case 5: // Preview
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Typography variant="body2">Review matches and route summary in the live preview. You can also run an explicit test.</Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <Button variant="outlined" startIcon={<PlayArrowIcon />} onClick={run} disabled={evaluating}>Run Preview</Button>
              {evaluating && <LinearProgress sx={{ flex: 1, alignSelf: 'center' }} />}
            </Box>
            {!!err && <Typography variant="caption" color="error">{err}</Typography>}
          </Box>
        )
  case 6: // Save
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Typography variant="body2">Save your route and run a quick test. You can edit again anytime.</Typography>
            <Button variant="contained" color="primary" startIcon={<SaveIcon />} onClick={async () => { await saveRoute(); await run() }}>
              Save & Test
            </Button>
          </Box>
        )
      default:
        return null
    }
  }

  // Layout with persistent live preview
  return (
    <Paper elevation={6} sx={{ p: 2, borderRadius: 3 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="h6" sx={{ fontWeight: 800 }}>Transform Workbench</Typography>
          <Chip size="small" color="secondary" label="Wizard" />
        </Box>
        <ToggleButtonGroup size="small" value={mode} exclusive onChange={handleMode}>
          <ToggleButton value="visual"><VisibilityIcon fontSize="small" />&nbsp;Visual</ToggleButton>
          <ToggleButton value="code"><CodeIcon fontSize="small" />&nbsp;Code</ToggleButton>
        </ToggleButtonGroup>
      </Box>

      <Stepper activeStep={activeStep} alternativeLabel sx={{ mb: 2 }}>
        {steps.map((label, i) => (
          <Step key={label} onClick={() => { if (i <= activeStep || gates.slice(0, i).every(Boolean)) setActiveStep(i) }} sx={{ cursor: 'pointer' }}>
            <StepLabel>{label}</StepLabel>
          </Step>
        ))}
      </Stepper>

      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
        <Box>
          <StepContent />
          <Divider sx={{ my: 2 }} />
          <Box sx={{ display: 'flex', gap: 1, justifyContent: 'space-between' }}>
            <Button variant="outlined" onClick={() => setActiveStep(s => Math.max(0, s - 1))} disabled={activeStep === 0}>Back</Button>
            <Button variant="contained" onClick={() => setActiveStep(s => Math.min(steps.length - 1, s + 1))} disabled={!canNext || activeStep === steps.length - 1}>Next</Button>
          </Box>
        </Box>

        <Box>
          <Paper variant="outlined" sx={{ p: 1.5 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: .5 }}>
              <FilterAltIcon fontSize="small" /><Typography variant="subtitle2">Live Preview</Typography>
            </Box>
            {evaluating && <LinearProgress sx={{ mb: 1 }} />}
            <Typography variant="caption" sx={{ opacity: .9 }}>Sample input</Typography>
            <TextField
              inputRef={beforeRef}
              value={before}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setBefore(e.target.value)}
              multiline minRows={8} fullWidth
              sx={{ '& textarea': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' }, mb: 1 }}
            />
            <Typography variant="caption" sx={{ opacity: .9 }}>Structured matches</Typography>
            <TextField
              value={after}
              multiline minRows={8} fullWidth
              InputProps={{ readOnly: true }}
              sx={{ '& textarea': { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' }, mb: 1 }}
            />
            <Divider sx={{ my: 1 }} />
            <Typography variant="caption" sx={{ opacity: .9 }}>Route summary</Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: .5, mt: .5 }}>
              <Chip icon={<AltRouteIcon />} label={routeName || 'unnamed'} />
              <Chip label={pattern ? `filter: ${pattern.slice(0, 24)}${pattern.length > 24 ? '…' : ''}` : 'no filter'} color="info" variant="outlined" />
              <Chip icon={<RocketLaunchIcon />} label={Array.isArray(pipelines.data) ? pipelines.data.find(p => p.id === pipeId)?.name || 'no pipeline' : 'no pipeline'} color="success" variant="outlined" />
              <Chip label={Array.isArray(dests.data) ? dests.data.find(d => d.id === destId)?.name || 'no destination' : 'no destination'} color="primary" variant="outlined" />
            </Box>
          </Paper>
        </Box>
      </Box>

      <Snackbar open={saveOk} autoHideDuration={2500} onClose={() => setSaveOk(false)}>
        <Alert onClose={() => setSaveOk(false)} severity="success" sx={{ width: '100%' }}>
          Route saved and preview updated.
        </Alert>
      </Snackbar>
    </Paper>
  )
}

// Lightweight enrichment step focusing on GeoIP
function GeoIPEnrichment({ pattern, before, onSuggestIp }: { pattern: string; before: string; onSuggestIp?: () => void }) {
  const [status, setStatus] = useState<{loaded:boolean; path?:string; size?:number; mtime?:number}>({loaded:false})
  const [ip, setIp] = useState('')
  const [geo, setGeo] = useState<any>(null)
  const [busy, setBusy] = useState(false)
  const reload = async () => {
    try { const r = await fetch('/api/v1/enrich/geoip/status'); setStatus(await r.json()) } catch {}
  }
  useEffect(() => { reload() }, [])
  const upload = async (file: File) => {
    const fd = new FormData(); fd.append('file', file)
    await fetch('/api/v1/enrich/geoip/upload', { method: 'POST', body: fd })
    await reload()
  }
  const preview = async () => {
    setBusy(true)
    setGeo(null)
    try {
      const body: any = { sample: before, pattern }
      if (ip) body.ip = ip
      const r = await fetch('/api/v1/preview/enrich', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
      const j = await r.json(); setGeo(j)
    } finally { setBusy(false) }
  }
  const suggest = () => {
    // naive IP pick from sample
    const m = before.match(/\b\d+\.\d+\.\d+\.\d+\b/); if (m) setIp(m[0]); onSuggestIp?.()
  }
  return (
    <Box sx={{ display:'flex', flexDirection:'column', gap:1 }}>
      <Typography variant="subtitle2">GeoIP</Typography>
      <Box sx={{ display:'flex', alignItems:'center', gap:1, flexWrap:'wrap' }}>
        <Chip label={status.loaded ? 'Database loaded' : 'No database'} color={status.loaded ? 'success' : 'warning'} />
        {!!status.path && <Chip label={(status.path.split(/[\\/]/).pop() || 'db')} />}
        <Button variant="outlined" size="small" startIcon={<RefreshIcon />} onClick={reload}>Status</Button>
        <Button variant="contained" component="label" size="small">Upload .mmdb<input hidden type="file" accept=".mmdb" onChange={e => { const f = e.target.files?.[0]; if (f) upload(f) }} /></Button>
      </Box>
      <Box sx={{ display:'flex', gap:1, alignItems:'center' }}>
        <TextField size="small" label="IP (optional)" value={ip} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setIp(e.target.value)} sx={{ maxWidth: 260 }} />
        <Button size="small" onClick={suggest}>Suggest from sample</Button>
        <Button variant="contained" size="small" onClick={preview} disabled={!status.loaded || busy}>{busy? 'Looking…':'Preview enrichment'}</Button>
      </Box>
      {geo && (
        <Paper variant="outlined" sx={{ p:1 }}>
          <Typography variant="caption">Preview</Typography>
          <pre className="code-preview">{JSON.stringify(geo, null, 2)}</pre>
        </Paper>
      )}
    </Box>
  )
}
