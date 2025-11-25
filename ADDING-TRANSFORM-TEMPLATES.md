# Quick Guide: Adding New Transform Templates

This guide shows how to add new templates to the Transform Workbench gallery.

## File to Edit

**Location**: `internal/web/src/components/transform/templates.ts`

## Template Structure

```typescript
{
  id: 'unique-kebab-case-id',           // Unique identifier
  title: 'User-Friendly Title',         // Shows on card
  description: 'What this template does in plain English',
  icon: '🔥',                           // Emoji icon
  category: 'firewall',                 // Gallery filter category
  difficulty: 'easy',                   // Difficulty badge
  sampleLog: 'Example log here...',    // Pre-fills preview
  config: {
    routeName: 'My Route Name',
    filterPattern: 'regex or "true"',
    filterDescription: 'Plain English explanation',
    extractedFields: ['field1', 'field2'], // Shows on card
    pipelineSuggestion: 'keyword',      // Auto-selects pipeline (optional)
    destinationSuggestion: 'keyword',   // Auto-selects destination (optional)
    enrichment: {                       // Optional
      enabled: true,
      type: 'geoip'
    }
  }
}
```

## Step-by-Step: Add a New Template

### 1. Open the templates file
```bash
# Location
internal/web/src/components/transform/templates.ts
```

### 2. Add to `transformTemplates` array

Find this line:
```typescript
export const transformTemplates: TransformTemplate[] = [
```

Add your new template to the end (before the closing `]`):

```typescript
  {
    id: 'my-new-template',
    title: 'My New Transform',
    description: 'Send custom logs to my destination',
    icon: '⚡',
    category: 'custom',
    difficulty: 'easy',
    sampleLog: '2023-11-17 14:23:45 [INFO] Sample log message',
    config: {
      routeName: 'My Custom Route',
      filterPattern: 'true',
      filterDescription: 'Accept all logs',
      extractedFields: ['timestamp', 'level', 'message']
    }
  }
```

### 3. Save the file

### 4. Rebuild the UI
```bash
cd internal/web
npm run build
```

### 5. Test
- Navigate to Transform Workbench
- Search for your new template
- Click it and verify fields pre-populate

---

## Examples

### Example 1: Simple Log Filter

```typescript
{
  id: 'auth-failures',
  title: 'Track Failed Login Attempts',
  description: 'Send authentication failure logs to security team',
  icon: '🔐',
  category: 'security',
  difficulty: 'easy',
  sampleLog: '2023-11-17 14:23:45 [ERROR] Authentication failed for user: admin from IP: 192.168.1.100',
  config: {
    routeName: 'Failed Auth → Security Team',
    filterPattern: '_raw.includes("Authentication failed")',
    filterDescription: 'Match logs containing "Authentication failed"',
    extractedFields: ['timestamp', 'user', 'ip'],
    destinationSuggestion: 'splunk'
  }
}
```

### Example 2: Application Performance Monitoring

```typescript
{
  id: 'slow-queries',
  title: 'Monitor Slow Database Queries',
  description: 'Track queries taking longer than 1 second for performance tuning',
  icon: '🐌',
  category: 'routing',
  difficulty: 'medium',
  sampleLog: '{"timestamp":"2023-11-17T14:23:45Z","query":"SELECT * FROM users","duration_ms":1523,"slow":true}',
  config: {
    routeName: 'Slow DB Queries → Performance',
    filterPattern: '_raw.includes("slow") && JSON.parse(_raw).slow === true',
    filterDescription: 'Match JSON logs where slow field is true',
    extractedFields: ['timestamp', 'query', 'duration_ms'],
    pipelineSuggestion: 'json-parser',
    destinationSuggestion: 'datadog'
  }
}
```

### Example 3: Multi-Field Extraction with Enrichment

```typescript
{
  id: 'web-traffic',
  title: 'Analyze Web Traffic with Location',
  description: 'Parse Apache/Nginx access logs and add visitor geographic info',
  icon: '🌍',
  category: 'network',
  difficulty: 'medium',
  sampleLog: '203.0.113.50 - - [17/Nov/2023:14:23:45 +0000] "GET /api/users HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"',
  config: {
    routeName: 'Web Traffic + GeoIP',
    filterPattern: '(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+).*?\\[(?P<timestamp>[^\\]]+)\\]\\s+"(?P<method>\\w+)\\s+(?P<path>[^\\s]+)',
    filterDescription: 'Extract IP, timestamp, HTTP method, and path from access logs',
    extractedFields: ['ip', 'timestamp', 'method', 'path', 'geo_city', 'geo_country'],
    pipelineSuggestion: 'apache-parser',
    destinationSuggestion: 'elasticsearch',
    enrichment: {
      enabled: true,
      type: 'geoip'
    }
  }
}
```

### Example 4: Severity-Based Routing

```typescript
{
  id: 'error-escalation',
  title: 'Escalate Errors to On-Call Team',
  description: 'Send ERROR and CRITICAL logs to PagerDuty, everything else to standard logging',
  icon: '🚨',
  category: 'routing',
  difficulty: 'medium',
  sampleLog: '{"timestamp":"2023-11-17T14:23:45Z","level":"ERROR","service":"payment-api","message":"Payment gateway timeout","trace_id":"abc123"}',
  config: {
    routeName: 'Errors → On-Call',
    filterPattern: '_raw.includes("level") && ["ERROR","CRITICAL"].includes(JSON.parse(_raw).level)',
    filterDescription: 'Match logs with level ERROR or CRITICAL',
    extractedFields: ['timestamp', 'level', 'service', 'message', 'trace_id'],
    pipelineSuggestion: 'json-parser',
    destinationSuggestion: 'pagerduty'
  }
}
```

