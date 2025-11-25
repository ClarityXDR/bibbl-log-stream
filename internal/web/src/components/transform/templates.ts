// Transform templates for common use cases
// Each template provides a pre-configured route for specific scenarios

export type TransformTemplate = {
  id: string
  title: string
  description: string
  icon: string
  category: 'firewall' | 'security' | 'routing' | 'network' | 'custom'
  difficulty: 'easy' | 'medium' | 'advanced'
  sampleLog: string
  config: {
    routeName: string
    filterPattern: string
    filterDescription: string
    extractedFields: string[]
    pipelineSuggestion?: string
    destinationSuggestion?: string
    enrichment?: {
      enabled: boolean
      type: 'geoip' | 'asn'
    }
  }
}

export const transformTemplates: TransformTemplate[] = [
  {
    id: 'paloalto-sentinel',
    title: 'Send Firewall Logs to Microsoft Sentinel',
    description: 'Parse Palo Alto Networks firewall logs and send them to Microsoft Sentinel for security monitoring',
    icon: '🔥',
    category: 'firewall',
    difficulty: 'easy',
    sampleLog: '1,2023/11/17 14:23:45,001234567890,TRAFFIC,end,2304,2023/11/17 14:23:45,192.168.1.10,203.0.113.50,0.0.0.0,0.0.0.0,Allow-All,,,ssl,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Syslog,2023/11/17 14:23:45,261748,1,54321,443,0,0,0x0,tcp,allow,1024,512,512,3,2023/11/17 14:23:44,0,any,0,12345678,0x0,192.168.0.0-192.168.255.255,203.0.0.0-203.255.255.255,0,3,2,aged-out,0,0,0,0,,PaloAlto,from-policy',
    config: {
      routeName: 'Palo Alto → Sentinel',
      filterPattern: 'true',
      filterDescription: 'Accept all Palo Alto firewall logs',
      extractedFields: ['timestamp', 'source_ip', 'dest_ip', 'action', 'protocol', 'source_port', 'dest_port'],
      pipelineSuggestion: 'paloalto-parser',
      destinationSuggestion: 'sentinel',
      enrichment: {
        enabled: true,
        type: 'geoip'
      }
    }
  },
  {
    id: 'versa-splunk',
    title: 'Route SD-WAN Logs to Splunk',
    description: 'Parse Versa SD-WAN key-value logs and forward them to Splunk for network analytics',
    icon: '🌐',
    category: 'network',
    difficulty: 'easy',
    sampleLog: '2023-11-17T14:23:45.123Z severity=warning device=versa-branch-01 event_type=bandwidth_alert interface=wan0 bandwidth_mbps=95.5 threshold_mbps=100 direction=outbound action=throttle',
    config: {
      routeName: 'Versa SD-WAN → Splunk',
      filterPattern: 'true',
      filterDescription: 'Accept all Versa SD-WAN logs',
      extractedFields: ['timestamp', 'severity', 'device', 'event_type', 'interface'],
      pipelineSuggestion: 'versa-kvp-parser',
      destinationSuggestion: 'splunk',
      enrichment: {
        enabled: false,
        type: 'geoip'
      }
    }
  },
  {
    id: 'severity-routing',
    title: 'Route Critical Alerts Separately',
    description: 'Send high-priority alerts (critical/error) to one destination, everything else to another',
    icon: '⚠️',
    category: 'routing',
    difficulty: 'easy',
    sampleLog: '{"timestamp":"2023-11-17T14:23:45Z","severity":"critical","message":"Database connection failed","service":"api-gateway","host":"prod-server-01"}',
    config: {
      routeName: 'Critical → Priority Destination',
      filterPattern: '_raw.includes("severity") && ["critical","error"].includes(JSON.parse(_raw).severity)',
      filterDescription: 'Match logs with severity: critical or error',
      extractedFields: ['severity', 'message', 'service', 'host'],
      pipelineSuggestion: 'json-parser',
      destinationSuggestion: 'sentinel'
    }
  },
  {
    id: 'ip-extraction',
    title: 'Extract IP Addresses and Add Location',
    description: 'Find IP addresses in any log format and add geographic location information',
    icon: '🌍',
    category: 'security',
    difficulty: 'easy',
    sampleLog: '2023-11-17 14:23:45 [INFO] User login from 203.0.113.50 - username: jsmith - status: success',
    config: {
      routeName: 'IP Extraction + GeoIP',
      filterPattern: '(?P<ip>\\d+\\.\\d+\\.\\d+\\.\\d+)',
      filterDescription: 'Extract IP addresses from logs',
      extractedFields: ['ip', 'geo_city', 'geo_country', 'geo_lat', 'geo_lon'],
      enrichment: {
        enabled: true,
        type: 'geoip'
      }
    }
  },
  {
    id: 'json-filter',
    title: 'Filter JSON Logs by Field Value',
    description: 'Extract specific JSON logs based on field values (like user, service, or status)',
    icon: '🔍',
    category: 'routing',
    difficulty: 'medium',
    sampleLog: '{"timestamp":"2023-11-17T14:23:45Z","user":"admin","action":"login","status":"success","ip":"192.168.1.10"}',
    config: {
      routeName: 'Filter JSON Logs',
      filterPattern: '_raw.includes("status") && JSON.parse(_raw).status === "success"',
      filterDescription: 'Match JSON logs where status equals "success"',
      extractedFields: ['user', 'action', 'status', 'ip'],
      pipelineSuggestion: 'json-parser'
    }
  },
  {
    id: 'cef-parser',
    title: 'Parse Security Events (CEF Format)',
    description: 'Parse Common Event Format (CEF) logs from security devices like firewalls and IDS/IPS',
    icon: '🛡️',
    category: 'security',
    difficulty: 'medium',
    sampleLog: 'CEF:0|Vendor|Product|1.0|100|Event Name|5|src=192.168.1.10 dst=203.0.113.50 spt=54321 dpt=443 proto=TCP act=blocked',
    config: {
      routeName: 'Parse CEF Security Events',
      filterPattern: '(?P<cef>CEF:.*)',
      filterDescription: 'Match CEF-formatted security events',
      extractedFields: ['cef_version', 'vendor', 'product', 'severity', 'src', 'dst'],
      pipelineSuggestion: 'cef-parser',
      destinationSuggestion: 'sentinel'
    }
  },
  {
    id: 'blank-canvas',
    title: 'Start from Scratch',
    description: 'Create a custom transform with full control over all settings',
    icon: '✏️',
    category: 'custom',
    difficulty: 'advanced',
    sampleLog: 'Paste your sample log here...',
    config: {
      routeName: 'Custom Transform',
      filterPattern: 'true',
      filterDescription: 'Custom filter (accepts all logs by default)',
      extractedFields: []
    }
  }
]

export const getTemplateById = (id: string): TransformTemplate | undefined => {
  return transformTemplates.find(t => t.id === id)
}

export const getTemplatesByCategory = (category: TransformTemplate['category']): TransformTemplate[] => {
  return transformTemplates.filter(t => t.category === category)
}

export const getTemplatesByDifficulty = (difficulty: TransformTemplate['difficulty']): TransformTemplate[] => {
  return transformTemplates.filter(t => t.difficulty === difficulty)
}
