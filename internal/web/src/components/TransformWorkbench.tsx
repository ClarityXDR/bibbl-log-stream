import React, { useEffect, useMemo, useState } from 'react'
import './styles.css'
import {
  Box,
  Paper,
  Typography,
  TextField,
  Select,
  MenuItem,
  Button,
  IconButton,
  Tooltip,
  Snackbar,
  Alert,
  Stack,
  LinearProgress,
  Switch,
  FormControlLabel,
  Chip,
  Divider
} from '@mui/material'
import { SelectChangeEvent } from '@mui/material'
import { apiClient } from '../utils/apiClient'
import {
  ArrowBack as BackIcon,
  Save as SaveIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material'
import TemplateGallery from './transform/TemplateGallery'
import BeforeAfterPreview, { PreviewResult } from './transform/BeforeAfterPreview'
import FieldExtractionUI from './transform/FieldExtractionUI'
import FriendlyPipelineBuilder from './transform/FriendlyPipelineBuilder'
import { TransformTemplate } from './transform/templates'

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



export default function TransformWorkbench({filtersInitialSelected}: {filtersInitialSelected?: string}) {
  // Wizard state - simplified to 3 main steps
  const [currentView, setCurrentView] = useState<'gallery' | 'builder'>('gallery')
  const [selectedTemplate, setSelectedTemplate] = useState<TransformTemplate | null>(null)
  const [testMode, setTestMode] = useState<boolean>(true)
  
  // Data
  const routes = useFetcher<Route[]>('/api/v1/routes', 8000)
  const pipelines = useFetcher<Pipeline[]>('/api/v1/pipelines', 10000)
  const dests = useFetcher<Destination[]>('/api/v1/destinations', 10000)

  // Form state
  const [routeName, setRouteName] = useState('New Route')
  const [pattern, setPattern] = useState('true')
  const [sampleLog, setSampleLog] = useState<string>('Paste your sample log here...')
  const [pipeId, setPipeId] = useState<string>('')
  const [destId, setDestId] = useState<string>('')
  const [previewResult, setPreviewResult] = useState<PreviewResult | null>(null)
  const [enableEnrichment, setEnableEnrichment] = useState(false)

  // Save state
  const [saveOk, setSaveOk] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  // Undo/Redo support
  const [history, setHistory] = useState<{ pattern: string; sampleLog: string }[]>([])
  const [historyIndex, setHistoryIndex] = useState(-1)

  // Handle template selection
  const handleSelectTemplate = (template: TransformTemplate) => {
    setSelectedTemplate(template)
    setCurrentView('builder')
    
    // Pre-populate form from template
    setRouteName(template.config.routeName)
    setPattern(template.config.filterPattern)
    setSampleLog(template.sampleLog)
    setEnableEnrichment(template.config.enrichment?.enabled || false)
    
    // Try to match suggested pipeline and destination
    if (template.config.pipelineSuggestion && Array.isArray(pipelines.data)) {
      const suggestedPipe = pipelines.data.find((p: Pipeline) => 
        p.name.toLowerCase().includes(template.config.pipelineSuggestion!.toLowerCase())
      )
      if (suggestedPipe) setPipeId(suggestedPipe.id)
    }
    
    if (template.config.destinationSuggestion && Array.isArray(dests.data)) {
      const suggestedDest = dests.data.find((d: Destination) =>
        d.type.toLowerCase().includes(template.config.destinationSuggestion!.toLowerCase())
      )
      if (suggestedDest) setDestId(suggestedDest.id)
    }
  }

  // Add to history when pattern or sample changes
  const addToHistory = () => {
    setHistory(prev => [...prev.slice(0, historyIndex + 1), { pattern, sampleLog }])
    setHistoryIndex(prev => prev + 1)
  }

  const undo = () => {
    if (historyIndex > 0) {
      const prev = history[historyIndex - 1]
      setPattern(prev.pattern)
      setSampleLog(prev.sampleLog)
      setHistoryIndex(historyIndex - 1)
    }
  }

  const redo = () => {
    if (historyIndex < history.length - 1) {
      const next = history[historyIndex + 1]
      setPattern(next.pattern)
      setSampleLog(next.sampleLog)
      setHistoryIndex(historyIndex + 1)
    }
  }

  // Save route
  const saveRoute = async () => {
    try {
      setSaveError(null)
      
      // Validation
      if (!routeName.trim()) {
        setSaveError('Route name is required')
        return
      }
      if (!pipeId) {
        setSaveError('Please select a pipeline')
        return
      }
      if (!destId) {
        setSaveError('Please select a destination')
        return
      }

      const body = { 
        name: routeName.trim(), 
        filter: pattern, 
        pipelineID: pipeId, 
        destination: destId, 
        final: true 
      }
      
      await apiClient.post('/api/v1/routes', body)
      await routes.refresh()
      setSaveOk(true)
      
      // Reset to gallery after successful save
      setTimeout(() => {
        setCurrentView('gallery')
        resetForm()
      }, 2000)
    } catch (err: any) {
      setSaveError(err?.message || 'Failed to save route')
    }
  }

  // Reset form
  const resetForm = () => {
    setSelectedTemplate(null)
    setRouteName('New Route')
    setPattern('true')
    setSampleLog('Paste your sample log here...')
    setPipeId('')
    setDestId('')
    setPreviewResult(null)
    setEnableEnrichment(false)
    setHistory([])
    setHistoryIndex(-1)
  }

  // Back to gallery
  const handleBack = () => {
    if (confirm('Go back to template selection? Your current work will be lost.')) {
      setCurrentView('gallery')
      resetForm()
    }
  }

  // Progress calculation
  const getProgress = () => {
    let completed = 0
    const total = 4
    if (pattern.trim().length > 0) completed++
    if (previewResult && previewResult.matchedLines > 0) completed++
    if (pipeId) completed++
    if (destId) completed++
    return (completed / total) * 100
  }

  // Validation helpers
  const isFormValid = () => {
    return routeName.trim().length > 0 && !!pipeId && !!destId
  }

  const getValidationMessage = () => {
    if (!routeName.trim()) return 'Enter a name for your route'
    if (!pipeId) return 'Select a pipeline to process logs'
    if (!destId) return 'Select where to send processed logs'
    if (previewResult && previewResult.matchedLines === 0) return 'Your pattern doesn\'t match any sample logs'
    return null
  }

  // Render template gallery
  if (currentView === 'gallery') {
    return (
      <Paper elevation={6} sx={{ borderRadius: 3, overflow: 'hidden' }}>
        <TemplateGallery onSelectTemplate={handleSelectTemplate} />
      </Paper>
    )
  }

  // Render builder view
  return (
    <Paper elevation={6} sx={{ borderRadius: 3, overflow: 'hidden' }}>
      {/* Header */}
      <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider', bgcolor: 'primary.50' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Box sx={{ flex: 1 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 0.5 }}>
              <Typography variant="h5" sx={{ fontWeight: 700 }}>
                {selectedTemplate?.icon} {selectedTemplate?.title || 'Custom Transform'}
              </Typography>
              {testMode && (
                <Chip 
                  label="🧪 Test Mode" 
                  size="small" 
                  color="warning"
                  sx={{ fontWeight: 600 }}
                />
              )}
            </Box>
            <Typography variant="body2" color="text.secondary">
              {selectedTemplate?.description || 'Configure your custom log transformation'}
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
            <Tooltip title="Back to templates">
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
              {Math.round(getProgress()) === 100 ? '🎉 Ready to save!' : 'Keep going...'}
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

        {/* Basic Info */}
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'flex-end' }}>
          <TextField
            label="Route Name"
            value={routeName}
            onChange={(e) => setRouteName(e.target.value)}
            required
            sx={{ flex: 1 }}
            helperText="Give your transform a descriptive name"
          />
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => {
              pipelines.refresh()
              dests.refresh()
            }}
          >
            Refresh
          </Button>
        </Box>
      </Box>

      {/* Main Content */}
      <Box sx={{ p: 3 }}>
        <Stack spacing={4}>
          {/* Step 1: Extract Fields */}
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
              Step 1: What fields do you want to extract?
            </Typography>
            <FieldExtractionUI
              sampleLog={sampleLog}
              pattern={pattern}
              onPatternChange={setPattern}
            />
          </Box>

          <Divider />

          {/* Step 2: Preview */}
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
              Step 2: Preview your results
            </Typography>
            <BeforeAfterPreview
              sampleInput={sampleLog}
              onSampleChange={setSampleLog}
              pattern={pattern}
              enrichmentEnabled={enableEnrichment}
              onPreviewResult={setPreviewResult}
            />
          </Box>

          <Divider />

          {/* Step 3: Choose Pipeline */}
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
              Step 3: How should we process the logs?
            </Typography>
            <Select
              fullWidth
              value={pipeId}
              onChange={(e) => setPipeId(e.target.value)}
              displayEmpty
              sx={{ mb: 2 }}
            >
              <MenuItem value="">
                <em>Select a pipeline...</em>
              </MenuItem>
              {(Array.isArray(pipelines.data) ? pipelines.data : []).map((p: Pipeline) => (
                <MenuItem key={p.id} value={p.id}>
                  {p.name}
                  {p.description && ` - ${p.description}`}
                </MenuItem>
              ))}
            </Select>

            {pipeId && (
              <FriendlyPipelineBuilder
                selectedPipelineId={pipeId}
                availableFunctions={
                  Array.isArray(pipelines.data)
                    ? pipelines.data.find((p: Pipeline) => p.id === pipeId)?.functions || []
                    : []
                }
              />
            )}
          </Box>

          <Divider />

          {/* Step 4: Choose Destination */}
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>
              Step 4: Where should we send the processed logs?
            </Typography>
            <Select
              fullWidth
              value={destId}
              onChange={(e) => setDestId(e.target.value)}
              displayEmpty
            >
              <MenuItem value="">
                <em>Select a destination...</em>
              </MenuItem>
              {(Array.isArray(dests.data) ? dests.data : []).map((d: Destination) => (
                <MenuItem key={d.id} value={d.id}>
                  {d.name} ({d.type})
                </MenuItem>
              ))}
            </Select>
          </Box>

          {/* Validation Message */}
          {getValidationMessage() && (
            <Alert severity={isFormValid() ? 'info' : 'warning'}>
              {getValidationMessage()}
            </Alert>
          )}

          {/* Save Error */}
          {saveError && (
            <Alert severity="error" onClose={() => setSaveError(null)}>
              {saveError}
            </Alert>
          )}

          {/* Actions */}
          <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
            <Button variant="outlined" onClick={handleBack}>
              Cancel
            </Button>
            <Button
              variant="contained"
              size="large"
              startIcon={<SaveIcon />}
              onClick={() => {
                if (testMode) {
                  if (confirm('🧪 You are in Test Mode. Switch to Live Mode to save this transform for real?')) {
                    setTestMode(false)
                  }
                } else {
                  saveRoute()
                }
              }}
              disabled={!isFormValid()}
              sx={{
                py: 1.5,
                px: 4,
                fontSize: '1.1rem',
                fontWeight: 700,
                background: isFormValid() 
                  ? 'linear-gradient(45deg, #4CAF50 30%, #8BC34A 90%)'
                  : undefined,
                boxShadow: isFormValid() ? '0 3px 5px 2px rgba(76, 175, 80, .3)' : undefined,
                '&:hover': isFormValid() ? {
                  background: 'linear-gradient(45deg, #45A049 30%, #7CB342 90%)',
                  transform: 'scale(1.05)',
                  boxShadow: '0 6px 10px 4px rgba(76, 175, 80, .4)'
                } : undefined,
                '&:disabled': {
                  background: 'grey.300'
                }
              }}
            >
              {testMode ? '🧪 Test Save' : '🎉 Save Transform!'}
            </Button>
          </Box>
        </Stack>
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
          🎉 {testMode ? 'Test completed! Changes not saved.' : 'Success! Transform saved and ready to use!'}
        </Alert>
      </Snackbar>
    </Paper>
  )
}