---

## Field Reference

### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `id` | string | Unique identifier (kebab-case) | `'my-template'` |
| `title` | string | Card title (user-facing) | `'Send Logs to Splunk'` |
| `description` | string | What template does (1-2 sentences) | `'Parse firewall logs and send...'` |
| `icon` | string | Emoji icon | `'🔥'` |
| `category` | enum | Gallery filter category | `'firewall'` |
| `difficulty` | enum | Difficulty badge | `'easy'` |
| `sampleLog` | string | Example log (pre-fills preview) | `'10.0.0.1 app - demo'` |
| `config.routeName` | string | Default route name | `'Firewall → Sentinel'` |
| `config.filterPattern` | string | Regex or `'true'` | `'(?P<ip>\\d+...)'` or `'true'` |
| `config.filterDescription` | string | Plain English pattern explanation | `'Extract IP addresses'` |
| `config.extractedFields` | string[] | Field names (shows on card) | `['ip', 'timestamp']` |

### Optional Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `config.pipelineSuggestion` | string | Keyword to match pipeline name | `'paloalto'` |
| `config.destinationSuggestion` | string | Keyword to match destination type | `'sentinel'` |
| `config.enrichment.enabled` | boolean | Enable GeoIP checkbox | `true` |
| `config.enrichment.type` | enum | Enrichment type | `'geoip'` or `'asn'` |

### Field Enums

**category** (affects filter buttons):
- `'firewall'` - Firewall logs
- `'security'` - Security/threat logs
- `'routing'` - Log routing/filtering
- `'network'` - Network/infrastructure logs
- `'custom'` - Custom/advanced templates

**difficulty** (affects badge color):
- `'easy'` - Green badge, beginner-friendly
- `'medium'` - Orange badge, some complexity
- `'advanced'` - Red badge, expert users

**enrichment.type**:
- `'geoip'` - IP → Location (city, country, lat/lon)
- `'asn'` - IP → ASN/Organization

---

## Tips for Great Templates

### 1. **Realistic Sample Logs**
❌ Bad: `'sample log here'`  
✅ Good: `'2023-11-17 14:23:45 192.168.1.10 GET /api/users 200'`

### 2. **Clear Descriptions**
❌ Bad: `'Process logs'`  
✅ Good: `'Parse Apache access logs and send high-traffic pages to analytics dashboard'`

### 3. **Specific Route Names**
❌ Bad: `'New Route'`  
✅ Good: `'Apache Access → Analytics'`

### 4. **Helpful Field Names**
❌ Bad: `['field1', 'field2']`  
✅ Good: `['source_ip', 'http_method', 'response_code']`

### 5. **Plain English Filter Descriptions**
❌ Bad: `'Regex pattern'`  
✅ Good: `'Extract IP addresses and HTTP request details from access logs'`

### 6. **Use Emojis Wisely**
- 🔥 Firewall
- 🌐 Network/SD-WAN
- 🛡️ Security/IDS
- 🌍 GeoIP/Location
- 📊 Analytics/Metrics
- ⚠️ Alerts/Errors
- 🔒 Encryption/Privacy
- 📝 Logging/Audit
- 🚨 Incidents/Escalation
- ⚡ Performance/Speed

---

## Testing Your Template

### Checklist
- [ ] Template appears in gallery
- [ ] Icon renders correctly
- [ ] Difficulty badge shows correct color
- [ ] Category filter works
- [ ] Search finds template by title/description
- [ ] Clicking card loads builder
- [ ] Route name pre-filled
- [ ] Sample log appears in preview
- [ ] Pattern set correctly
- [ ] Fields extracted in preview
- [ ] Pipeline auto-selected (if suggestion provided)
- [ ] Destination auto-selected (if suggestion provided)
- [ ] Enrichment checkbox state correct

### Common Issues

**Template doesn't appear**
- Check syntax: Missing comma, bracket mismatch
- Verify file saved and UI rebuilt (`npm run build`)

**Pattern doesn't work**
- Test regex at regex101.com
- Escape backslashes: `\\d+` not `\d+`
- For JavaScript expressions, use: `_raw.includes(...)`

**Auto-selection doesn't work**
- Suggestion is case-insensitive partial match
- Check actual pipeline/destination names in UI
- Leave empty (`undefined`) if uncertain

**Enrichment not available**
- User must upload .mmdb database first
- Setup dialog appears automatically if `requiresSetup: true`

---

## Advanced: Template Categories

Want to add a new category? Edit two places:

### 1. Type definition (templates.ts)
```typescript
export type TransformTemplate = {
  // ...
  category: 'firewall' | 'security' | 'routing' | 'network' | 'custom' | 'yourNewCategory'
  // ...
}
```

### 2. Gallery icons (TemplateGallery.tsx)
```typescript
const categoryIcons: Record<string, React.ReactNode> = {
  firewall: <FireIcon />,
  security: <SecurityIcon />,
  routing: <RouteIcon />,
  network: <NetworkIcon />,
  custom: <CreateIcon />,
  yourNewCategory: <YourIcon />  // Add this
}
```

---

## Need Help?

- **Template not matching logs?** → Test pattern at `/api/v1/preview/regex` endpoint
- **Want to see example?** → Look at existing templates in `templates.ts`
- **TypeScript errors?** → Run `npm run build` to see specific error messages
- **Template ideas?** → Check user support tickets for common requests

---

**Quick Start**: Copy an existing template similar to your use case, modify the fields, save, rebuild, test!
