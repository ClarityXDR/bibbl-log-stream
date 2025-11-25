import React, { useState, useRef } from 'react'
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Chip,
  Stack,
  Alert,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon
} from '@mui/material'
import {
  TipsAndUpdates as TipIcon,
  Close as CloseIcon,
  Check as CheckIcon,
  LocationOn as IpIcon,
  AccessTime as TimeIcon,
  Person as UserIcon,
  Email as EmailIcon,
  Phone as PhoneIcon,
  DataObject as JsonIcon
} from '@mui/icons-material'

type FieldExtractionUIProps = {
  sampleLog: string
  pattern: string
  onPatternChange: (pattern: string) => void
  onFieldsExtracted?: (fields: string[]) => void
}

type FieldPreset = {
  id: string
  name: string
  description: string
  icon: React.ReactNode
  pattern: string
  example: string
}

const fieldPresets: FieldPreset[] = [
  {
    id: 'ip',
    name: 'IP Address',
    description: 'IPv4 address (e.g., 192.168.1.1)',
    icon: <IpIcon />,
    pattern: '(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)',
    example: '192.168.1.1 or 10.0.0.50'
  },
  {
    id: 'timestamp',
    name: 'Timestamp',
    description: 'ISO 8601 or common log timestamp',
    icon: <TimeIcon />,
    pattern: '(?P<timestamp>\\d{4}-\\d{2}-\\d{2}[T ]\\d{2}:\\d{2}:\\d{2})',
    example: '2023-11-17 14:23:45 or 2023-11-17T14:23:45'
  },
  {
    id: 'email',
    name: 'Email Address',
    description: 'Email address',
    icon: <EmailIcon />,
    pattern: '(?P<email>[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,})',
    example: 'user@example.com'
  },
  {
    id: 'username',
    name: 'Username',
    description: 'Username or account name',
    icon: <UserIcon />,
    pattern: '(?P<username>[a-zA-Z0-9._-]+)',
    example: 'john_smith or user123'
  },
  {
    id: 'phone',
    name: 'Phone Number',
    description: 'US phone number',
    icon: <PhoneIcon />,
    pattern: '(?P<phone>\\d{3}[-.\\s]?\\d{3}[-.\\s]?\\d{4})',
    example: '555-123-4567 or 5551234567'
  },
  {
    id: 'json',
    name: 'JSON Object',
    description: 'Match entire JSON log',
    icon: <JsonIcon />,
    pattern: 'true',
    example: 'Use JavaScript filter for JSON parsing'
  }
]

