# AI-Model-Card-Generator

Automated model card generation for responsible AI and EU AI Act readiness.

## Highlights

- Go-first CLI pipeline (`mcg`) for extraction, analysis, compliance, and export.
- Phase 1 support for Hugging Face extraction.
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

Strict mode exits non-zero only when required gaps exist:

```bash
go run ./cmd/mcg-cli check \
  --framework eu-ai-act \
  --input artifacts/model_card.json \
  --strict true
```

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

## Roadmap

### Phase 1 (implemented baseline)

- HuggingFace extractor
- CLI (`generate`, `validate`, `check`)
- Performance + fairness analyzers
- EU AI Act compliance checker
- Markdown/JSON/HTML/PDF generators
- CI workflow

### Phase 2 (scaffolded)

- W&B and MLflow full integrations
- Carbon footprint estimator
- i18n and Next.js web UI
- NIST AI RMF mapping expansion

### Phase 3 (planned)

- Custom template builder
- Batch processing
- API server mode
- Audit trail and release hardening

## Legal Note

EU AI Act checks are advisory engineering guidance, not legal advice.
