# Changelog

All notable changes to this project are documented in this file.

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
