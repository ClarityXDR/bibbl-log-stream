# Bibbl Log Stream - Enterprise Log Forwarding Platform

## Project Overview
Build a high-performance, cost-effective alternative to Cribl Stream that provides enterprise-grade log forwarding, processing, and routing capabilities. The solution must be compiled into single executables for both Windows (.exe) and Linux, with no Java dependencies.

## Core Requirements

### 1. Architecture
- **Single Binary**: Compile entire application into one executable for each platform (Windows .exe and Linux ELF)
- **Cross-Platform**: Single codebase that compiles for Windows, Linux (x86_64 and ARM64)
- **No JVM**: Pure Go implementation, no Java Virtual Machine required
- **High Performance**: Use Go's concurrency model (goroutines, channels) for parallel processing
- **Memory Efficient**: Target 10x less memory usage than Logstash/Cribl
- **Horizontal Scaling**: Support multiple worker processes without additional licensing

### 2. Data Pipeline Components

#### Inputs (Must Support)
- Syslog (UDP/TCP with TLS)
- HTTP/HTTPS endpoints
- Windows Event Log (Windows only)
- File monitoring (tail -f style)
- Beats protocol (Filebeat, Winlogbeat compatibility)
- Kafka consumer
- Azure Event Hub
- AWS Kinesis
- Systemd journal (Linux only)
- Linux audit logs (Linux only)

#### Processing Capabilities
- **Parsing**: JSON, CSV, XML, Windows Events, Syslog formats
- **Field Extraction**: Regex, Grok patterns, Key-Value pairs
- **Transformation**: Field renaming, type conversion, enrichment
- **Filtering**: Drop events, route based on conditions
- **Aggregation**: Time-based windows, counting, metrics
- **Data Masking**: PII redaction, field encryption
- **CEF/LEEF Formatting**: Support Common Event Format for SIEM compatibility

#### Outputs (Must Support)

##### Priority 1 - Microsoft Security Stack
- **Microsoft Sentinel Data Lake** (Critical)
  - Azure Data Explorer (ADX/Kusto) direct ingestion
  - Support for custom log tables via Data Collection Rules (DCR)
  - Azure Monitor Ingestion API integration
  - Batch optimization for cost efficiency
  - Automatic schema detection and mapping
  - Support for Advanced Hunting tables in Defender XDR
- **Azure Log Analytics Workspace**
  - HTTP Data Collector API
  - Custom log support
- **Azure Event Hub** (for streaming to Sentinel)
- **Azure Blob Storage** (for archive and batch ingestion)

##### Priority 2 - Other Destinations
- Elasticsearch (with bulk API)
- Splunk HEC (HTTP Event Collector)
- S3/GCS for archive
- Kafka producer
- HTTP/HTTPS webhooks
- PostgreSQL/MySQL/MSSQL
- InfluxDB for metrics
- Multiple simultaneous destinations with different formats

### 3. Microsoft Sentinel Integration Requirements

#### Authentication
- **Azure AD/Entra ID**: Service Principal authentication
- **Managed Identity**: Support for Azure-hosted deployments
- **Certificate-based auth**: For high-security environments
- **SAS tokens**: For blob storage operations

#### Data Format Support
- **Kusto Query Language (KQL)** compatible schemas
- **CEF (Common Event Format)** for security events
- **Syslog RFC3164/RFC5424** formats
- **Windows Event Log** native format
- **Custom JSON** with field mapping

#### Sentinel-Specific Features
- **Data Collection Rules (DCR)** integration
  - Dynamic DCR creation and management
  - Transform data at ingestion
  - Route to multiple tables
- **Data Collection Endpoints (DCE)**
  - Private endpoint support
  - Regional deployment
  - Network isolation
- **Custom Log Tables** support
  - Automatic table creation
  - Schema management
  - Retention policy configuration
- **Cost Optimization**
  - Batch compression before sending
  - Deduplication to reduce ingestion costs
  - Sampling options for high-volume logs
  - Support for Basic vs Analytics log tiers

#### Azure Buffer Storage
- **Primary Buffer**: Azure Storage Account with Blob containers
  - Automatic container creation per pipeline
  - Retention policies for processed/failed events
  - Encryption at rest with customer keys
  - Lifecycle management for old data
