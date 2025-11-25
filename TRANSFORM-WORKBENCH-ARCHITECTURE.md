# Transform Workbench - Architecture Diagram

## Component Hierarchy

```
TransformWorkbench.tsx (Main Container)
├─ currentView: 'gallery' | 'builder'
│
├─ GALLERY VIEW
│  └─ TemplateGallery.tsx
│     ├─ Search Bar
│     ├─ Category Filters (Firewall, Security, Routing, Network, Custom)
│     └─ Template Cards (7 templates)
│        ├─ Icon + Title + Description
│        ├─ Difficulty Badge (Easy/Medium/Advanced)
│        ├─ Category Badge
│        └─ Field Preview Chips
│
└─ BUILDER VIEW
   ├─ Header Section
   │  ├─ Template Title + Icon
   │  ├─ Description
   │  ├─ Route Name Input
   │  └─ Back Button
   │
   ├─ Step 1: Field Extraction
   │  └─ FieldExtractionUI.tsx
   │     ├─ Help Banner (Instructions)
   │     ├─ Quick Actions
   │     │  ├─ "Accept All JSON Fields" (for JSON logs)
   │     │  ├─ "Choose Common Fields" (preset dialog)
   │     │  └─ "Reset" (clear pattern)
   │     ├─ Current Pattern Display
   │     │  ├─ Pattern Text (monospace)
   │     │  └─ Edit Button (toggle manual mode)
   │     └─ Visual Field Picker
   │        ├─ Sample Log Textarea (highlight text)
   │        ├─ Field Name Input
   │        └─ Add Field Button
   │
   ├─ Step 2: Preview
   │  └─ BeforeAfterPreview.tsx
   │     ├─ Status Bar
   │     │  ├─ Match Count Alert (✓/⚠ with colors)
   │     │  ├─ Field Summary Chips
   │     │  └─ Refresh Button
   │     ├─ Split View
   │     │  ├─ Left Panel: Sample Input
   │     │  │  ├─ Editable Textarea
   │     │  │  └─ Line Count Chip
   │     │  └─ Right Panel: Extracted Data
   │     │     ├─ Human-Readable Format (default)
   │     │     │  └─ Field Cards (key: value pairs)
   │     │     ├─ Raw JSON Toggle Button
   │     │     └─ Raw JSON View (collapsed)
   │     └─ Error Messages (if any)
   │
   ├─ Step 3: Pipeline Selection
   │  ├─ Pipeline Dropdown (Select)
   │  └─ FriendlyPipelineBuilder.tsx
   │     ├─ Function Cards (for each pipeline function)
   │     │  ├─ Icon + Friendly Name
   │     │  ├─ Description (plain English)
   │     │  ├─ Category Badge (Parsing/Enrichment/Filtering/Transformation)
   │     │  ├─ Step Number Chip
   │     │  ├─ Enable/Disable Checkbox
   │     │  └─ Expandable Details
   │     │     ├─ Before Example
   │     │     └─ After Example
   │     └─ Setup Dialogs
   │        └─ GeoIP Setup Dialog
   │           ├─ Status Display
   │           ├─ File Upload Button (.mmdb)
   │           └─ Refresh Status Button
   │
   ├─ Step 4: Destination Selection
   │  └─ Destination Dropdown (Select)
   │
   ├─ Validation Section
   │  ├─ Inline Validation Alerts
   │  └─ Save Error Alerts
   │
   └─ Action Buttons
      ├─ Cancel (back to gallery)
      └─ Save Transform (validates then saves)
```

## Data Flow

```
┌─────────────────────┐
│  Template Gallery   │
│  (User picks goal)  │
└──────────┬──────────┘
           │ onSelectTemplate(template)
           ▼
┌─────────────────────┐
│   Builder View      │
│ (Pre-populate form) │
└──────────┬──────────┘
           │
           ├─── routeName ◄─────────────────┐
           ├─── sampleLog (from template)   │
           ├─── pattern (from template)     │
           ├─── pipelineId (auto-suggest)   │
           └─── destinationId (auto-suggest)│
                                            │
┌───────────────────────────────────────────┤
│ Field Extraction UI                       │
│ • User edits pattern                      │
│ • Picks common fields                     │
│ • Highlights text → creates fields        │
└──────────┬────────────────────────────────┘
           │ onPatternChange(newPattern)
           ▼
┌───────────────────────────────────────────┐
│ Before/After Preview                      │
│ • Auto-runs on pattern/sample change     │
│ • Calls /api/v1/preview/regex             │
│ • Optionally /api/v1/preview/enrich       │
│ • Returns PreviewResult                   │
└──────────┬────────────────────────────────┘
           │ onPreviewResult(result)
           │ { matchedLines, totalLines,
           │   extractedFields, errors }
           ▼
┌───────────────────────────────────────────┐
│ Validation Logic                          │
│ • Check routeName not empty               │
│ • Check pipelineId selected               │
│ • Check destinationId selected            │
│ • Warn if matchedLines === 0              │
└──────────┬────────────────────────────────┘
           │ isFormValid()
           ▼
┌───────────────────────────────────────────┐
│ Save Route                                │
│ POST /api/v1/routes                       │
│ { name, filter, pipelineID, destination } │
└──────────┬────────────────────────────────┘
           │ Success
           ▼
┌───────────────────────────────────────────┐
│ Success Snackbar                          │
│ "Transform saved! Returning to gallery"   │
│ → Auto-redirect after 2s                  │
└───────────────────────────────────────────┘
```

