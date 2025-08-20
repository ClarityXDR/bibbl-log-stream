# Copilot Instructions for Bibbl Log Stream

## Project Overview
- **Single-binary, cross-platform log pipeline**: Written in Go, compiles to Windows and Linux executables. No Java/JVM dependencies.
- **Major components**: Inputs (Syslog, HTTP, Windows Event Log, Kafka, etc.), Processing (parsing, filtering, transformation, aggregation), Outputs (Microsoft Sentinel, Azure, Splunk, S3, etc.), Embedded Web UI (React/TypeScript), Azure automation (ARM templates), and File/Filter sandbox.
- **Key directories**: See `internal/` for pipeline, plugins, API, web UI, authentication, Azure integration, metrics, and platform-specific code. `cmd/bibbl/` is the main entrypoint.

## Build & Run
- **Cross-platform builds**: Use the provided Makefile. Example: `make windows`, `make linux`, `make web`, `make docker`.
- **Web UI**: Build with `make web` (runs React build in `internal/web/`).
- **Run as service**: Windows: `./bibbl-stream.exe install --service`; Linux: `sudo ./bibbl-stream install --service`.
- **Azure setup**: `./bibbl-stream setup --azure --interactive` for guided deployment.

## Testing
- **Unit tests**: Target 80%+ coverage. Use Go's standard testing tools.
- **Integration tests**: Focus on Sentinel ingestion, SSO auth, buffer failover, and cross-platform compatibility.
- **Load/cost tests**: Simulate high-volume ingestion and Azure cost scenarios.

## Patterns & Conventions
- **Plugin architecture**: Inputs, processors, and outputs are modular under `internal/`.
- **Config**: Use Viper for config management. YAML/JSON import/export supported in the UI.
- **Authentication**: SSO via Entra ID (OAuth2/OpenID), with fallback to local accounts. See `internal/auth/`.
- **Session management**: Secure cookies or Redis. Short-lived JWTs, refresh tokens, MFA enforced.
- **Azure integration**: ARM template builder, DCR/DCE management, and blob buffer in `internal/azure/`.
- **Metrics**: Prometheus client library, with Azure Monitor export. Custom business metrics supported.
- **Web UI**: Embedded via Go's embed.FS. React/TypeScript code in `internal/web/`.
- **Sandbox**: Filter development and test data management in `/sandbox` routes and UI.
- **File library**: S3-compatible (MinIO) and Azure Blob support, deduplication, antivirus, chunked uploads.

## Integration Points
- **Azure**: Uses Azure SDK for Go v2. ARM templates for resource deployment. DCR/DCE for Sentinel.
- **Grafana/Prometheus**: Embedded dashboards, custom exporters, alert routing.
- **Authentication**: OAuth2 PKCE, FIDO2/WebAuthn, rate limiting, CAPTCHA, audit logging.

## Examples
- **Add a new input plugin**: Place in `internal/inputs/`, register in pipeline engine.
- **Add a new output**: Place in `internal/outputs/`, follow existing output plugin structure.
- **Update web UI**: Edit React code in `internal/web/`, then `make web` to rebuild.
- **ARM template changes**: See `/azure/templates` in the portal and `internal/azure/arm/` for code.

## Special Notes
- **No Java/JVM**: All code must be pure Go or TypeScript/React for UI.
- **Single binary**: All features (including web UI) must be embedded in the main executable.
- **Security**: TLS everywhere, RBAC, audit logging, and Azure Key Vault integration are mandatory.

---

## Production Hardening & Readiness Checklist

### 1. Operational Robustness
- [ ] Add graceful shutdown (context, signal handling) in `cmd/bibbl/main.go`
- [ ] Uniform structured logging (zap/zerolog) with traceID, component, error fields
- [ ] Startup self-check (config validation + dependency reachability: Azure, storage, outputs)
- [ ] Health & readiness endpoints (liveness vs readiness separation)
- [ ] Configurable concurrency & worker pool sizing (CPU/IO adaptive)

### 2. Configuration & Secrets
- [ ] Central `Config.Validate()` with bounds, durations, enumerations
- [ ] `--config`, `--print-effective-config`, and redaction of secrets
- [ ] Environment override matrix documented
- [ ] Azure Key Vault / managed identity secret fetch with local caching
- [ ] Disallow plaintext secrets in exported configs

### 3. Security & Hardening
- [ ] HTTP security headers (HSTS, CSP, Referrer-Policy, Permissions-Policy, COOP/COEP, CORP)
- [ ] Request size limits & body streaming for large uploads
- [ ] GZIP/Brotli compression with MIME allowlist
- [ ] CSRF protection on state-changing routes
- [ ] Strict cookie attributes: Secure, HttpOnly, SameSite=Strict
- [ ] Dependency vulnerability scan in CI (grype) + SBOM (syft)
- [ ] Banned functions / unsafe patterns linting (gosec)

### 4. Authentication & Authorization
- [ ] Entra ID / OAuth2 PKCE flows validated
- [ ] MFA / WebAuthn enforcement path
- [ ] Short-lived access tokens; rotating refresh tokens
- [ ] RBAC middleware (route → required roles) deny-by-default
- [ ] Brute force, anomaly, geo / impossible travel detection
- [ ] Audit logging for auth events & privilege changes