- **Queue Management**:
  - Azure Queue Storage for work items
  - Dead letter queue for failed events
  - Automatic replay capability
- **Failover Strategy**:
  - Local disk buffer when Azure unreachable
  - Automatic sync when connection restored
  - Configurable buffer size limits

#### Compliance & Security
- **Azure Private Endpoints** support
- **Customer-Managed Keys (CMK)** for encryption
- **Data residency** compliance
- **GDPR/HIPAA** compliant data handling

### 4. Authentication & Portal Requirements

#### SSO Integration with Entra ID
- **OAuth 2.0/OpenID Connect** implementation
- **App Registration** automatic setup
  - Generate client ID/secret
  - Configure redirect URIs
  - Set API permissions
  - Request admin consent
- **RBAC Integration**:
  - Map Entra ID groups to portal roles
  - Admin, Operator, Viewer roles
  - Pipeline-specific permissions
- **Session Management**:
  - JWT token validation
  - Refresh token handling
  - Session timeout configuration
  - Multi-factor authentication support

#### Secure Login Implementation
- **Multi-Provider Support**:
  - Entra ID (primary)
  - LDAP/Active Directory
  - SAML 2.0
  - Local accounts (fallback)
- **Security Features**:
  - Account lockout after failed attempts
  - Password complexity requirements
  - Two-factor authentication (TOTP/SMS)
  - IP whitelisting per user/role
  - Session recording for audit
  - Automatic logout on idle (configurable)
  - Concurrent session limits
- **Token Management**:
  - Short-lived access tokens (15 minutes)
  - Refresh tokens with rotation
  - Token revocation endpoint
  - Secure token storage (httpOnly cookies)
- **Audit Trail**:
  - Login/logout events
  - Failed authentication attempts
  - Permission changes
  - Configuration modifications
  - Export to Sentinel for SIEM

#### Portal Authentication Flow
```yaml
authentication:
  providers:
    - type: entra_id
      tenant_id: "${AZURE_TENANT_ID}"
      client_id: "${APP_CLIENT_ID}"
      client_secret: "${APP_CLIENT_SECRET}"
      redirect_uri: "https://portal.bibbl.local/auth/callback"
      scopes:
        - "User.Read"
        - "Group.Read.All"
      role_mappings:
        admin: "sg-bibbl-admins"
        operator: "sg-bibbl-operators"
        viewer: "sg-bibbl-viewers"
        sandbox_user: "sg-bibbl-sandbox"
    - type: local  # Fallback for initial setup
      users:
        - username: admin
          password_hash: "${ADMIN_PASSWORD_HASH}"
          role: admin
  security:
    session_timeout: 30m
    max_sessions_per_user: 3
    failed_login_attempts: 5
    lockout_duration: 15m
    require_mfa: true
    allowed_ips:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
```

### 5. Azure Deployment Automation Portal

#### ARM Template Builder Page
The portal must include a dedicated "Azure Setup" page with the following features:

##### Setup Wizard Components
1. **Prerequisites Checker**
   - Verify Azure subscription access
   - Check required resource providers
   - Validate permissions
   - Test connectivity to Azure

2. **App Registration Setup**
   - Automated App Registration creation
   - Service Principal generation
   - API permissions configuration
   - Admin consent workflow
   - Certificate or secret generation
   - Store credentials securely

3. **Resource Deployment**
   - **Data Collection Endpoint (DCE)**:
     - Name and region selection
     - Network configuration (public/private)
     - Generate ARM template
   - **Data Collection Rule (DCR)**:
     - Select destination workspace
     - Configure transformation rules
     - Set up custom tables
     - Define stream declarations
   - **Storage Account**:
     - Create buffer storage account
     - Configure blob containers
     - Set retention policies
     - Enable encryption
   - **Key Vault** (optional):
     - Store sensitive configuration
     - Manage certificates
     - Rotate secrets

4. **ARM Template Features**
   - Visual template builder
   - Parameter customization
   - Template validation
   - One-click deployment
   - Export for CI/CD pipelines
   - Import existing templates

