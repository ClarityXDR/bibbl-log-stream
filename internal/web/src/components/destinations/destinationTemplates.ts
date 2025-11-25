// Destination configuration templates for common use cases

export type DestinationTemplate = {
  id: string
  title: string
  description: string
  icon: string
  category: 'azure' | 'cloud' | 'siem' | 'storage' | 'custom'
  difficulty: 'easy' | 'medium' | 'advanced'
  config: {
    name: string
    type: 'sentinel' | 'splunk' | 's3' | 'azure_blob' | 'elasticsearch' | 'azure_datalake' | 'azure_loganalytics'
    config: Record<string, any>
  }
}

export const destinationTemplates: DestinationTemplate[] = [
  {
    id: 'sentinel',
    title: 'Microsoft Sentinel',
    description: 'Send logs to Microsoft Sentinel via Data Collection Rules (DCR) for cloud-native SIEM',
    icon: '🛡️',
    category: 'azure',
    difficulty: 'medium',
    config: {
      name: 'Microsoft Sentinel',
      type: 'sentinel',
      config: {
        workspaceId: '',
        dceEndpoint: '',
        dcrId: '',
        tableName: 'Custom_BibblLogs_CL',
        batchMaxEvents: 500,
        batchMaxBytes: 524288,
        flushIntervalSec: 5,
        concurrency: 2,
        compression: 'gzip'
      }
    }
  },
  {
    id: 'azure-log-analytics',
    title: 'Azure Log Analytics',
    description: 'Stream logs to Azure Log Analytics workspace using HTTP Data Collector API',
    icon: '📊',
    category: 'azure',
    difficulty: 'easy',
    config: {
      name: 'Azure Log Analytics',
      type: 'azure_loganalytics',
      config: {
        workspaceID: '',
        sharedKey: '',
        logType: 'SecurityAlerts',
        resourceGroup: '',
        batchMaxEvents: 500,
        batchMaxBytes: 1048576,
        flushIntervalSec: 10,
        concurrency: 2,
        maxRetries: 3
      }
    }
  },
  {
    id: 'azure-datalake',
    title: 'Azure Data Lake Gen2',
    description: 'Archive logs to Azure Data Lake Storage Gen2 for long-term retention and analytics',
    icon: '🏔️',
    category: 'azure',
    difficulty: 'medium',
    config: {
      name: 'Azure Data Lake',
      type: 'azure_datalake',
      config: {
        storageAccount: '',
        filesystem: 'logs',
        directory: 'bibbl/raw/',
        pathTemplate: 'bibbl/raw/${yyyy}/${MM}/${dd}/${HH}/data-${mm}.jsonl',
        format: 'jsonl',
        compression: 'gzip',
        batchMaxEvents: 1000,
        batchMaxBytes: 1048576,
        flushIntervalSec: 5,
        concurrency: 4
      }
    }
  },
  {
    id: 'splunk-hec',
    title: 'Splunk HEC',
    description: 'Forward logs to Splunk via HTTP Event Collector for enterprise search and analysis',
    icon: '🔍',
    category: 'siem',
    difficulty: 'easy',
    config: {
      name: 'Splunk',
      type: 'splunk',
      config: {
        hecUrl: 'https://splunk.example.com:8088',
        hecToken: '',
        index: 'main'
      }
    }
  },
  {
    id: 's3-archive',
    title: 'Amazon S3',
    description: 'Archive logs to Amazon S3 buckets for cost-effective long-term storage',
    icon: '📦',
    category: 'storage',
    difficulty: 'easy',
    config: {
      name: 'Amazon S3',
      type: 's3',
      config: {
        bucket: '',
        region: 'us-east-1',
        prefix: 'logs/'
      }
    }
  },
  {
    id: 'elasticsearch',
    title: 'Elasticsearch',
    description: 'Index logs into Elasticsearch for full-text search and visualization with Kibana',
    icon: '🔎',
    category: 'siem',
    difficulty: 'medium',
    config: {
      name: 'Elasticsearch',
      type: 'elasticsearch',
      config: {}
    }
  },
  {
    id: 'azure-blob',
    title: 'Azure Blob Storage',
    description: 'Store logs in Azure Blob Storage for low-cost archival and compliance',
    icon: '💾',
    category: 'storage',
    difficulty: 'easy',
    config: {
      name: 'Azure Blob',
      type: 'azure_blob',
      config: {}
    }
  },
  {
    id: 'custom-destination',
    title: 'Start from Scratch',
    description: 'Configure a custom destination with your own settings',
    icon: '✏️',
    category: 'custom',
    difficulty: 'easy',
    config: {
      name: 'Custom Destination',
      type: 'sentinel',
      config: {}
    }
  }
]

// Helper to get template by ID
export const getDestinationTemplateById = (id: string): DestinationTemplate | undefined => {
  return destinationTemplates.find(t => t.id === id)
}

// Helper to filter templates by category
export const getDestinationTemplatesByCategory = (category: string): DestinationTemplate[] => {
  if (category === 'all') return destinationTemplates
  return destinationTemplates.filter(t => t.category === category)
}