## API Integration

```
┌──────────────────────────────────────────────────────┐
│                    Frontend (React)                  │
└─────────────┬────────────────────────────────────────┘
              │
              ├─ GET /api/v1/routes
              │  (Load existing routes list)
              │
              ├─ GET /api/v1/pipelines
              │  (Load pipeline dropdown options)
              │
              ├─ GET /api/v1/destinations
              │  (Load destination dropdown options)
              │
              ├─ POST /api/v1/preview/regex
              │  Body: { sample, pattern }
              │  Returns: { captures: {...} }
              │  (Used by BeforeAfterPreview)
              │
              ├─ POST /api/v1/preview/enrich
              │  Body: { sample, pattern, ip? }
              │  Returns: { geo: {...}, enriched: {...} }
              │  (Optional, if enrichment enabled)
              │
              ├─ GET /api/v1/enrich/geoip/status
              │  Returns: { loaded: bool, path: string }
              │  (Check if GeoIP database available)
              │
              ├─ POST /api/v1/enrich/geoip/upload
              │  FormData: file (.mmdb)
              │  (Upload GeoIP database)
              │
              └─ POST /api/v1/routes
                 Body: { name, filter, pipelineID, destination, final }
                 Returns: { id, ...route }
                 (Save the configured transform)
```

## State Management

```
TransformWorkbench Component State:
┌───────────────────────────────────────┐
│ currentView: 'gallery' | 'builder'    │ ◄─ Toggle view
│ selectedTemplate: Template | null     │ ◄─ Template selection
│ routeName: string                     │ ◄─ User input
│ pattern: string                       │ ◄─ Field extraction
│ sampleLog: string                     │ ◄─ User input
│ pipeId: string                        │ ◄─ Dropdown selection
│ destId: string                        │ ◄─ Dropdown selection
│ previewResult: PreviewResult | null   │ ◄─ Preview component
│ enableEnrichment: boolean             │ ◄─ Checkbox
│ saveOk: boolean                       │ ◄─ Success state
│ saveError: string | null              │ ◄─ Error state
│ history: Array<{pattern, sampleLog}>  │ ◄─ Undo/redo (future)
│ historyIndex: number                  │ ◄─ Undo/redo (future)
└───────────────────────────────────────┘

Child Component Props:
┌───────────────────────────────────────┐
│ TemplateGallery                       │
│   onSelectTemplate: (Template) => void│
├───────────────────────────────────────┤
│ FieldExtractionUI                     │
│   sampleLog: string                   │
│   pattern: string                     │
│   onPatternChange: (string) => void   │
├───────────────────────────────────────┤
│ BeforeAfterPreview                    │
│   sampleInput: string                 │
│   onSampleChange: (string) => void    │
│   pattern: string                     │
│   enrichmentEnabled: boolean          │
│   onPreviewResult: (result) => void   │
│   autoRun: boolean                    │
├───────────────────────────────────────┤
│ FriendlyPipelineBuilder               │
│   selectedPipelineId: string          │
│   availableFunctions: string[]        │
│   onFunctionsChange: (string[]) => void│
└───────────────────────────────────────┘
```

## Template System Architecture

```
templates.ts
├─ TransformTemplate Type Definition
│  ├─ id: string (unique identifier)
│  ├─ title: string (user-facing name)
│  ├─ description: string (what it does)
│  ├─ icon: string (emoji)
│  ├─ category: 'firewall' | 'security' | 'routing' | 'network' | 'custom'
│  ├─ difficulty: 'easy' | 'medium' | 'advanced'
│  ├─ sampleLog: string (example log)
│  └─ config: TemplateConfig
│     ├─ routeName: string
│     ├─ filterPattern: string (regex or 'true')
│     ├─ filterDescription: string (plain English)
│     ├─ extractedFields: string[] (field names)
│     ├─ pipelineSuggestion?: string (match hint)
│     ├─ destinationSuggestion?: string (match hint)
│     └─ enrichment?: { enabled: boolean, type: 'geoip'|'asn' }
│
├─ transformTemplates: TransformTemplate[]
│  ├─ [0] Palo Alto → Sentinel
│  ├─ [1] Versa → Splunk
│  ├─ [2] Severity Routing
│  ├─ [3] IP Extraction + GeoIP
│  ├─ [4] JSON Filter
│  ├─ [5] CEF Parser
│  └─ [6] Start from Scratch
│
└─ Helper Functions
   ├─ getTemplateById(id: string): Template | undefined
   ├─ getTemplatesByCategory(category): Template[]
   └─ getTemplatesByDifficulty(difficulty): Template[]
```