##### ARM Template Structure
```json
{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "location": {
      "type": "string",
      "defaultValue": "[resourceGroup().location]"
    },
    "workspaceName": {
      "type": "string"
    },
    "dceName": {
      "type": "string"
    },
    "dcrName": {
      "type": "string"
    },
    "storageAccountName": {
      "type": "string"
    },
    "keyVaultName": {
      "type": "string"
    }
  },
  "resources": [
    {
      "type": "Microsoft.Insights/dataCollectionEndpoints",
      "apiVersion": "2022-06-01",
      "name": "[parameters('dceName')]",
      "location": "[parameters('location')]",
      "properties": {
        "networkAcls": {
          "publicNetworkAccess": "Enabled"
        }
      }
    },
    {
      "type": "Microsoft.Insights/dataCollectionRules",
      "apiVersion": "2022-06-01",
      "name": "[parameters('dcrName')]",
      "location": "[parameters('location')]",
      "properties": {
        "dataCollectionEndpointId": "[resourceId('Microsoft.Insights/dataCollectionEndpoints', parameters('dceName'))]",
        "streamDeclarations": {
          "Custom-BibblStream": {
            "columns": [
              {
                "name": "TimeGenerated",
                "type": "datetime"
              },
              {
                "name": "RawData",
                "type": "string"
              }
            ]
          }
        },
        "destinations": {
          "logAnalytics": [
            {
              "workspaceResourceId": "[parameters('workspaceResourceId')]",
              "name": "workspace"
            }
          ]
        },
        "dataFlows": [
          {
            "streams": ["Custom-BibblStream"],
            "destinations": ["workspace"],
            "transformKql": "source | extend TimeGenerated = now()",
            "outputStream": "Custom-BibblLogs_CL"
          }
        ]
      }
    },
    {
      "type": "Microsoft.Storage/storageAccounts",
      "apiVersion": "2021-09-01",
      "name": "[parameters('storageAccountName')]",
      "location": "[parameters('location')]",
      "sku": {
        "name": "Standard_LRS"
      },
      "kind": "StorageV2",
      "properties": {
        "encryption": {
          "services": {
            "blob": {
              "enabled": true
            }
          }
        }
      }
    }
  ]
}
```

##### Deployment UI Features
```typescript
// Portal page component structure
interface AzureSetupPage {
  sections: {
    authentication: {
      testConnection(): Promise<boolean>
      createAppRegistration(): Promise<AppRegistration>
      configurePermissions(): Promise<void>
    }
    
    resourceBuilder: {
      selectSubscription(): Promise<Subscription>
      selectResourceGroup(): Promise<ResourceGroup>
      configureDCE(): DCEConfig
      configureDCR(): DCRConfig
      configureStorage(): StorageConfig
      generateTemplate(): ARMTemplate
      validateTemplate(): ValidationResult
      deployTemplate(): DeploymentResult
    }
    
    postDeployment: {
      testIngestion(): Promise<TestResult>
      downloadConfig(): ConfigFile
      viewDeploymentStatus(): DeploymentStatus
    }
  }
}
```

### 6. Security Requirements
- **TLS Everything**: All network connections must support TLS 1.2+
- **Certificate Management**: Built-in certificate generation and rotation
- **Authentication**: API keys, OAuth2, LDAP/AD, Azure AD integration
- **RBAC**: Role-based access control for configuration
- **Secrets Management**: Azure Key Vault integration, encrypted local storage
- **Audit Logging**: Track all configuration changes, forward to Sentinel

### 7. Cross-Platform Requirements

#### Platform-Specific Features
```yaml
windows:
  - Windows Event Log collection
  - Windows Service installation
  - WMI queries for system metrics
  - Active Directory integration
  - Windows Certificate Store support
  - Named pipes for IPC

linux:
  - Systemd service installation
  - Systemd journal collection
  - Linux audit log collection
  - SELinux support
  - Unix domain sockets
  - Signal handling (SIGTERM, SIGHUP for reload)
```

#### Build Targets
- **Windows**: 
  - Windows Server 2016+ (amd64)
  - Windows 10/11 (amd64)
- **Linux**:
  - RHEL/CentOS 7+ (amd64, arm64)
  - Ubuntu 18.04+ (amd64, arm64)
  - Debian 10+ (amd64, arm64)
  - Amazon Linux 2 (amd64, arm64)
  - Alpine Linux (amd64, arm64) for containers

