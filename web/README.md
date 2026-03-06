# Web UI (Sprint 4 Skeleton)

This is a Next.js app-router skeleton for Sprint 4.

## What is included

- Locale-ready route structure (`/en`, `/tr`) with `en` default redirect from `/`.
- Generate form for `custom` source.
- API route (`/api/generate`) that calls existing Go CLI:
  - `generate`
  - `validate`
  - `check --framework nist`
- In-browser preview for:
  - Carbon section
  - NIST section
  - Markdown output

## Run locally

```bash
cd web
npm install
npm run dev
```

Open:

- `http://localhost:3000/en`

## Notes

- This sprint intentionally supports `custom` source first in the web flow.
- API route defaults to deterministic fixtures:
  - `tests/fixtures/fairness_stub.py`
  - `tests/fixtures/carbon/carbon_fixture.json`