### 5. Observability
- [ ] Prometheus RED metrics per plugin (requests, errors, duration)
- [ ] Pipeline stage timing (input→parse→filter→transform→output)
- [ ] OpenTelemetry traces exported (OTLP) with sampling controls
- [ ] Structured audit log sink separate from app logs
- [ ] Log level dynamic reload (SIGHUP / admin API)
- [ ] Log sampling for high-volume debug categories

### 6. Pipeline Resilience
- [ ] Bounded queues with backpressure & depth metrics
- [ ] Retry + exponential backoff + circuit breakers for outputs
- [ ] Dead letter / quarantine channel for malformed or persistently failing events
- [ ] Spill-to-disk / Azure Blob buffer for outage scenarios with forward replay
- [ ] Idempotency / deduplication where target requires
- [ ] Graceful drain on shutdown (flush buffers, cut new intake)

### 7. Performance & Load
- [ ] Benchmarks for hot paths (parsers, serialization)
- [ ] k6 / vegeta load test scenarios (steady, spike, soak)
- [ ] Memory profiling & object pooling for large allocations
- [ ] Zero-copy strategies (reuse buffers, avoid unnecessary JSON roundtrips)

### 8. Testing Strategy
- [ ] Unit coverage ≥80% (critical packages >90%)
- [ ] Integration test matrix (Windows/Linux, Sentinel ingestion, failover buffer)
- [ ] Golden file tests for parsers / transformations
- [ ] Fuzz tests for parsers & protocol decoders
- [ ] Chaos tests (induced latency, network partition, disk full)
- [ ] Automated regression tests for security headers & auth flows

### 9. Build, Release & Supply Chain
- [ ] Reproducible builds (`-trimpath`, embedded version via ldflags)
- [ ] Multi-arch builds (amd64, arm64) signed (cosign + Authenticode on Windows)
- [ ] SBOM included in release artifacts
- [ ] Provenance attestation (cosign SLSA provenance)
- [ ] Go module verification & private proxy caching
- [ ] Release checklist (CHANGELOG, migration notes, deprecation warnings)

### 10. Runtime Safety
- [ ] Panic recovery middleware increments metric & returns sanitized 500
- [ ] Defensive timeouts (server read/write, outbound HTTP, Azure SDK)
- [ ] Context propagation across pipeline stages
- [ ] Memory & file descriptor usage metrics / alerts
- [ ] Safe temp file handling & secure deletion of sensitive spill files
- [ ] Rate limiting (global + per-IP / per-user tiers)

### 11. Web UI
- [ ] CSP with nonce or hash for any inline scripts (no unsafe-inline)
- [ ] Remove dev artifacts (source maps optional behind auth toggle)
- [ ] Localization / accessibility baseline
- [ ] Front-end build integrity (subresource integrity hashes if external assets ever used)
- [ ] UI telemetry (page load, error boundaries) -> internal metrics

### 12. Data Governance & Compliance
- [ ] PII detection / redaction filters
- [ ] Configurable retention policies for local buffers & file library
- [ ] Audit log immutability / WORM option
- [ ] Encryption at rest validation (disk, blob)
- [ ] Key rotation procedure (Azure Key Vault + app reload)

### 13. Azure Integration
- [ ] ARM/Bicep templates validated (what-if) in CI
- [ ] DCR/DCE lifecycle idempotency checks
- [ ] Managed identity usage over client secrets where possible
- [ ] Sentinel ingestion throughput / throttling handling
- [ ] Cost monitoring metrics (events/sec by source, dropped events)

### 14. CLI & UX
- [ ] `--version` output (version, commit, build date, Go version)
- [ ] `diagnostics` command (env summary, effective config sans secrets)
- [ ] Shell completion scripts generation
- [ ] Helpful error messages with remediation hints

### 15. Documentation
- [ ] Architecture diagram (updated)
- [ ] Plugin development guide
- [ ] Operational runbook (alerts, common failures)
- [ ] Security model & threat assessment
- [ ] Upgrade / migration guide between minor versions

### 16. Migration & Backward Compatibility
- [ ] Config schema versioning + migration tool
- [ ] Deprecation warnings with sunset versions
- [ ] Fallback behavior for removed fields

### 17. SRE & Alerting
- [ ] Default alert rules (error rate, queue depth, dropped events, auth failures)
- [ ] Runbook links embedded in alert annotations
- [ ] Synthetic canary events path with validation

### 18. Quality Gates in CI
- [ ] Lint (golangci-lint) mandatory pass
- [ ] Unit + integration tests pass
- [ ] Coverage threshold enforced
- [ ] Vulnerability scan must be clean or explicitly waived
- [ ] SBOM + provenance generated
- [ ] Race detector run on critical packages

### 19. Examples / Code Artifacts To Implement
- [ ] Graceful shutdown scaffold in `cmd/bibbl/main.go`
- [ ] Config validation (`internal/config`)
- [ ] Security headers & rate limiting middleware
- [ ] Panic recovery & metrics middleware
- [ ] Output circuit breaker abstraction
- [ ] Spill/replay buffer implementation
- [ ] Tracing instrumentation wrappers
- [ ] RBAC middleware & role mapping registry

### 20. Open Questions / Decisions (Fill as resolved)
- [ ] Log format: JSON vs. text in interactive mode
- [ ] Default queue sizes per plugin category
- [ ] Sampling strategy for tracing (parent-based? 1%?)
- [ ] SLA / SLO targets (latency, success rate)
- [ ] Encryption approach for on-disk spill (AES-GCM + envelope?)

(Keep this checklist updated as items are completed.)
