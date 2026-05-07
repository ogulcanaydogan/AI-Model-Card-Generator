# Roadmap

## v1.3.0 — EU AI Act Annex IV Export (target: 2026-06-30)

- Full Annex IV technical documentation template covering all 9 required elements
- Strict-mode compliance flag: block export if any mandatory Annex IV field is empty
- Vertex AI and SageMaker extractor support (alongside existing HF, W&B, MLflow)
- PDF export via headless Chrome hardened for server-side rendering without X11

## v1.4.0 — Compliance Monitoring (target: 2026-08-31)

- Webhook endpoint: re-evaluate compliance when upstream model metadata changes
- Audit trail dashboard (`/audit` route) surfacing runs.jsonl with diff view
- NIST AI RMF 1.0 full coverage (currently partial) with per-function scoring
- i18n: French and German translations for EU deployment readiness

## v2.0.0 — Multi-Registry Support (target: 2026-Q4)

- Pull model metadata from OCI registries (Harbor, GHCR) in addition to ML platforms
- Team-level model card approval workflow with sign-off audit trail
- Signed model card artifact — attach attestation to model release via DSSE envelope
- Plugin API for custom extractors and custom compliance rulesets