### 8. Management Interface

#### Embedded Web UI
- **Dashboard**: Real-time throughput, error rates, resource usage
- **Pipeline Builder**: Visual drag-and-drop pipeline configuration
- **Live Data Preview**: Sample events at any pipeline stage
- **Configuration Management**: YAML/JSON import/export
- **Health Monitoring**: Node status, queue depths, backpressure indicators
- **Sentinel Integration Status**: DCR health, ingestion metrics, cost tracking
- **Azure Setup Wizard**: Guided deployment of Azure resources
- **SSO Configuration**: Entra ID integration setup
- **Grafana Integration**: Embedded dashboards for visualization
- **Filter Sandbox**: Test and develop filters with live preview
- **File Library**: Manage test data and filter configurations

#### Portal Pages Structure
```
/                         - Dashboard (requires auth)
/login                    - SSO login page
/auth/callback           - OAuth callback handler
/setup                   - Initial setup wizard
  /setup/azure           - Azure resources deployment
  /setup/auth            - Entra ID configuration
  /setup/storage         - Buffer storage setup
/pipelines               - Pipeline management
/pipelines/new           - Pipeline builder
/pipelines/:id/edit      - Pipeline editor
/sandbox                 - Filter development sandbox
  /sandbox/editor        - Filter editor with live preview
  /sandbox/library       - Test file library
  /sandbox/history       - Filter test history
  /sandbox/templates     - Filter template gallery
/monitoring              - Monitoring and observability
  /monitoring/grafana    - Embedded Grafana dashboards
  /monitoring/metrics    - Prometheus metrics explorer
  /monitoring/alerts     - Alert configuration
/files                   - File management
  /files/library         - Test data library
  /files/upload          - File upload interface
  /files/filters         - Saved filter configurations
/azure                   - Azure management
  /azure/dcr             - DCR management
  /azure/dce             - DCE management
  /azure/templates       - ARM template builder
  /azure/deployment      - Deployment status
/admin                   - Administration
  /admin/users           - User management
  /admin/roles           - Role assignments
  /admin/audit           - Audit logs
  /admin/security        - Security settings
/api/v1/*               - REST API endpoints
/metrics                - Prometheus metrics endpoint
```

### 9. Monitoring & Observability

#### Prometheus Integration
- **Metrics Collection**:
  - Pipeline throughput (events/sec)
  - Processing latency histograms
  - Error rates by pipeline/processor
  - Queue depths and backpressure
  - Resource usage (CPU, memory, disk)
  - Azure ingestion metrics
  - Authentication success/failure rates
- **Custom Metrics**:
  - Business metrics from processed events
  - Cost tracking for Azure services
  - Data volume by source/destination
- **Service Discovery**:
  - Automatic node discovery in cluster mode
  - Azure Service Discovery for cloud deployments
- **Metric Endpoints**:
  ```yaml
  metrics:
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
      include_go_metrics: true
      custom_labels:
        environment: "production"
        region: "us-east"
  ```

#### Grafana Integration
- **Embedded Dashboards**:
  - Pre-built dashboards for common use cases
  - Real-time pipeline performance
  - Azure Sentinel ingestion monitoring
  - Cost analysis dashboard
  - Security event trends
- **Dashboard Features**:
  - Auto-refresh with configurable intervals
  - Alert annotations
  - Variable templating for multi-tenant
  - Mobile-responsive design
- **Data Sources**:
  - Prometheus (primary)
  - Azure Monitor
  - Elasticsearch (optional)
  - Direct SQL queries
- **Configuration**:
  ```yaml
  grafana:
    enabled: true
    embedded: true  # Run embedded Grafana
    port: 3000
    auth:
      type: proxy  # Use portal authentication
      header: X-WEBAUTH-USER
    datasources:
      - name: prometheus
        type: prometheus
        url: http://localhost:9090
        default: true
      - name: azure_monitor
        type: grafana-azure-monitor-datasource
        json_data:
          tenant_id: "${AZURE_TENANT_ID}"
          client_id: "${MONITORING_CLIENT_ID}"
    dashboards:
      - pipeline_overview
      - azure_ingestion
      - security_events
      - cost_analysis
      - sandbox_usage
  ```

