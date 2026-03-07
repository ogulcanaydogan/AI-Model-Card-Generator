# Web UI (Sprint 4 Skeleton)

This is a Next.js app-router skeleton for Sprint 4.

## What is included

- Locale-ready route structure (`/en`, `/tr`) with `en` default redirect from `/`.
- Generate form for `custom`, `hf`, `wandb`, `mlflow` sources.
- Template source selector (`built-in` or `template-file`) in generate flow.
- API route (`/api/generate`) that calls existing Go CLI:
  - `generate`
  - `validate`
  - `check --framework nist`
- Template API routes that call CLI template commands:
  - `POST /api/template/init`
  - `POST /api/template/validate`
  - `POST /api/template/preview`
- In-browser preview for:
  - Carbon section
  - NIST section (GOVERN/MAP/MEASURE/MANAGE breakdown + control coverage + short remediation)
  - Markdown output
  - Compliance tabs (`EU AI Act`, `NIST`, `ISO42001`)

## Run locally

```bash
cd web
npm install
npm run dev
```

Open:

- `http://localhost:3000/en`

## Notes

- API route defaults to deterministic fixtures:
  - `tests/fixtures/fairness_stub.py`
  - `tests/fixtures/carbon/carbon_fixture.json`
- `hf` flow can be pointed to a mock/base URL with `MCG_WEB_HF_BASE_URL`.
- `wandb/mlflow` live mode requires existing CLI environment variables unless fixtures are provided.
- Template API paths are restricted to repository-relative paths (path traversal and absolute paths are rejected).