## Pipeline Function Mapping

```
Backend Function Name          Friendly UI Name
────────────────────────────── ──────────────────────────────────
"Parse CEF"                  → "Parse Security Events (CEF Format)"
"Parse Palo Alto"            → "Parse Palo Alto Firewall Logs"
"Parse Versa KVP"            → "Parse Versa SD-WAN Logs"
"geoip_enrich"               → "Add Location Info from IP Addresses"
"asn_enrich"                 → "Add Network Owner Info (ASN)"
"redact_pii"                 → "Remove Sensitive Data (PII)"
[unknown]                    → [shows original name]

Function Library Metadata:
┌────────────────────────────────────────┐
│ functionLibrary: Record<string, {...}> │
│ ├─ name: string (backend function ID)  │
│ ├─ friendlyName: string                │
│ ├─ description: string                 │
│ ├─ category: 'parsing' | 'enrichment'  │
│ │             'filtering' | 'transform' │
│ ├─ icon: string (emoji)                │
│ ├─ beforeExample?: string              │
│ ├─ afterExample?: string               │
│ ├─ requiresSetup?: boolean             │
│ └─ setupInstructions?: string          │
└────────────────────────────────────────┘
```

## User Journey Map

```
Step 1: Gallery (Goal Selection)
┌────────────────────────────────────────┐
│ "What do you want to do?"              │
│ [Search: _________]                    │
│ [All][Firewall][Security][Routing]...  │
│                                        │
│ ┌────────┐ ┌────────┐ ┌────────┐      │
│ │🔥 Send │ │🌐 Route│ │⚠️ Route│      │
│ │Firewall│ │SD-WAN  │ │Critical│      │
│ │to Sen..│ │to Splu.│ │Alerts  │      │
│ │  Easy  │ │  Easy  │ │  Easy  │      │
│ └────────┘ └────────┘ └────────┘      │
│         [Click any card]               │
└────────────────────────────────────────┘
                  │
                  ▼
Step 2: Builder - Field Extraction
┌────────────────────────────────────────┐
│ 🔥 Send Firewall Logs to Sentinel     │
│ [← Back]                               │
│ Route Name: [Palo Alto → Sentinel]    │
│                                        │
│ ℹ️ For JSON logs: Click "Accept All"   │
│    For text logs: Click "Choose..."   │
│                                        │
│ [Accept All JSON] [Choose Common]      │
│                                        │
│ Pattern: true (matches all logs)       │
└────────────────────────────────────────┘
                  │
                  ▼
Step 3: Preview Results
┌────────────────────────────────────────┐
│ ✓ Matched 20 of 20 sample logs        │
│ Extracted 7 fields: timestamp, src_ip..│
│                                        │
│ ┌─────────────┐ ┌─────────────┐       │
│ │ Sample Logs │ │ Extracted   │       │
│ │ (Before)    │ │ Data (After)│       │
│ │             │ │             │       │
│ │ [Paste here]│ │ ✨ Shows    │       │
│ │             │ │   fields    │       │
│ └─────────────┘ └─────────────┘       │
└────────────────────────────────────────┘
                  │
                  ▼
Step 4: Choose Processing & Destination
┌────────────────────────────────────────┐
│ Pipeline: [Palo Alto Parser ▼]        │
│                                        │
│ ✓ 🛡️ Parse Palo Alto Firewall Logs    │
│ ✓ 🌍 Add Location Info from IPs        │
│ ✓ 🔒 Remove Sensitive Data             │
│                                        │
│ Destination: [Microsoft Sentinel ▼]   │
│                                        │
│ [Cancel] [Save Transform]              │
└────────────────────────────────────────┘
                  │
                  ▼
Step 5: Success & Return
┌────────────────────────────────────────┐
│ ✓ Transform saved successfully!       │
│   Returning to gallery...              │
│                                        │
│ [Auto-redirect after 2 seconds]        │
└────────────────────────────────────────┘
```

## Key Design Principles

1. **Goal-First**: Start with "what do you want to do?" not "configure a route"
2. **Progressive Disclosure**: Show simple options first, hide complexity
3. **Visual Feedback**: Live preview, color-coded alerts, match counts
4. **Plain English**: No jargon, friendly field names, helpful descriptions
5. **Guided Flow**: 4 clear steps with validation at each stage
6. **Templates**: 80% use case coverage with pre-configured templates
7. **Escape Hatches**: "Back to gallery", "Start from scratch", "Reset"
8. **Instant Feedback**: Debounced preview (400ms), inline validation
9. **Error Recovery**: Clear error messages, form persists on error
10. **Accessibility**: Keyboard navigation, screen reader support, focus management

---

**Legend**:
- `┌─┐` = Component boundary
- `│` = Data flow
- `▼` = User action / state change
- `◄─` = State update
- `→` = API call
- `✓` = Success state
- `⚠` = Warning state
- `❌` = Error state
