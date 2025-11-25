# Production Readiness Gap Matrix

Status legend: [Complete] ready, [Partial] in progress, [Missing] not started

## 1. Operational Robustness (Partial)

- Evidence: `cmd/bibbl/main.go` wires graceful shutdown (`signal.NotifyContext` + `srv.ShutdownWithContext`), CLI flags expose `--health`, `--diagnostics`, `--print-effective-config`; `internal/api/server.go` exposes `/api/v1/health`, `/live`, `/ready`, `/api/v1/healthz` and panic recovery middleware.
- Gaps: No startup dependency probe (Azure, storage, outputs) before serving traffic, readiness endpoint always returns 200 even when outputs unavailable, worker-pool sizing (`pkg/pipeline/worker_pool.go`) unused, and shutdown path doesn't flush inputs/outputs.

## 2. Configuration & Secrets (Partial)

- Evidence: `internal/config/config.go` implements `Config.Validate`, env overrides, TLS auto-cert defaults; `cmd/bibbl/main.go` supports `--config` + `--print-effective-config`; `internal/diagnostics/diagnostics.go` redacts secrets in summaries.
- Gaps: No documented env override matrix, no Azure Key Vault / managed identity fetch path (`internal/azure/config` is stubs), exported configs can still include plaintext secrets, and no cache/rotation policy for secrets.

## 3. Security & Hardening (Partial)

- Evidence: Security headers, request size limits, and rate limiting configured in `internal/api/server.go`; TLS helpers in `pkg/tls`; gosec report captured in `gosec-report.json`.
- Gaps: `internal/middleware/csrf.go` exists but unused; no gzip/Brotli middleware, cookies not used with Secure/HttpOnly/SameSite, dependency scanning isn't enforced in CI (no workflows), and gosec/grype results aren't wired into automation.

## 4. Authentication & Authorization (Missing)

- Evidence: Static bearer token auth plus minimal RBAC token map in `internal/api/server.go`.
- Gaps: Missing Entra ID/OAuth2 PKCE, MFA/WebAuthn, short-lived & refresh tokens, brute-force/anomaly detection, and audit logging for auth events (`internal/api/auth*.go` only handles tokens for synthetic tests).

## 5. Observability (Partial)

- Evidence: Structured logging via `internal/platform/logger/logger.go`; Prometheus metrics in `internal/metrics/metrics.go`; tracing stubs via `go.opentelemetry.io/otel` in `internal/api/server.go`.
- Gaps: No OTLP exporter wiring, no per-plugin RED metrics (helpers unused beyond HTTP), no pipeline stage timing, log level reload missing, log sampling not implemented, and audit logs share app log sink.

## 6. Pipeline Resilience (Partial)

- Evidence: Syslog batching (`internal/inputs/syslog/batch_handler.go`), retry logic in `internal/outputs/azureloganalytics` (basic), circular buffer in `pkg/buffer/ringbuffer.go`, circuit breaker abstraction in `pkg/pipeline/circuit_breaker.go`.
- Gaps: Worker pool/backpressure not wired, circuit breaker unused, no dead-letter queues, no spill-to-disk/Azure Blob buffering, no idempotency/dedup, and shutdown doesn't drain buffers.

## 7. Performance & Load (Partial)

- Evidence: Benchmarks in `internal/api/engine_memory_bench_test.go` and docs `PERFORMANCE-OPTIMIZATION.md`; synthetic metrics generator (`internal/inputs/synthetic`).
- Gaps: No k6/vegeta scripts, no automated profiling or zero-copy strategies beyond TODOs, limited object pooling, and load/cost tests not automated.

## 8. Testing Strategy (Partial)

- Evidence: Unit tests across config, pagination, filters, rate limiting (`*_test.go` under `internal/api`, `pkg/filters`, `pkg/pipeline`).
- Gaps: No documented coverage metrics, no integration or failover tests, no golden/fuzz/chaos suites, and no automated regression tests for headers/auth.

## 9. Build, Release & Supply Chain (Partial)

- Evidence: `Makefile` builds linux/windows/web and embeds ldflags; Docker artifacts exist; `build.ps1`/`start.ps1` scripts help packaging.
- Gaps: No reproducible build flags (`-trimpath`), no signing (cosign/Authenticode), no SBOM/provenance generation, Go proxy verification absent, and no release checklist or CHANGELOG automation.

## 10. Runtime Safety (Partial)

