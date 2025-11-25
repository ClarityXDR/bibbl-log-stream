import React, { useState, useEffect } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Checkbox,
  Stack,
  Chip,
  Collapse,
  IconButton,
  Alert,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField
} from '@mui/material'
import {
  ExpandMore as ExpandIcon,
  Info as InfoIcon,
  CloudUpload as UploadIcon,
  Refresh as RefreshIcon
} from '@mui/icons-material'

type ProcessingFunction = {
  id: string
  name: string
  friendlyName: string
  description: string
  category: 'parsing' | 'enrichment' | 'filtering' | 'transformation'
  icon: string
  enabled: boolean
  beforeExample?: string
  afterExample?: string
  requiresSetup?: boolean
  setupInstructions?: string
}

type FriendlyPipelineBuilderProps = {
  selectedPipelineId: string
  availableFunctions?: string[]
  onFunctionsChange?: (functions: string[]) => void
}

// Map backend function names to friendly descriptions
const functionLibrary: Record<string, Omit<ProcessingFunction, 'id' | 'enabled'>> = {
  'Parse CEF': {
    name: 'Parse CEF',
    friendlyName: 'Parse Security Events (CEF Format)',
    description: 'Extract fields from Common Event Format logs used by firewalls, IDS/IPS, and security devices',
    category: 'parsing',
    icon: '🛡️',
    beforeExample: 'CEF:0|Vendor|Product|1.0|100|Event|5|src=192.168.1.10',
    afterExample: 'Extracts: vendor, product, severity, src, dst, etc.'
  },
  'Parse Palo Alto': {
    name: 'Parse Palo Alto',
    friendlyName: 'Parse Palo Alto Firewall Logs',
    description: 'Extract fields from Palo Alto Networks firewall CSV logs (TRAFFIC, THREAT, CONFIG)',
    category: 'parsing',
    icon: '🔥',
    beforeExample: '1,2023/11/17 14:23:45,001234567890,TRAFFIC...',
    afterExample: 'Extracts: timestamp, source_ip, dest_ip, action, protocol, ports'
  },
  'Parse Versa KVP': {
    name: 'Parse Versa KVP',
    friendlyName: 'Parse Versa SD-WAN Logs',
    description: 'Extract key=value pairs from Versa Networks SD-WAN logs',
    category: 'parsing',
    icon: '🌐',
    beforeExample: 'severity=warning device=branch-01 event_type=alert',
    afterExample: 'Extracts: severity, device, event_type, and all other key=value pairs'
  },
  'geoip_enrich': {
    name: 'geoip_enrich',
    friendlyName: 'Add Location Info from IP Addresses',
    description: 'Look up geographic location (city, country, coordinates) for IP addresses in your logs',
    category: 'enrichment',
    icon: '🌍',
    beforeExample: 'IP: 203.0.113.50',
    afterExample: 'Adds: geo_city, geo_country, geo_lat, geo_lon, geo_timezone',
    requiresSetup: true,
    setupInstructions: 'Upload a MaxMind GeoIP database (.mmdb file) to enable this feature'
  },
  'asn_enrich': {
    name: 'asn_enrich',
    friendlyName: 'Add Network Owner Info (ASN)',
    description: 'Look up which organization owns the IP address (ISP, company, etc.)',
    category: 'enrichment',
    icon: '🏢',
    beforeExample: 'IP: 8.8.8.8',
    afterExample: 'Adds: asn_number, asn_org (e.g., "Google LLC")',
    requiresSetup: true,
    setupInstructions: 'Upload a MaxMind ASN database (.mmdb file) to enable this feature'
  },
  'redact_pii': {
    name: 'redact_pii',
    friendlyName: 'Remove Sensitive Data (PII)',
    description: 'Automatically mask social security numbers, credit cards, emails, and phone numbers',
    category: 'filtering',
    icon: '🔒',
    beforeExample: 'SSN: 123-45-6789, Card: 4111-1111-1111-1111',
    afterExample: 'SSN: ***-**-****, Card: ****-****-****-****'
  }
}

