# Web UI (Sprint 4 Skeleton)

This is a Next.js app-router skeleton for Sprint 4.

## What is included

- Locale-ready route structure (`/en`, `/tr`) with `en` default redirect from `/`.
- Generate form for `custom`, `hf`, `wandb`, `mlflow` sources.
- API route (`/api/generate`) that calls existing Go CLI:
  - `generate`
  - `validate`
  - `check --framework nist`
- In-browser preview for:
  - Carbon section
  - NIST section
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