- Evidence: Panic recovery middleware (`internal/api/server.go`), HTTP timeouts, TLS min-version enforcement, coarse token-bucket limiter, Azure client timeouts in `internal/outputs/azureloganalytics`.
- Gaps: TLS verification disabled in CLI health check, no per-IP/per-user throttling, context propagation stops at HTTP layer, no memory/FD metrics, temp files from enrichment uploads (`internal/api/routes_api.go`) lack secure deletion, and no spill file hygiene.

## 11. Web UI (Partial)

- Evidence: Embedded React app (`internal/web/embed.go`, `internal/web/src/*`), CSP configurable via config.
- Gaps: Default CSP uses `'unsafe-inline'`, CSRF middleware unused, dev artifacts (source maps) always built, no localization/accessibility baseline, no SRI for external assets (if added), and no UI telemetry pipeline.

## 12. Data Governance & Compliance (Partial)

- Evidence: PII redactor in `pkg/filters/pii_redactor.go`; sandbox/library endpoints manage uploaded data.
- Gaps: PII filters not wired into pipelines by default, no retention/expiration policies for local buffers or uploads, audit logs not immutable, no encryption-at-rest validation, and no key rotation procedures.

## 13. Azure Integration (Partial)

- Evidence: Auth manager (`internal/azure/auth/manager.go`), Azure Log Analytics output plugin (`internal/outputs/azureloganalytics`), docs like `AZURE-LOG-ANALYTICS-INTEGRATION.md`.
- Gaps: No DCR/DCE lifecycle tooling, no managed identity path, no ARM/Bicep CI validation, no Sentinel throughput throttling logic, and cost metrics not emitted.

## 14. CLI & UX (Partial)

- Evidence: `cmd/bibbl/main.go` supports `--version`, `--diagnostics`, `--health`, `--print-effective-config`; diagnostics collector (`internal/diagnostics/diagnostics.go`) handles env summaries; helper tool `cmd/toolsyslogtls/main.go`.
- Gaps: No shell completion scripts, errors lack remediation hints, diagnostics output not bundled/exportable, and CLI lacks interactive prompts for setup.

## 15. Documentation (Partial)

- Evidence: Numerous targeted guides (`PERFORMANCE-OPTIMIZATION.md`, `SEVERITY-ROUTING-SETUP.md`, `VERSA-*` docs, `WORLD-CLASS-FEATURES.md`).
- Gaps: No architecture diagram, plugin dev guide, operational runbook, security model, or upgrade/migration guide; checklist in `.github/copilot-instructions.md` is the only consolidated reference.

## 16. Migration & Backward Compatibility (Missing)

- No config schema versioning, migration tooling, deprecation warnings, or fallback behavior across `internal/config` or docs.

## 17. SRE & Alerting (Missing)

- No default alert rules, runbook links, or synthetic canary pipelines. Metrics exist but nothing provisions alerting configs.

## 18. Quality Gates in CI (Missing)

- `.github/workflows/` is absent, so linting, tests, coverage, gosec/grype, SBOM, provenance, and race detector enforcement do not run.

## 19. Example Artifacts (Partial)

- Evidence: Graceful shutdown scaffold (`cmd/bibbl/main.go`), config validation (`internal/config/config.go`), security headers + rate limiting + panic recovery (same server file), circuit breaker abstraction (`pkg/pipeline/circuit_breaker.go`), tracing helper stubs.
- Gaps: Circuit breaker, tracing exporter, spill/replay buffer, advanced rate limiting, and RBAC registry aren't wired end-to-end.

## 20. Open Questions / Decisions (Outstanding)

- Items listed in `.github/copilot-instructions.md` (log format, queue sizing, tracing sampling, SLAs, spill encryption) remain undecided; no ADRs or design notes in `docs/` directory.

## Critical Unknowns & Follow-Ups

1. **Pipeline wiring**: `internal/api/engine_memory.go` mocks pipeline behavior, but it's unclear where real outputs (Azure, S3) connect. Need confirmation before adding resilience features.
2. **Secret management direction**: No code references Key Vault or managed identity; clarify preferred approach before implementing fetch/cache logic.
3. **Observability exports**: OTLP exporter package absent; decide on collector endpoint and sampling defaults before coding.
4. **CI expectations**: No workflows to inspect; need confirmation on target platforms and required stages to scaffold automation.
5. **Documentation owners**: Architectural/runbook artifacts need designated owners to keep future updates consistent.
