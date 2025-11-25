import React, { useState } from 'react'
import {
  Box,
  Card,
  CardContent,
  CardActionArea,
  Typography,
  Chip,
  TextField,
  InputAdornment,
  Grid,
  ToggleButtonGroup,
  ToggleButton,
  Stack,
  Tooltip,
  Zoom
} from '@mui/material'
import {
  Search as SearchIcon,
  Security as SecurityIcon,
  NetworkCheck as NetworkIcon,
  Cloud as CloudIcon,
  Apps as AppsIcon,
  Create as CreateIcon,
  EmojiObjects as LightbulbIcon
} from '@mui/icons-material'
import { sourceTemplates, SourceTemplate } from './sourceTemplates'

type SourceTemplateGalleryProps = {
  onSelectTemplate: (template: SourceTemplate) => void
}

export default function SourceTemplateGallery({ onSelectTemplate }: SourceTemplateGalleryProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('all')
  const [selectedId, setSelectedId] = useState<string | null>(null)

  const handleTemplateSelect = (template: SourceTemplate) => {
    setSelectedId(template.id)
    // Brief celebration pause before transitioning
    setTimeout(() => {
      onSelectTemplate(template)
    }, 300)
  }

  // Filter templates
  const filteredTemplates = sourceTemplates.filter(template => {
    const matchesSearch = searchQuery === '' || 
      template.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      template.description.toLowerCase().includes(searchQuery.toLowerCase())
    
    const matchesCategory = categoryFilter === 'all' || template.category === categoryFilter
    
    return matchesSearch && matchesCategory
  })

  const categoryIcons: Record<string, React.ReactNode> = {
    network: <NetworkIcon />,
    cloud: <CloudIcon />,
    security: <SecurityIcon />,
    application: <AppsIcon />,
    custom: <CreateIcon />
  }

  const difficultyColors: Record<SourceTemplate['difficulty'], 'success' | 'warning' | 'error'> = {
    easy: 'success',
    medium: 'warning',
    advanced: 'error'
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ mb: 4, textAlign: 'center' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 1, mb: 2 }}>
          <LightbulbIcon sx={{ fontSize: 40, color: 'primary.main' }} />
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Which data source do you want to add?
          </Typography>
        </Box>
        <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
          Choose a template to get started, or create your own from scratch
        </Typography>

        {/* Search */}
        <TextField
          fullWidth
          size="medium"
          placeholder="Search source templates..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          sx={{ maxWidth: 600, mx: 'auto', mb: 3 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            )
          }}
        />

        {/* Category Filter */}
        <ToggleButtonGroup
          value={categoryFilter}
          exclusive
          onChange={(_, val) => val && setCategoryFilter(val)}
          size="small"
          sx={{ flexWrap: 'wrap' }}
        >
          <ToggleButton value="all">All</ToggleButton>
          <ToggleButton value="network">
            <NetworkIcon sx={{ mr: 0.5, fontSize: 18 }} />
            Network
          </ToggleButton>
          <ToggleButton value="cloud">
            <CloudIcon sx={{ mr: 0.5, fontSize: 18 }} />
            Cloud
          </ToggleButton>
          <ToggleButton value="security">
            <SecurityIcon sx={{ mr: 0.5, fontSize: 18 }} />
            Security
          </ToggleButton>
          <ToggleButton value="application">
            <AppsIcon sx={{ mr: 0.5, fontSize: 18 }} />
            Application
          </ToggleButton>
          <ToggleButton value="custom">
            <CreateIcon sx={{ mr: 0.5, fontSize: 18 }} />
            Custom
          </ToggleButton>
        </ToggleButtonGroup>
      </Box>

      {/* Template Grid */}
      <Grid container spacing={3}>
        {filteredTemplates.map((template) => (
          <Grid item xs={12} sm={6} md={4} key={template.id}>
            <Zoom in={selectedId !== template.id} timeout={300}>
              <Card 
                sx={{ 
                  height: '100%',
                  transition: 'transform 0.2s, box-shadow 0.2s',
                  bgcolor: selectedId === template.id ? 'primary.50' : 'background.paper',
                  border: selectedId === template.id ? 2 : 0,
                  borderColor: 'primary.main',
                  '&:hover': {
                    transform: selectedId === null ? 'translateY(-4px)' : 'none',
                    boxShadow: 6
                  }
                }}
              >
                <CardActionArea 
                  onClick={() => handleTemplateSelect(template)}
                  sx={{ height: '100%' }}
                  disabled={selectedId !== null}
                >
                  <CardContent>
                    <Stack spacing={2}>
                      {/* Icon and Title */}
                      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                        <Typography sx={{ fontSize: 40, lineHeight: 1 }}>
                          {template.icon}
                        </Typography>
                        <Box sx={{ flex: 1 }}>
                          <Typography variant="h6" sx={{ fontWeight: 600, mb: 0.5 }}>
                            {template.title}
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                            <Chip 
                              size="small" 
                              label={template.difficulty}
                              color={difficultyColors[template.difficulty]}
                              sx={{ textTransform: 'capitalize' }}
                            />
                            <Tooltip title={template.category}>
                              <Chip 
                                size="small" 
                                icon={categoryIcons[template.category] as any}
                                label={template.category}
                                variant="outlined"
                                sx={{ textTransform: 'capitalize' }}
                              />
                            </Tooltip>
                          </Box>
                        </Box>
                      </Box>

                      {/* Description */}
                      <Typography variant="body2" color="text.secondary" sx={{ minHeight: 60 }}>
                        {template.description}
                      </Typography>

                      {/* Preview Info */}
                      <Box sx={{ pt: 1, borderTop: 1, borderColor: 'divider' }}>
                        <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                          Configuration:
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 0.5, mt: 0.5, flexWrap: 'wrap' }}>
                          <Chip
                            label={template.config.type.toUpperCase()}
                            size="small"
                            variant="outlined"
                            sx={{ fontSize: '0.7rem', height: 20 }}
                          />
                          {template.config.config.port && (
                            <Chip
                              label={`Port ${template.config.config.port}`}
                              size="small"
                              variant="outlined"
                              sx={{ fontSize: '0.7rem', height: 20 }}
                            />
                          )}
                          {template.config.config.protocol && (
                            <Chip
                              label={template.config.config.protocol.toUpperCase()}
                              size="small"
                              variant="outlined"
                              sx={{ fontSize: '0.7rem', height: 20 }}
                            />
                          )}
                        </Box>
                      </Box>
                    </Stack>
                  </CardContent>
                </CardActionArea>
              </Card>
            </Zoom>
          </Grid>
        ))}
      </Grid>

      {/* No Results */}
      {filteredTemplates.length === 0 && (
        <Box sx={{ textAlign: 'center', py: 8 }}>
          <Typography variant="h6" color="text.secondary" gutterBottom>
            No source templates found
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Try a different search term or category filter
          </Typography>
        </Box>
      )}
    </Box>
  )
}
