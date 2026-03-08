# Changelog

All notable changes to this project are documented in this file.

## [v1.2.0] - 2026-03-08

### Added
- API security middleware for `mcg serve`:
  - `X-API-Key` validation on protected routes (`POST /generate`, `POST /validate`, `POST /check`)
  - auth config via `MCG_REQUIRE_AUTH` and `MCG_API_KEYS`
- New readiness endpoint:
  - `GET /readyz` with dependency checks for audit logger, schema path, and pipeline wiring
- Rate limiting middleware:
  - per-IP and per-endpoint token-bucket limits
  - config via `MCG_RATE_LIMIT_ENABLED`, `MCG_RATE_LIMIT_RPM`, `MCG_RATE_LIMIT_BURST`
  - stable API error code `rate_limited` with HTTP 429
- Request tracing and structured logs:
  - `X-Request-ID` passthrough/generation
  - JSON request logs containing `request_id`, `route`, `status`, `latency_ms`, `client_ip`, and `error_code`

### Updated
- Timeout propagation for API handlers with endpoint-specific defaults (`generate`, `validate`, `check`).
- API error classification with new `unauthorized` and `rate_limited` codes.
- Integration and unit tests covering auth, rate limiting, request-id propagation, and readyz checks.
- CI integration job now explicitly runs API hardening integration coverage.

## [v1.1.0] - 2026-03-07

### Added
- Path-based web template builder API endpoints:
  - `POST /api/template/init`
  - `POST /api/template/validate`
  - `POST /api/template/preview`
- Web generate flow support for template file overrides:
  - `POST /api/generate` now accepts optional `templateFile` and `templateSource`
  - template file mode maps to CLI `--template-file`
- Repository-root path guard for web template operations:
  - absolute paths and traversal (`..`) are rejected with 400 responses
- Web test coverage for template builder parity:
  - unit tests for path guard and template payload validation
  - smoke test coverage for `init -> validate -> preview` chain and invalid path rejection

### Updated
- Web UI with template source toggle (`built-in` / `template-file`) and template actions panel (`init`, `validate`, `preview`).
- README and web README with v1.1.0 template endpoint usage notes.

## [v1.0.1] - 2026-03-07

### Added
- Custom template builder CLI commands:
  - `mcg template init --name <name> --out <path> --base <standard|minimal|eu-ai-act>`
  - `mcg template validate --input <template.tmpl>`
  - `mcg template preview --input <template.tmpl> --card <model_card.json> --out <preview.md>`
- `generate` template file override:
  - new flag `--template-file <path>`
  - precedence: `--template-file` overrides `--template`
- Batch manifest template override support:
  - optional `template_file` on defaults and per-job definitions
- Additional tests for template builder flow, pipeline template precedence, and batch template file behavior.

### Updated
- README with v1.0.1 template commands and usage examples.
- Batch manifest docs and examples to include `template_file`.

## [v1.0.0] - 2026-03-06

### Added
- Batch-first generation flow with manifest orchestration:
  - `mcg generate --batch <manifest.yaml> --workers <n> --fail-fast <true|false>`
  - deterministic `batch_report.json` output and continue-on-error defaults
- API server mode:
  - `mcg serve --addr :8080 --read-timeout 30s --write-timeout 180s`
  - `POST /generate`, `POST /validate`, `POST /check`, `GET /healthz`
- Mandatory append-only audit trail for CLI and API runs:
  - default path `artifacts/audit/runs.jsonl`
  - includes run id, input hash, status, duration, source/framework context, and artifact paths
- Hardening updates:
  - retry/backoff (`max 2` retries) for external extractors (`hf|wandb|mlflow`)
  - context timeout/cancel propagation
  - standardized API error codes: `invalid_input`, `unsupported_source`, `internal_error`, `compliance_failed`
- Release automation:
  - cross-platform binaries (Linux/macOS/Windows amd64)
  - SHA256 checksum generation
  - GitHub release workflow on tag push

### Updated
- README usage for batch, API server mode, and audit trail.
- Phase 3 roadmap status moved to completed.
