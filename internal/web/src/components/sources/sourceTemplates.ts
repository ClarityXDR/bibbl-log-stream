// Source configuration templates for common use cases

export type SourceTemplate = {
  id: string
  title: string
  description: string
  icon: string
  category: 'network' | 'cloud' | 'security' | 'application' | 'custom'
  difficulty: 'easy' | 'medium' | 'advanced'
  config: {
    name: string
    type: 'syslog' | 'http' | 'kafka' | 'file' | 'windows_event' | 'akamai_ds2'
    config: Record<string, any>
  }
}

export const sourceTemplates: SourceTemplate[] = [
  {
    id: 'firewall-syslog',
    title: 'Firewall Syslog Receiver',
    description: 'Receive firewall logs from Palo Alto, Fortinet, or other network security devices over syslog',
    icon: '🔥',
    category: 'security',
    difficulty: 'easy',
    config: {
      name: 'Firewall Syslog',
      type: 'syslog',
      config: {
        host: '0.0.0.0',
        port: 514,
        protocol: 'udp'
      }
    }
  },
  {
    id: 'versa-sdwan-tls',
    title: 'Versa SD-WAN (TLS)',
    description: 'Secure syslog receiver with TLS encryption for Versa Networks SD-WAN appliances',
    icon: '🌐',
    category: 'network',
    difficulty: 'medium',
    config: {
      name: 'Versa SD-WAN',
      type: 'syslog',
      config: {
        host: '0.0.0.0',
        port: 6514,
        protocol: 'tls',
        certFile: '/certs/server.crt',
        keyFile: '/certs/server.key'
      }
    }
  },
  {
    id: 'http-webhook',
    title: 'HTTP Webhook',
    description: 'Accept log data via HTTP POST from applications, APIs, or cloud services',
    icon: '🌍',
    category: 'application',
    difficulty: 'easy',
    config: {
      name: 'HTTP Webhook',
      type: 'http',
      config: {
        port: 8080,
        path: '/logs'
      }
    }
  },
  {
    id: 'akamai-datastream',
    title: 'Akamai DataStream 2',
    description: 'Pull edge logs from Akamai DataStream 2 API for CDN and security analytics',
    icon: '☁️',
    category: 'cloud',
    difficulty: 'advanced',
    config: {
      name: 'Akamai DS2',
      type: 'akamai_ds2',
      config: {
        host: 'akab-xxxx.luna.akamaiapis.net',
        clientToken: '',
        clientSecret: '',
        accessToken: '',
        streams: '',
        intervalSeconds: 60
      }
    }
  },
  {
    id: 'kafka-consumer',
    title: 'Kafka Consumer',
    description: 'Consume log streams from Apache Kafka topics for high-throughput ingestion',
    icon: '📨',
    category: 'application',
    difficulty: 'medium',
    config: {
      name: 'Kafka Stream',
      type: 'kafka',
      config: {
        brokers: 'localhost:9092',
        topics: 'logs',
        consumerGroup: 'bibbl-stream'
      }
    }
  },
  {
    id: 'windows-events',
    title: 'Windows Event Log',
    description: 'Collect Windows Event Logs from local or remote Windows systems',
    icon: '🪟',
    category: 'security',
    difficulty: 'medium',
    config: {
      name: 'Windows Events',
      type: 'windows_event',
      config: {}
    }
  },
  {
    id: 'syslog-tcp',
    title: 'Generic Syslog (TCP)',
    description: 'TCP-based syslog receiver for reliable log delivery from any syslog-compatible source',
    icon: '📡',
    category: 'network',
    difficulty: 'easy',
    config: {
      name: 'Syslog TCP',
      type: 'syslog',
      config: {
        host: '0.0.0.0',
        port: 514,
        protocol: 'tcp'
      }
    }
  },
  {
    id: 'custom-source',
    title: 'Start from Scratch',
    description: 'Configure a custom source with your own settings',
    icon: '✏️',
    category: 'custom',
    difficulty: 'easy',
    config: {
      name: 'Custom Source',
      type: 'syslog',
      config: {
        host: '0.0.0.0',
        port: 514,
        protocol: 'udp'
      }
    }
  }
]

// Helper to get template by ID
export const getSourceTemplateById = (id: string): SourceTemplate | undefined => {
  return sourceTemplates.find(t => t.id === id)
}

// Helper to filter templates by category
export const getSourceTemplatesByCategory = (category: string): SourceTemplate[] => {
  if (category === 'all') return sourceTemplates
  return sourceTemplates.filter(t => t.category === category)
}