### 10. Filter Sandbox Environment

#### Sandbox Features
- **Interactive Filter Development**:
  - Split-screen before/after view
  - Real-time filter application
  - Syntax highlighting for filter expressions
  - Auto-completion for field names
  - Filter validation and error reporting
- **Test Data Management**:
  - Upload sample log files
  - Capture live stream samples
  - Generate synthetic test data
  - Save and version test datasets
- **Filter Testing Workflow**:
  1. Select or upload test data
  2. Write filter expressions
  3. See immediate before/after comparison
  4. Test with different data sets
  5. Save successful filters to library
  6. Export to pipeline configuration

#### Sandbox UI Components
```typescript
interface SandboxPage {
  components: {
    dataSource: {
      type: 'file' | 'stream' | 'synthetic'
      uploadFile(): Promise<File>
      captureStream(duration: number): Promise<LogData>
      generateSynthetic(template: Template): LogData
    }
    
    filterEditor: {
      syntax: 'grok' | 'regex' | 'kql' | 'json'
      autocomplete: boolean
      validation: boolean
      templates: FilterTemplate[]
      history: FilterHistory[]
    }
    
    preview: {
      mode: 'split' | 'diff' | 'overlay'
      beforePanel: LogViewer
      afterPanel: LogViewer
      statistics: {
        matched: number
        dropped: number
        modified: number
        processingTime: number
      }
    }
    
    library: {
      savedFilters: SavedFilter[]
      testDatasets: TestDataset[]
      shareFilter(): ShareLink
      exportFilter(): ConfigYAML
    }
  }
}
```

#### Sandbox Configuration
```yaml
sandbox:
  enabled: true
  max_file_size: 100MB
  max_processing_time: 30s
  storage:
    type: local  # or azure_blob
    path: /var/lib/bibbl/sandbox
    retention_days: 30
  limits:
    max_files_per_user: 100
    max_filters_per_user: 500
    max_concurrent_tests: 10
  sample_capture:
    enabled: true
    max_duration: 60s
    max_events: 10000
  synthetic_data:
    templates:
      - syslog
      - windows_event
      - json_structured
      - cef_security
      - apache_access
```

### 11. File Library System

#### File Management Features
- **File Operations**:
  - Upload (drag & drop, multi-file)
  - Download (individual or bulk)
  - Delete with confirmation
  - Rename and organize
  - Version control for filters
- **File Organization**:
  - Folders/categories
  - Tagging system
  - Search and filter
  - Favorites/bookmarks
  - Sharing with team members
- **Supported Formats**:
  - Raw log files (.log, .txt)
  - JSON/JSONL
  - CSV/TSV
  - XML
  - Binary formats (EVTX for Windows)
  - Compressed files (.gz, .zip)

#### File Library API
```typescript
interface FileLibrary {
  // File operations
  upload(files: File[]): Promise<UploadResult>
  download(fileIds: string[]): Promise<Blob>
  delete(fileIds: string[]): Promise<void>
  rename(fileId: string, newName: string): Promise<void>
  
  // Organization
  createFolder(name: string, parentId?: string): Promise<Folder>
  moveFiles(fileIds: string[], folderId: string): Promise<void>
  tagFiles(fileIds: string[], tags: string[]): Promise<void>
  
  // Search and filter
  search(query: SearchQuery): Promise<SearchResult>
  getRecent(limit: number): Promise<File[]>
  getFavorites(): Promise<File[]>
  
  // Sharing
  share(fileId: string, users: string[]): Promise<ShareLink>
  getSharedWithMe(): Promise<File[]>
  
  // Metadata
  getFileInfo(fileId: string): Promise<FileMetadata>
  getFilePreview(fileId: string, lines: number): Promise<string>
}

interface FileMetadata {
  id: string
  name: string
  size: number
  type: string
  created: Date
  modified: Date
  owner: string
  tags: string[]
  folder: string
  shared: boolean
  checksum: string
  lineCount?: number
  encoding?: string
}
```