export default function FieldExtractionUI({
  sampleLog,
  pattern,
  onPatternChange,
  onFieldsExtracted
}: FieldExtractionUIProps) {
  const [showPresets, setShowPresets] = useState(false)
  const [showHelp, setShowHelp] = useState(false)
  const [customFieldName, setCustomFieldName] = useState('')
  const [selectedText, setSelectedText] = useState('')
  const textAreaRef = useRef<HTMLTextAreaElement>(null)

  // Handle text selection in sample log
  const handleTextSelection = () => {
    const textarea = textAreaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    
    if (start !== end) {
      const selected = sampleLog.substring(start, end)
      setSelectedText(selected)
    } else {
      setSelectedText('')
    }
  }

  // Create pattern from selected text
  const createPatternFromSelection = () => {
    if (!selectedText || !customFieldName) return

    // Escape special regex characters
    const escaped = selectedText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    
    // Build named capture group
    const newPattern = `(?P<${customFieldName}>${escaped})`
    
    // If there's already a pattern and it's not just 'true', combine them
    if (pattern && pattern !== 'true') {
      // Simple append for now - in production, this would be smarter
      onPatternChange(`${pattern}.*${newPattern}`)
    } else {
      onPatternChange(newPattern)
    }
    
    setCustomFieldName('')
    setSelectedText('')
  }

  // Apply a preset pattern
  const applyPreset = (preset: FieldPreset) => {
    if (preset.id === 'json') {
      // For JSON, use the JavaScript expression
      onPatternChange('true')
    } else {
      // For regex presets, combine with existing pattern if needed
      if (pattern && pattern !== 'true') {
        onPatternChange(`${pattern}.*${preset.pattern}`)
      } else {
        onPatternChange(preset.pattern)
      }
    }
    setShowPresets(false)
  }

  // Simplify mode - use "true" for JSON logs
  const useSimpleMode = () => {
    onPatternChange('true')
  }

  const isJsonLog = sampleLog.trim().startsWith('{')

  return (
    <Box>
      <Stack spacing={2}>
        {/* Help Banner */}
        <Alert 
          severity="info" 
          icon={<TipIcon />}
          onClose={() => setShowHelp(false)}
        >
          <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.5 }}>
            How to extract fields from your logs:
          </Typography>
          <Typography variant="caption" component="div">
            • For JSON logs: Click "Accept All JSON Fields" below<br />
            • For text logs: Click "Choose Common Fields" to add IP, timestamp, etc.<br />
            • Or highlight text in your sample and give it a name
          </Typography>
        </Alert>

        {/* Quick Actions */}
        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {isJsonLog && (
            <Tooltip title="Parse all fields from JSON logs automatically">
              <Button
                variant="contained"
                size="large"
                fullWidth={isJsonLog}
                startIcon={<CheckIcon />}
                onClick={useSimpleMode}
                sx={{ 
                  py: 1.5,
                  fontSize: '1.1rem',
                  fontWeight: 700,
                  background: 'linear-gradient(45deg, #2196F3 30%, #21CBF3 90%)',
                  boxShadow: '0 3px 5px 2px rgba(33, 203, 243, .3)',
                  animation: 'pulse 2s ease-in-out infinite',
                  '@keyframes pulse': {
                    '0%, 100%': {
                      boxShadow: '0 3px 5px 2px rgba(33, 203, 243, .3)'
                    },
                    '50%': {
                      boxShadow: '0 6px 10px 4px rgba(33, 203, 243, .5)'
                    }
                  },
                  '&:hover': {
                    background: 'linear-gradient(45deg, #1976D2 30%, #0DACCC 90%)',
                    transform: 'scale(1.02)'
                  }
                }}
              >
                ⚡ Accept All JSON Fields ⚡
              </Button>
            </Tooltip>
          )}
          
          <Button
            variant="outlined"
            onClick={() => setShowPresets(true)}
          >
            Choose Common Fields
          </Button>

          <Tooltip title="Clear pattern and start over">
            <Button
              variant="outlined"
              color="error"
              onClick={() => onPatternChange('true')}
            >
              Reset
            </Button>
          </Tooltip>
        </Box>

        {/* Current Pattern Display */}
        {pattern && pattern !== 'true' && (
          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'primary.50' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                Current Pattern
              </Typography>
              <Tooltip title="Edit pattern manually">
                <Button size="small" onClick={() => setShowHelp(!showHelp)}>
                  {showHelp ? 'Hide' : 'Edit'}
                </Button>
              </Tooltip>
            </Box>
            
            {!showHelp ? (
              <Box>
                <Typography 
                  variant="caption" 
                  sx={{ 
                    fontFamily: 'monospace',
                    wordBreak: 'break-all',
                    display: 'block',
                    p: 1,
                    bgcolor: 'background.paper',
                    borderRadius: 1
                  }}
                >
                  {pattern}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ mt: 1, display: 'block' }}>
                  This pattern will extract fields from your logs. Preview the results on the right.
                </Typography>
              </Box>
            ) : (
              <TextField
                fullWidth
                multiline
                minRows={3}
                value={pattern}
                onChange={(e) => onPatternChange(e.target.value)}
                label="Regex Pattern (Advanced)"
                helperText="Use named capture groups like (?P<fieldname>pattern)"
                sx={{ '& textarea': { fontFamily: 'monospace' } }}
              />
            )}
          </Paper>
        )}

        {/* Visual Field Picker */}
        {pattern === 'true' && !isJsonLog && (
          <Paper variant="outlined" sx={{ p: 2 }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
              ✏️ Visual Field Picker
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 2 }}>
              Highlight text in your sample log, then name it to create a field
            </Typography>

            {/* Sample log with selection */}
            <Box sx={{ mb: 2 }}>
              <Typography variant="caption" sx={{ fontWeight: 600, display: 'block', mb: 0.5 }}>
                Your sample log:
              </Typography>
              <TextField
                inputRef={textAreaRef}
                value={sampleLog}
                multiline
                fullWidth
                minRows={3}
                onMouseUp={handleTextSelection}
                onKeyUp={handleTextSelection}
                InputProps={{ readOnly: true }}
                sx={{
                  '& textarea': {
                    fontFamily: 'monospace',
                    fontSize: '0.9rem',
                    cursor: 'text',
                    userSelect: 'text'
                  }
                }}
              />
            </Box>

            {/* Field creation form */}
            {selectedText && (
              <Alert severity="success" sx={{ mb: 2 }}>
                <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
                  Selected: "{selectedText}"
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-end' }}>
                  <TextField
                    size="small"
                    label="Field name"
                    value={customFieldName}
                    onChange={(e) => setCustomFieldName(e.target.value.replace(/[^a-zA-Z0-9_]/g, '_'))}
                    placeholder="e.g., source_ip"
                    helperText="Use letters, numbers, and underscores only"
                    autoFocus
                  />
                  <Button
                    variant="contained"
                    onClick={createPatternFromSelection}
                    disabled={!customFieldName}
                  >
                    Add Field
                  </Button>
                  <Button
                    variant="outlined"
                    onClick={() => {
                      setSelectedText('')
                      setCustomFieldName('')
                    }}
                  >
                    Cancel
                  </Button>
                </Box>
              </Alert>
            )}

            {!selectedText && (
              <Alert severity="info" icon={<TipIcon />}>
                <Typography variant="body2">
                  Click and drag to highlight text in your sample log above, then name it to create a field.
                </Typography>
              </Alert>
            )}
          </Paper>
        )}
      </Stack>

      {/* Preset Dialog */}
      <Dialog 
        open={showPresets} 
        onClose={() => setShowPresets(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Typography variant="h6">Choose Common Fields</Typography>
            <IconButton onClick={() => setShowPresets(false)} size="small">
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Select a field type to add to your pattern. You can add multiple fields.
          </Typography>
          <List>
            {fieldPresets.map((preset) => (
              <ListItem key={preset.id} disablePadding>
                <ListItemButton onClick={() => applyPreset(preset)}>
                  <ListItemIcon>{preset.icon}</ListItemIcon>
                  <ListItemText
                    primary={preset.name}
                    secondary={
                      <>
                        <Typography variant="caption" component="span" display="block">
                          {preset.description}
                        </Typography>
                        <Typography variant="caption" component="span" sx={{ fontStyle: 'italic', color: 'text.disabled' }}>
                          Example: {preset.example}
                        </Typography>
                      </>
                    }
                  />
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowPresets(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
