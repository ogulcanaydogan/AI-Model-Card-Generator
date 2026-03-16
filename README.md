# AI-Model-Card-Generator
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/12185/badge)](https://www.bestpractices.dev/projects/12185)

Automated model card generation for responsible AI and EU AI Act readiness.

## Highlights

- Go-first CLI pipeline (`mcg`) for extraction, analysis, compliance, and export.
- Phase 1 support for Hugging Face extraction.
- Phase 2 Sprint 1 support for W&B extraction (`entity/project/run_id`).
- Phase 2 Sprint 2 support for MLflow extraction (`run:<run_id>`).
- Phase 2 Sprint 3 support for Carbon estimate + NIST AI RMF checks.
- Phase 2 Sprint 4 web skeleton (`/en`, `/tr`) with Carbon + NIST preview.
- Phase 2 Sprint 4.1 web source parity (`custom|hf|wandb|mlflow`) and compliance tabs.
- v1.1.0 web template builder (path-based): `/api/template/init|validate|preview` + `templateFile` generate flow.
- v1.2.0 production hardening: static API key auth, rate limiting, structured request logs, and `GET /readyz`.
- Phase 3 API server mode (`mcg serve`) and audit trail (`artifacts/audit/runs.jsonl`).
- v1.0.1 custom template builder (CLI-first): `template init|validate|preview`, `--template-file`.
- Performance and fairness metrics from evaluation CSV.
- EU AI Act advisory compliance checks with strict mode option.
- Export formats: Markdown, JSON, HTML, PDF (Chromium-based).
- CI workflow covering Go tests, Python bridge checks, CLI integration, and binary builds.

## Repository Layout

```text
ai-model-card-generator/
├── README.md
├── LICENSE
├── cmd/
│   └── mcg-cli/
├── pkg/
│   ├── extractors/
│   ├── analyzers/
│   ├── generators/
│   ├── compliance/
│   ├── core/
│   └── templates/
├── templates/
├── schemas/
├── scripts/
├── examples/
├── web/
├── tests/
└── .github/workflows/
```

## Requirements

- Go `1.22+`
- Python `3.11+`
- Python deps for fairness bridge: `fairlearn`, `pandas`
- Chromium/Chrome installed for PDF export

Install Python dependencies:

```bash
python3 -m pip install fairlearn pandas
```

## CLI Usage

### Generate

```bash
go run ./cmd/mcg-cli generate \
  --model bert-base-uncased \
  --source hf \
  --template standard \
  --eval-file examples/eval_sample.csv \
  --formats md,json,pdf \
  --out-dir artifacts \
  --lang en \
  --compliance eu-ai-act
```

Custom template override:

```bash
go run ./cmd/mcg-cli generate \
  --model demo-model \
  --source custom \
  --uri tests/fixtures/custom_metadata.json \
  --template-file examples/templates/custom-v101.tmpl \
  --eval-file examples/eval_sample.csv \
  --formats md,json \
  --out-dir artifacts/custom-template
```

### Generate (W&B)

`--model` must be in this format: `<entity>/<project>/<run_id>`.

```bash
WANDB_API_KEY=your_api_key_here \
go run ./cmd/mcg-cli generate \
  --model entity/project/run_id \
  --source wandb \
  --template standard \
  --eval-file examples/eval_sample.csv \
  --formats md,json \
  --out-dir artifacts/wandb \
  --lang en \
  --compliance eu-ai-act
```

### Generate (Batch Manifest)

```bash
go run ./cmd/mcg-cli generate \
  --batch examples/batch_manifest.yaml \
  --workers 4 \
  --fail-fast false \
  --out-dir artifacts/batch
```

Batch outputs:
- Per-job artifacts default to `<out-dir>/<job-id>/`
- Summary report is written to `<out-dir>/batch_report.json`
- Exit code is non-zero when at least one job fails

### Template Builder (CLI)

Initialize from a built-in base:

```bash
go run ./cmd/mcg-cli template init \
  --name "My Custom Template" \
  --base standard \
  --out examples/templates/my-custom.tmpl
```

Validate template placeholders:

```bash
go run ./cmd/mcg-cli template validate \
  --input examples/templates/my-custom.tmpl
```

Preview template against a model card JSON:

```bash
go run ./cmd/mcg-cli template preview \
  --input examples/templates/my-custom.tmpl \
  --card artifacts/model_card.json \
  --out artifacts/template-preview.md
```

### Validate

```bash
go run ./cmd/mcg-cli validate \
  --schema schemas/model-card.v1.json \
  --input artifacts/model_card.json
```

### Check

```bash
go run ./cmd/mcg-cli check \
  --framework eu-ai-act \
  --input artifacts/model_card.json \
  --strict false
```

NIST check example:

```bash
go run ./cmd/mcg-cli check \
  --framework nist \
  --input artifacts/model_card.json \
  --strict false
```

NIST report coverage is grouped by `GOVERN`, `MAP`, `MEASURE`, `MANAGE`.
Status rule is deterministic:
- `fail`: at least one required gap
- `warn`: no required gaps and at least one advisory finding
- `pass`: no required gaps and no advisory findings

NIST mapping expansion principles:
- Each control has a stable control ID, evidence field, and requirement class (`required` / `advisory`).
- Checker messages include `[evidence:<field>]` markers for traceable remediation.
- Score is control-level weighted (required penalties are stronger than advisory penalties).

Strict mode exits non-zero only when required gaps exist:

```bash
go run ./cmd/mcg-cli check \
  --framework eu-ai-act \
  --input artifacts/model_card.json \
  --strict true
```

### Serve (HTTP API)

```bash
go run ./cmd/mcg-cli serve \
  --addr :8080 \
  --read-timeout 30s \
  --write-timeout 180s
```

Endpoints:
- `POST /generate`
- `POST /validate`
- `POST /check`
- `GET /healthz`
- `GET /readyz`

Protected endpoint headers:
- `X-API-Key` (required when `MCG_REQUIRE_AUTH=true`)
- `X-Request-ID` (optional, generated if not provided)

### W&B Environment Variables

- `WANDB_API_KEY` (required for live W&B extraction)
- `WANDB_BASE_URL` (optional, defaults to `https://api.wandb.ai`)
- `MCG_WANDB_FIXTURE` (optional, enables deterministic fixture mode for tests)

Example script:

- `examples/wandb-generate.sh`

### Generate (MLflow)

`--model` must be in this format: `run:<run_id>`.

```bash
MLFLOW_TRACKING_URI=http://localhost:5000 \
go run ./cmd/mcg-cli generate \
  --model run:abc123 \
  --source mlflow \
  --template standard \
  --eval-file examples/eval_sample.csv \
  --formats md,json \
  --out-dir artifacts/mlflow \
  --lang en \
  --compliance eu-ai-act
```

### MLflow Environment Variables

- `MLFLOW_TRACKING_URI` (required for live MLflow extraction)
- `MLFLOW_TRACKING_TOKEN` (optional bearer token auth)
- `MLFLOW_TRACKING_USERNAME` (optional basic auth username)
- `MLFLOW_TRACKING_PASSWORD` (optional basic auth password)
- `MCG_MLFLOW_FIXTURE` (optional, enables deterministic fixture mode for tests)

Example script:

- `examples/mlflow-generate.sh`

### Carbon Environment Variables

- `MCG_CARBON_FIXTURE` (optional, deterministic fixture mode for CI/tests)
- `MCG_CARBON_SCRIPT` (optional, defaults to `scripts/carbon_metrics.py`)
- `MCG_CARBON_PYTHON_BIN` (optional, defaults to `MCG_PYTHON_BIN` or `python3`)
- `MCG_CARBON_KG_CO2E` (optional manual value consumed by bridge script)
- `MCG_CARBON_EMISSIONS_FILE` (optional CodeCarbon-like CSV path with `emissions` column)

### Audit Environment Variables

- `MCG_AUDIT_PATH` (optional, default `artifacts/audit/runs.jsonl`)
- `MCG_OPERATOR` (optional, default `unknown`)

### Server Security Environment Variables

- `MCG_REQUIRE_AUTH` (optional, default `false`)
- `MCG_API_KEYS` (optional comma-separated keys; required when auth is enabled)
- `MCG_RATE_LIMIT_ENABLED` (optional, default `true`)
- `MCG_RATE_LIMIT_RPM` (optional, default `120`)
- `MCG_RATE_LIMIT_BURST` (optional, default `30`)
- `MCG_GENERATE_TIMEOUT` (optional, default `180s`)
- `MCG_VALIDATE_TIMEOUT` (optional, default `60s`)
- `MCG_CHECK_TIMEOUT` (optional, default `60s`)

## Eval CSV Contract

Required columns:

- `y_true`
- `y_pred`
- `group`

Optional columns:

- `y_score`
- `sample_weight`

## JSON Schema

Schema path:

- `schemas/model-card.v1.json`
- `schemas/batch-manifest.v1.yaml` (batch manifest reference contract)

Required top-level keys:

- `metadata`
- `performance`
- `fairness`
- `risk_assessment`
- `compliance`
- `version`

## Development

Run tests:

```bash
go test ./...
```

API integration tests:

```bash
go test ./tests/integration -run APIServer -v
```

Run integration tests with W&B fixture mode:

```bash
MCG_WANDB_FIXTURE=tests/fixtures/wandb/run_fixture.json go test ./tests/integration -v
```

Run integration tests with MLflow fixture mode:

```bash
MCG_MLFLOW_FIXTURE=tests/fixtures/mlflow/run_get_fixture.json go test ./tests/integration -v
```

Run integration tests with Carbon fixture mode:

```bash
MCG_CARBON_FIXTURE=tests/fixtures/carbon/carbon_fixture.json go test ./tests/integration -v
```

Run batch fixture integration tests:

```bash
MCG_WANDB_FIXTURE=tests/fixtures/wandb/run_fixture.json go test ./tests/integration -run Batch -v
```

Run template command integration tests:

```bash
go test ./tests/integration -run "Template|GenerateTemplateFile" -v
```

Run web UI (Sprint 4 skeleton):

```bash
cd web
npm install
npm run dev
```

Run web tests (Sprint 4.1):

```bash
cd web
npm run test:unit
npm run test:smoke
```

Web template endpoints (v1.1.0 path-based):

- `POST /api/template/init`
- `POST /api/template/validate`
- `POST /api/template/preview`

All template file paths are validated as repository-relative paths.

## Roadmap

### Phase 1 (implemented baseline)

- HuggingFace extractor
- CLI (`generate`, `validate`, `check`)
- Performance + fairness analyzers
- EU AI Act compliance checker
- Markdown/JSON/HTML/PDF generators
- CI workflow

### Phase 2 (implemented)

- W&B integration (implemented in Sprint 1)
- MLflow integration (implemented in Sprint 2)
- Carbon footprint estimator (implemented in Sprint 3)
- NIST AI RMF rule-based mapping (implemented in Sprint 3)
- i18n and Next.js web UI (Sprint 4 skeleton implemented)
- Web source parity and compliance tab UX (Sprint 4.1 implemented)
- Web template builder (v1.1.0, path-based) implemented
- NIST checker deepening + function-based compliance UX hardening (Sprint 4.2 implemented)
- NIST AI RMF mapping expansion (Phase 2.3 implemented)

### Phase 3 (implemented)

- Batch processing (Sprint 5 implemented)
- API server mode (Sprint 6 implemented)
- Audit trail and release hardening (Sprint 7 implemented)
- Custom template builder (v1.0.1 implemented, CLI-first)

## Legal Note

EU AI Act checks are advisory engineering guidance, not legal advice.
