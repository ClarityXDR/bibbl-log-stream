import React, { useEffect, useState } from 'react'
import {
  Box,
  Paper,
  Typography,
  TextField,
  Chip,
  LinearProgress,
  Stack,
  Alert,
  Tooltip,
  IconButton,
  Collapse
} from '@mui/material'
import {
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
  Info as InfoIcon,
  Refresh as RefreshIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon
} from '@mui/icons-material'

type BeforeAfterPreviewProps = {
  sampleInput: string
  onSampleChange: (value: string) => void
  pattern: string
  enrichmentEnabled?: boolean
  onPreviewResult?: (result: PreviewResult) => void
  autoRun?: boolean
}

export type PreviewResult = {
  matchedLines: number
  totalLines: number
  extractedFields: Record<string, any>[]
  errors: string[]
}

export default function BeforeAfterPreview({
  sampleInput,
  onSampleChange,
  pattern,
  enrichmentEnabled = false,
  onPreviewResult,
  autoRun = true
}: BeforeAfterPreviewProps) {
  const [output, setOutput] = useState<string>('')
  const [result, setResult] = useState<PreviewResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showRawOutput, setShowRawOutput] = useState(false)

  // Debounced preview execution
  useEffect(() => {
    if (!autoRun) return
    
    const timer = setTimeout(() => {
      runPreview()
    }, 400)

    return () => clearTimeout(timer)
  }, [sampleInput, pattern, enrichmentEnabled])

  const runPreview = async () => {
    setLoading(true)
    setError(null)
    
    try {
      const lines = sampleInput.split(/\r?\n/).filter(Boolean).slice(0, 20)
      const outputs: any[] = []
      const errors: string[] = []
      let matchCount = 0

      // Check if GeoIP is available
      let geoLoaded = false
      if (enrichmentEnabled) {
        try {
          const statusRes = await fetch('/api/v1/enrich/geoip/status')
          const statusData = await statusRes.json()
          geoLoaded = statusData?.loaded || false
        } catch {
          // Ignore status check errors
        }
      }

      // Process each line
      for (const line of lines) {
        try {
          const res = await fetch('/api/v1/preview/regex', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ sample: line, pattern })
          })
          
          if (!res.ok) {
            errors.push(`Line preview failed: ${res.statusText}`)
            outputs.push({})
            continue
          }

          const data = await res.json()
          let result: any = data.captures || {}

          // Check if we matched anything
          if (Object.keys(result).length > 0) {
            matchCount++
          }

          // Add enrichment if enabled and available
          if (enrichmentEnabled && geoLoaded && Object.keys(result).length > 0) {
            try {
              const enrichRes = await fetch('/api/v1/preview/enrich', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sample: line, pattern })
              })
              
              if (enrichRes.ok) {
                const enrichData = await enrichRes.json()
                if (enrichData?.geo) {
                  result.geo = enrichData.geo
                }
              }
            } catch {
              // Enrichment is optional, continue without it
            }
          }

          outputs.push(result)
        } catch (err: any) {
          errors.push(err?.message || 'Unknown error')
          outputs.push({})
        }
      }

      // Build result
      const previewResult: PreviewResult = {
        matchedLines: matchCount,
        totalLines: lines.length,
        extractedFields: outputs,
        errors
      }

      setResult(previewResult)
      setOutput(outputs.map(o => JSON.stringify(o, null, 2)).join('\n\n'))
      
      if (onPreviewResult) {
        onPreviewResult(previewResult)
      }
    } catch (err: any) {
      setError(err?.message || 'Preview failed')
    } finally {
      setLoading(false)
    }
  }

  // Generate field summary from first matched result
  const getFieldSummary = () => {
    if (!result || result.extractedFields.length === 0) return []
    
    const firstMatch = result.extractedFields.find(f => Object.keys(f).length > 0)
    if (!firstMatch) return []
    
    return Object.keys(firstMatch)
  }

  const matchRate = result ? (result.matchedLines / result.totalLines) * 100 : 0
  const fields = getFieldSummary()

  return (
    <Box>
      <Stack spacing={2}>
        {/* Status Bar */}
        {loading && <LinearProgress />}
        
        {result && !loading && (
          <Alert 
            severity={matchRate === 100 ? 'success' : matchRate > 0 ? 'info' : 'warning'}
            icon={matchRate === 100 ? <CheckIcon /> : matchRate > 0 ? <InfoIcon /> : <ErrorIcon />}
            action={
              <Tooltip title="Refresh preview">
                <IconButton size="small" onClick={runPreview}>
                  <RefreshIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            }
            sx={{
              animation: matchRate === 100 ? 'celebrate 0.5s ease-in-out' : 'none',
              '@keyframes celebrate': {
                '0%, 100%': { transform: 'scale(1)' },
                '25%': { transform: 'scale(1.02) rotate(-1deg)' },
                '75%': { transform: 'scale(1.02) rotate(1deg)' }
              }
            }}
          >
            <Box>
              <Typography variant="body2" sx={{ fontWeight: 600 }}>
                {matchRate === 100 && '🎉 Perfect match! All logs will be processed'}
                {matchRate > 0 && matchRate < 100 && `✓ Matched ${result.matchedLines} of ${result.totalLines} sample logs`}
                {matchRate === 0 && '⚠ No matches found - check your pattern'}
              </Typography>
              
              {matchRate === 0 && (
                <Typography variant="caption" sx={{ display: 'block', mt: 1 }}>
                  💡 <strong>Suggestions:</strong> Try using the "Accept All JSON Fields" button for JSON logs, 
                  or use the "Choose Common Fields" picker to select fields from your log text.
                </Typography>
              )}
              
              {fields.length > 0 && (
                <Box sx={{ mt: 1 }}>
                  <Typography variant="caption" sx={{ fontWeight: 600, mr: 1 }}>
                    Extracted {fields.length} field{fields.length !== 1 ? 's' : ''}:
                  </Typography>
                  <Box sx={{ display: 'inline-flex', gap: 0.5, flexWrap: 'wrap' }}>
                    {fields.map((field) => (
                      <Chip
                        key={field}
                        label={field}
                        size="small"
                        color="primary"
                        variant="outlined"
                        sx={{ fontSize: '0.7rem', height: 20 }}
                      />
                    ))}
                  </Box>
                </Box>
              )}
            </Box>
          </Alert>
        )}

        {error && (
          <Alert severity="error">
            <Typography variant="body2">{error}</Typography>
          </Alert>
        )}

        {result && result.errors.length > 0 && (
          <Alert severity="warning">
            <Typography variant="body2" sx={{ fontWeight: 600 }}>
              Some logs had errors:
            </Typography>
            {result.errors.slice(0, 3).map((err, idx) => (
              <Typography key={idx} variant="caption" sx={{ display: 'block' }}>
                • {err}
              </Typography>
            ))}
            {result.errors.length > 3 && (
              <Typography variant="caption">
                ... and {result.errors.length - 3} more errors
              </Typography>
            )}
          </Alert>
        )}

        {/* Split View */}
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
          {/* Before (Input) */}
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                📝 Sample Logs (Before)
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <Tooltip title="Load example logs to test with">
                  <IconButton 
                    size="small" 
                    onClick={() => {
                      const samples = [
                        '2024-11-18 14:23:45 [INFO] User login successful from 203.0.113.50',
                        '2024-11-18 14:24:12 [ERROR] Connection timeout to 198.51.100.25',
                        '2024-11-18 14:25:33 [WARN] High CPU usage detected: 87%',
                        '2024-11-18 14:26:01 [INFO] Database backup completed in 45.3s'
                      ]
                      onSampleChange(samples.join('\n'))
                    }}
                    sx={{ 
                      bgcolor: 'primary.50',
                      '&:hover': { bgcolor: 'primary.100' }
                    }}
                  >
                    <RefreshIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
                <Chip label={`${sampleInput.split(/\r?\n/).filter(Boolean).length} lines`} size="small" />
              </Box>
            </Box>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              Paste your log samples here, or click <RefreshIcon sx={{ fontSize: 12, verticalAlign: 'middle' }} /> to load examples
            </Typography>
            <TextField
              value={sampleInput}
              onChange={(e) => onSampleChange(e.target.value)}
              multiline
              minRows={12}
              maxRows={20}
              fullWidth
              placeholder="Paste your sample logs here...&#10;Each line will be processed separately.&#10;&#10;Example:&#10;2023-11-17 14:23:45 [INFO] User login from 203.0.113.50&#10;2023-11-17 14:24:12 [ERROR] Connection timeout to 198.51.100.25"
              sx={{
                '& textarea': {
                  fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                  fontSize: '0.85rem',
                  lineHeight: 1.5
                }
              }}
            />
          </Paper>

          {/* After (Output) */}
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                ✨ Extracted Data (After)
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                {result && (
                  <Chip 
                    label={`${result.matchedLines} matched`} 
                    size="small"
                    color={matchRate === 100 ? 'success' : 'default'}
                  />
                )}
                <Tooltip title={showRawOutput ? 'Hide raw JSON' : 'Show raw JSON'}>
                  <IconButton size="small" onClick={() => setShowRawOutput(!showRawOutput)}>
                    {showRawOutput ? <CollapseIcon fontSize="small" /> : <ExpandIcon fontSize="small" />}
                  </IconButton>
                </Tooltip>
              </Box>
            </Box>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
              {fields.length > 0 
                ? 'Fields extracted from your logs' 
                : 'No data extracted yet - adjust your pattern'}
            </Typography>
            
            {/* Human-readable summary */}
            {!showRawOutput && result && result.matchedLines > 0 && (
              <Box sx={{ 
                p: 1.5, 
                bgcolor: 'background.paper', 
                borderRadius: 1,
                maxHeight: 400,
                overflowY: 'auto'
              }}>
                {result.extractedFields
                  .filter(f => Object.keys(f).length > 0)
                  .slice(0, 10)
                  .map((fields, idx) => (
                    <Paper 
                      key={idx} 
                      variant="outlined" 
                      sx={{ 
                        p: 1.5, 
                        mb: 1,
                        '&:last-child': { mb: 0 }
                      }}
                    >
                      <Typography variant="caption" sx={{ fontWeight: 600, color: 'primary.main', display: 'block', mb: 0.5 }}>
                        Log {idx + 1}:
                      </Typography>
                      <Stack spacing={0.5}>
                        {Object.entries(fields).map(([key, value]) => (
                          <Box key={key} sx={{ display: 'flex', gap: 1 }}>
                            <Typography 
                              variant="caption" 
                              sx={{ 
                                fontWeight: 600, 
                                minWidth: 100,
                                color: 'text.secondary'
                              }}
                            >
                              {key}:
                            </Typography>
                            <Typography 
                              variant="caption" 
                              sx={{ 
                                fontFamily: 'monospace',
                                wordBreak: 'break-all'
                              }}
                            >
                              {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                            </Typography>
                          </Box>
                        ))}
                      </Stack>
                    </Paper>
                  ))}
                {result.extractedFields.filter(f => Object.keys(f).length > 0).length > 10 && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1, textAlign: 'center' }}>
                    ... and {result.extractedFields.filter(f => Object.keys(f).length > 0).length - 10} more logs
                  </Typography>
                )}
              </Box>
            )}

            {/* Raw JSON output */}
            <Collapse in={showRawOutput}>
              <TextField
                value={output}
                multiline
                minRows={12}
                maxRows={20}
                fullWidth
                InputProps={{ readOnly: true }}
                placeholder={loading ? 'Processing...' : 'Output will appear here'}
                sx={{
                  '& textarea': {
                    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
                    fontSize: '0.85rem',
                    lineHeight: 1.5,
                    bgcolor: 'grey.100'
                  }
                }}
              />
            </Collapse>

            {/* Empty state */}
            {!showRawOutput && (!result || result.matchedLines === 0) && !loading && (
              <Box sx={{ 
                p: 4, 
                textAlign: 'center',
                bgcolor: 'background.paper',
                borderRadius: 1,
                minHeight: 200,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center'
              }}>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  No fields extracted yet
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {sampleInput.trim() === '' 
                    ? 'Add sample logs to see extracted data' 
                    : 'Your pattern didn\'t match any logs - try adjusting it'}
                </Typography>
              </Box>
            )}
          </Paper>
        </Box>
      </Stack>
    </Box>
  )
}