export default function FriendlyPipelineBuilder({
  selectedPipelineId,
  availableFunctions = [],
  onFunctionsChange
}: FriendlyPipelineBuilderProps) {
  const [functions, setFunctions] = useState<ProcessingFunction[]>([])
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [showGeoIPSetup, setShowGeoIPSetup] = useState(false)
  const [geoipStatus, setGeoipStatus] = useState<{ loaded: boolean; path?: string }>({ loaded: false })

  // Initialize functions from available list
  useEffect(() => {
    const initialized = availableFunctions.map((fnName, index) => {
      const library = functionLibrary[fnName] || {
        name: fnName,
        friendlyName: fnName,
        description: 'Process logs with this function',
        category: 'transformation' as const,
        icon: '⚙️'
      }
      
      return {
        id: `fn-${index}`,
        enabled: true,
        ...library
      }
    })
    
    setFunctions(initialized)
  }, [availableFunctions])

  // Load GeoIP status
  useEffect(() => {
    loadGeoIPStatus()
  }, [])

  const loadGeoIPStatus = async () => {
    try {
      const res = await fetch('/api/v1/enrich/geoip/status')
      const data = await res.json()
      setGeoipStatus(data)
    } catch {
      // Ignore errors
    }
  }

  const uploadGeoIPDatabase = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    
    try {
      await fetch('/api/v1/enrich/geoip/upload', {
        method: 'POST',
        body: formData
      })
      await loadGeoIPStatus()
      setShowGeoIPSetup(false)
    } catch (err) {
      console.error('Failed to upload GeoIP database:', err)
    }
  }

  const toggleFunction = (id: string) => {
    setFunctions(prev => {
      const updated = prev.map(fn => 
        fn.id === id ? { ...fn, enabled: !fn.enabled } : fn
      )
      
      // Notify parent of enabled function names
      if (onFunctionsChange) {
        const enabledNames = updated.filter(f => f.enabled).map(f => f.name)
        onFunctionsChange(enabledNames)
      }
      
      return updated
    })
  }

  const getCategoryIcon = (category: ProcessingFunction['category']) => {
    switch (category) {
      case 'parsing': return '📋'
      case 'enrichment': return '✨'
      case 'filtering': return '🔍'
      case 'transformation': return '⚙️'
      default: return '•'
    }
  }

  const getCategoryColor = (category: ProcessingFunction['category']) => {
    switch (category) {
      case 'parsing': return 'primary'
      case 'enrichment': return 'success'
      case 'filtering': return 'warning'
      case 'transformation': return 'info'
      default: return 'default'
    }
  }

  return (
    <Box>
      <Stack spacing={2}>
        {/* Header */}
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 1 }}>
            Choose How to Process Your Logs
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Select the processing steps you want. They'll run in the order shown below.
          </Typography>
        </Box>

        {/* No pipeline selected */}
        {!selectedPipelineId && (
          <Alert severity="info">
            <Typography variant="body2">
              Select a pipeline from the previous step to see available processing options
            </Typography>
          </Alert>
        )}

        {/* Function Cards */}
        {functions.length === 0 && selectedPipelineId && (
          <Alert severity="warning">
            <Typography variant="body2">
              This pipeline doesn't have any processing functions configured yet.
            </Typography>
          </Alert>
        )}

        {functions.map((fn, index) => (
          <Card
            key={fn.id}
            variant="outlined"
            sx={{
              opacity: fn.enabled ? 1 : 0.6,
              transition: 'all 0.2s',
              border: fn.enabled ? 2 : 1,
              borderColor: fn.enabled ? `${getCategoryColor(fn.category)}.main` : 'divider'
            }}
          >
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                {/* Checkbox */}
                <Checkbox
                  checked={fn.enabled}
                  onChange={() => toggleFunction(fn.id)}
                  sx={{ mt: -1 }}
                />

                {/* Content */}
                <Box sx={{ flex: 1 }}>
                  {/* Title Row */}
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                    <Typography sx={{ fontSize: 28 }}>{fn.icon}</Typography>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="h6" sx={{ fontWeight: 600 }}>
                        {fn.friendlyName}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 0.5, mt: 0.5 }}>
                        <Chip
                          label={`Step ${index + 1}`}
                          size="small"
                          sx={{ height: 20, fontSize: '0.7rem' }}
                        />
                        <Chip
                          icon={<span>{getCategoryIcon(fn.category)}</span>}
                          label={fn.category}
                          size="small"
                          color={getCategoryColor(fn.category)}
                          variant="outlined"
                          sx={{ height: 20, fontSize: '0.7rem', textTransform: 'capitalize' }}
                        />
                      </Box>
                    </Box>
                    <IconButton
                      size="small"
                      onClick={() => setExpandedId(expandedId === fn.id ? null : fn.id)}
                    >
                      <ExpandIcon
                        sx={{
                          transform: expandedId === fn.id ? 'rotate(180deg)' : 'rotate(0deg)',
                          transition: 'transform 0.2s'
                        }}
                      />
                    </IconButton>
                  </Box>

                  {/* Description */}
                  <Typography variant="body2" color="text.secondary">
                    {fn.description}
                  </Typography>

                  {/* Setup Required Alert */}
                  {fn.requiresSetup && !geoipStatus.loaded && fn.enabled && (
                    <Alert 
                      severity="warning" 
                      sx={{ mt: 1 }}
                      action={
                        <Button size="small" onClick={() => setShowGeoIPSetup(true)}>
                          Setup
                        </Button>
                      }
                    >
                      <Typography variant="caption">
                        {fn.setupInstructions}
                      </Typography>
                    </Alert>
                  )}

                  {/* Expanded Details */}
                  <Collapse in={expandedId === fn.id}>
                    <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1 }}>
                        Example:
                      </Typography>
                      {fn.beforeExample && (
                        <Box sx={{ mb: 1 }}>
                          <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                            Before:
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{
                              display: 'block',
                              fontFamily: 'monospace',
                              p: 1,
                              bgcolor: 'background.paper',
                              borderRadius: 0.5,
                              mt: 0.5
                            }}
                          >
                            {fn.beforeExample}
                          </Typography>
                        </Box>
                      )}
                      {fn.afterExample && (
                        <Box>
                          <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                            After:
                          </Typography>
                          <Typography
                            variant="caption"
                            sx={{
                              display: 'block',
                              fontFamily: 'monospace',
                              p: 1,
                              bgcolor: 'success.50',
                              borderRadius: 0.5,
                              mt: 0.5,
                              color: 'success.dark'
                            }}
                          >
                            {fn.afterExample}
                          </Typography>
                        </Box>
                      )}
                    </Box>
                  </Collapse>
                </Box>
              </Box>
            </CardContent>
          </Card>
        ))}

        {/* Info Alert */}
        {functions.length > 0 && (
          <Alert severity="info" icon={<InfoIcon />}>
            <Typography variant="body2">
              <strong>Tip:</strong> Functions run in order from top to bottom. 
              For example, parsing happens first, then enrichment, then filtering.
            </Typography>
          </Alert>
        )}
      </Stack>

      {/* GeoIP Setup Dialog */}
      <Dialog open={showGeoIPSetup} onClose={() => setShowGeoIPSetup(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Setup GeoIP Enrichment</DialogTitle>
        <DialogContent>
          <Stack spacing={2}>
            <Alert severity="info">
              <Typography variant="body2">
                To add location information to IP addresses, you need a MaxMind GeoIP database.
              </Typography>
            </Alert>

            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Status:
              </Typography>
              <Chip
                label={geoipStatus.loaded ? 'Database loaded ✓' : 'No database'}
                color={geoipStatus.loaded ? 'success' : 'default'}
              />
              {geoipStatus.path && (
                <Typography variant="caption" sx={{ display: 'block', mt: 1 }}>
                  File: {geoipStatus.path.split(/[\\/]/).pop()}
                </Typography>
              )}
            </Box>

            <Box>
              <Typography variant="subtitle2" gutterBottom>
                Upload Database:
              </Typography>
              <Button
                variant="contained"
                component="label"
                startIcon={<UploadIcon />}
                fullWidth
              >
                Choose .mmdb file
                <input
                  hidden
                  type="file"
                  accept=".mmdb"
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (file) uploadGeoIPDatabase(file)
                  }}
                />
              </Button>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                You can download GeoLite2 City database from MaxMind.com (free account required)
              </Typography>
            </Box>

            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={loadGeoIPStatus}
              fullWidth
            >
              Refresh Status
            </Button>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowGeoIPSetup(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