#### File Storage Configuration
```yaml
file_library:
  storage:
    type: hybrid  # local + azure_blob
    local:
      path: /var/lib/bibbl/files
      max_size: 10GB
    azure:
      container: bibbl-files
      tier: cool  # hot, cool, archive
    deduplication: true
    compression: true
    encryption:
      enabled: true
      key_source: azure_keyvault
  limits:
    max_file_size: 1GB
    max_total_size_per_user: 50GB
    allowed_extensions:
      - .log
      - .txt
      - .json
      - .jsonl
      - .csv
      - .xml
      - .evtx
      - .gz
      - .zip
  retention:
    default_days: 90
    sandbox_files: 30
    shared_files: 180
  antivirus:
    enabled: true
    engine: windows_defender  # or clamav
    scan_on_upload: true
```

### 12. Development Guidelines

#### Code Structure
```
cmd/bibbl/          - Main executable
internal/
  pipeline/         - Core pipeline engine
  inputs/           - Input plugins
    syslog/
    windows_event/  - Windows build tag
    systemd/        - Linux build tag
    azure_eventhub/
  processors/       - Processing plugins
    cef/
    kql/
  outputs/          - Output plugins
    sentinel/       - Sentinel Data Lake implementation
      dcr/          - Data Collection Rules client
      dce/          - Data Collection Endpoints client
      ingestion/    - Azure Monitor Ingestion API
      schema/       - Schema mapping and validation
    azure/          - Azure common utilities
  api/              - REST API server
    routes/
      auth/         - SSO authentication endpoints
      azure/        - Azure management endpoints
      templates/    - ARM template endpoints
  web/              - Embedded web UI (using embed.FS)
    static/         - React build output
    templates/      - HTML templates
  auth/             - Authentication providers
    entra/          - Entra ID SSO implementation
    local/          - Local auth fallback
  azure/            - Azure management
    arm/            - ARM template builder
    deployment/     - Resource deployment
    storage/        - Blob storage buffer
  cluster/          - Node coordination
  config/           - Configuration management
  metrics/          - Telemetry collection
  platform/         - Platform-specific code
    windows/        - Windows-specific implementations
    linux/          - Linux-specific implementations
pkg/
  queue/            - Persistent queue implementation
  buffer/           - Memory buffer pools
    azure/          - Azure blob buffer implementation
  tls/              - TLS utilities
  azure/            - Azure SDK wrappers
```

#### Technology Stack
- **Language**: Go 1.21+
- **Azure SDK**: Azure SDK for Go v2
- **Web Framework**: Fiber or Gin for API
- **UI**: React with TypeScript (embedded via embed.FS)
- **Authentication**: golang-jwt/jwt for tokens
- **OAuth**: golang.org/x/oauth2
- **Queue**: BadgerDB or BoltDB for local persistence
- **Azure Storage**: Azure Blob SDK for buffering
- **Metrics**: Prometheus client library + Azure Monitor exporter
- **Configuration**: Viper for config management
- **Build Tool**: Makefile with cross-compilation support

### 13. Testing Requirements
- Unit tests for all components (target 80% coverage)
- Integration tests for Sentinel ingestion
- Azure ARM deployment testing
- SSO authentication flow testing
- Buffer failover testing
- Cross-platform testing (Windows and Linux)
- Load testing for ingestion rate limits
- Cost estimation testing for Azure ingestion

### 14. Build & Deployment

#### Build System
```makefile
# Makefile targets
all: windows linux

windows:
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o bibbl-stream.exe cmd/bibbl/main.go

linux:
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o bibbl-stream cmd/bibbl/main.go

linux-arm:
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o bibbl-stream-arm64 cmd/bibbl/main.go

web:
    cd internal/web && npm run build
    go generate ./...

docker:
    docker build -t bibbl-stream:latest .

release: web windows linux linux-arm
    tar -czf bibbl-stream-linux-amd64.tar.gz bibbl-stream
    tar -czf bibbl-stream-linux-arm64.tar.gz bibbl-stream-arm64
    zip bibbl-stream-windows-amd64.zip bibbl-stream.exe
```

#### Deployment Options

##### Windows
- Windows Service installation via `sc.exe` or custom installer
- PowerShell deployment scripts
- Group Policy deployment
- SCCM/Intune package

##### Linux
- Systemd service files
- RPM/DEB packages
- Snap package
- Docker container
- Kubernetes DaemonSet

#### Installation Scripts
```bash
# Linux installation
sudo ./bibbl-stream install --service
sudo systemctl enable bibbl-stream
sudo systemctl start bibbl-stream

# Windows installation (PowerShell as Admin)
.\bibbl-stream.exe install --service
Start-Service BibblStream

# Initial setup with Azure
./bibbl-stream setup --azure --interactive
```

### 15. Implementation Priorities

#### Phase 1 (MVP) - Core + Azure Setup
1. Core pipeline engine (cross-platform)
2. Platform abstraction layer
3. **Entra ID SSO authentication with MFA**
4. **Secure session management**
5. **Azure Setup Wizard UI**
6. **ARM template builder and deployment**
7. **Azure Blob buffer implementation**
8. **Basic Prometheus metrics**
9. Windows Event Log input (Windows)
10. Systemd journal input (Linux)
11. Syslog input with CEF parsing
12. **Microsoft Sentinel Data Lake output with DCE/DCR**
13. Basic web portal with authentication
14. **Filter sandbox with basic before/after preview**
15. Single binary builds for Windows and Linux

#### Phase 2 - Enhanced Azure Integration & Monitoring
1. Advanced ARM template customization
2. Multi-tenant support
3. **Full Grafana integration with dashboards**
4. **Advanced Prometheus metrics and alerts**
5. **Complete file library system**
6. **Advanced sandbox with live stream capture**
7. Azure Event Hub input/output
8. Advanced KQL transformations
9. Pipeline configuration UI
10. Azure Key Vault integration
11. Platform-specific package builds (MSI, RPM, DEB)
12. Role-based access control
13. Audit logging to Sentinel

#### Phase 3 - Enterprise Features
1. Additional inputs/outputs (Elasticsearch, Splunk)
2. Advanced processors (Grok, aggregation)
3. **Synthetic data generation for sandbox**
4. **Filter template marketplace**
5. **Advanced Grafana dashboard customization**
6. Clustering support
7. Full web-based management
8. Advanced Defender XDR integration
9. ARM64 support for cloud deployments
10. Multi-region deployment support

## Success Criteria
- Single codebase compiles to both Windows and Linux binaries
- Zero-touch Azure deployment via ARM templates
- SSO authentication working with Entra ID
- Reliable buffering with Azure Blob Storage
- Successfully ingest 100% of security logs into Sentinel Data Lake
- Reduce Cribl licensing costs by 100% ($100k/year savings)
- Maintain compliance with Microsoft security requirements
- Achieve 5x better performance than Cribl/Logstash
- Provide seamless integration with Defender XDR portal

## Notes for Development

### Secure Login Implementation
- Implement PKCE for OAuth flows
- Use secure session storage (Redis or encrypted cookies)
- Implement rate limiting on authentication endpoints
- Add CAPTCHA after failed login attempts
- Log all authentication events for audit
- Support passwordless authentication (FIDO2/WebAuthn)
- Implement secure password reset flow
- Add admin impersonation with audit trail

### Grafana/Prometheus Integration
- Embed Grafana using iframe with authentication proxy
- Pre-provision dashboards via ConfigMaps
- Implement custom Prometheus exporters for business metrics
- Use Prometheus Alertmanager for alert routing
- Support PromQL in filter expressions
- Export metrics to Azure Monitor for long-term storage
- Implement SLI/SLO tracking dashboards
- Add cost tracking metrics for Azure services

### Sandbox Implementation
- Use WebAssembly for client-side filter processing (optional)
- Implement diff algorithm for before/after comparison
- Support regex, Grok, and KQL filter syntaxes
- Add filter performance profiling
- Implement filter versioning with Git-like history
- Support collaborative filter development
- Add filter unit testing framework
- Export filters as reusable modules

### File Library Implementation
- Use MinIO for S3-compatible storage abstraction
- Implement content-based deduplication
- Add virus scanning on upload
- Support chunked uploads for large files
- Implement file preview generation
- Add full-text search using Bleve or similar
- Support file expiration policies
- Implement quota management per user/team