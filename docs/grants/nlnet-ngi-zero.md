# NLnet NGI Zero Commons Fund Application

| Field | Value |
|-------|-------|
| Fund | NGI Zero Commons Fund |
| URL | https://nlnet.nl/propose/ |
| Deadline | 2026-04-01 |
| Requested amount | EUR 30,000 |
| Project name | AI-Model-Card-Generator (mcg) |
| License | Apache 2.0 |
| Repository | https://github.com/ogulcanaydogan/AI-Model-Card-Generator |
| Applicant | Ogulcan Aydogan |
| Duration | 6 months |

---

## Abstract

AI-Model-Card-Generator (`mcg`) is an open-source Go CLI pipeline that pulls model metadata from HuggingFace, Weights & Biases, MLflow, and custom JSON, then runs fairness analysis, checks EU AI Act compliance, and exports finished model cards to Markdown, JSON, HTML, and PDF. The tool already ships 4,777 lines of Go across 4 releases (v1.0.0 through v1.2.0), 18 test files, 3 CI workflows, and a Next.js web UI skeleton with i18n support.

This grant will fund six months of focused work: deeper regulatory mapping (EU AI Act Article-level cross-references and NIST AI RMF subcategory coverage), a public plugin API so third-party extractors and checkers can register without forking, batch-mode scaling tests, and an interactive web dashboard that replaces the current skeleton. The result is a single binary practitioners can run against any experiment tracker to get a standards-ready model card in seconds. We'll also add carbon footprint reporting tied to CodeCarbon CSV output, making environmental cost visible by default. Every milestone produces a tagged release, documentation, and passing CI so adopters can track progress commit by commit.

---

## Description of Work

### Background

Model cards, first proposed by Mitchell et al. (2019), are the closest thing ML has to a nutrition label. The EU AI Act (entered into force August 2024) makes documentation of high-risk AI systems a legal requirement, not just a best practice. In practice, most teams skip model cards because writing them is tedious, the metadata lives in three or four different systems, and there's no single tool that ties extraction, analysis, and compliance together.

Several partial solutions exist (Hugging Face's `modelcard` Python library, Google's Model Card Toolkit, Fiddler's reporting layer), but none of them cover the full loop: pull from multiple trackers, run fairness metrics, check against a specific regulation, and export to a format a compliance officer can actually read. That gap is what `mcg` fills.

### Current State

`mcg` today is a working, tested CLI with an HTTP API layer. Here's what's already shipped:

- **4 extraction sources**: HuggingFace Hub API, Weights & Biases REST API, MLflow tracking API, local custom JSON
- **5 API endpoints**: `/generate`, `/validate`, `/check`, `/healthz`, `/readyz`
- **4 export formats**: Markdown, JSON, HTML, PDF (Chromium-based)
- **EU AI Act checker**: advisory and strict modes, deterministic pass/warn/fail scoring
- **NIST AI RMF mapping**: GOVERN, MAP, MEASURE, MANAGE function coverage with evidence markers
- **Carbon estimation**: Python bridge to CodeCarbon-compatible CSV, fixture mode for CI
- **Batch processing**: YAML manifest, parallel workers, per-job artifacts, summary report
- **Custom templates**: `template init|validate|preview` CLI and web API
- **API security**: static key auth (`X-API-Key`), per-IP rate limiting, structured JSON request logs, request ID tracing
- **Audit trail**: append-only JSONL at `artifacts/audit/runs.jsonl`
- **Web UI skeleton**: Next.js with `/en` and `/tr` i18n, source parity, compliance tabs
- **CI**: `ci.yml` (tests + builds), `release.yml` (tagged binary releases), `scorecard.yml` (OpenSSF)
- **18 Go test files**, 4,777 Go LOC, Apache 2.0 license

The project has real functionality but limited adoption, incomplete regulatory depth, and a web UI that's still a skeleton. It works well for a single developer; it doesn't yet work well for a team or an organization.

### Proposed Milestones

The grant funds six milestones over six months. Each milestone ends with a tagged release.

**M1: Plugin architecture and extractor SDK (months 1-2)**

Build a Go plugin interface so external extractors and compliance checkers can register at runtime. Ship an `ExtractorPlugin` and `CheckerPlugin` interface, a registration mechanism, and documentation showing how to write a third-party extractor. Migrate the existing HuggingFace, W&B, MLflow, and custom extractors to use this interface internally so it's proven before we ask others to adopt it.

Trade-off: Go's plugin system has platform constraints (Linux/macOS only, same Go version). We'll use a shared-library approach first and evaluate HashiCorp's go-plugin (gRPC-based) if cross-platform demand materializes.

Deliverables: `pkg/plugin/` package, migration of 4 extractors, contributor guide, v1.3.0 release.

**M2: Regulatory depth and Article-level mapping (month 2-3)**

Expand the EU AI Act checker from the current advisory heuristics to Article-level cross-references (Articles 9, 11, 13, 14 for high-risk systems). Add NIST AI RMF subcategory mappings beyond the current function-level coverage. Each check will cite the specific Article or subcategory, the evidence field it evaluated, and whether the finding is required or advisory.

Deliverables: updated `pkg/compliance/`, Article-level test fixtures, NIST subcategory matrix, v1.4.0 release.

**M3: Interactive web dashboard (months 3-4)**

Replace the Next.js skeleton with a functional dashboard: source selection, real-time generation progress, compliance result visualization (pass/warn/fail per Article), template editing, and export downloads. Support English and Turkish (existing i18n). Connect to the `mcg serve` API backend.

Trade-off: we'll ship server-side rendering for the initial version rather than a full SPA, keeping deployment simple at the cost of some interactivity.

Deliverables: `web/` rewrite, Playwright e2e tests, Docker Compose for local dev, v1.5.0 release.

**M4: Batch scaling and organizational workflows (month 4-5)**

Harden batch mode for 100+ model runs: connection pooling, retry with backoff, progress streaming, and a summary dashboard. Add role-based audit entries (operator identity, approval status) to the JSONL trail. Test with real W&B and MLflow instances at scale.

Deliverables: batch benchmarks, updated audit schema, organizational deployment guide, v1.6.0 release.

**M5: Carbon and environmental reporting (month 5)**

Expand the current carbon bridge from a single-number estimate to a structured report: training energy (kWh), CO2 equivalent by region, hardware utilization, and comparison baselines. Integrate with CodeCarbon CSV and cloud provider billing APIs (AWS, GCP) where available. Surface results in both the CLI output and the web dashboard.

Deliverables: `pkg/analyzers/carbon/` expansion, region-aware calculations, web carbon tab, v1.7.0 release.

**M6: Documentation, outreach, and ecosystem integration (month 6)**

Write a user guide, API reference, and contributor handbook. Publish to Go package index. Present at an open-source ML meetup or conference. Submit the tool to the EU AI Act compliance tooling registries. Release v2.0.0.

Deliverables: docs site, Go package listing, conference submission, v2.0.0 release.

---

## Budget

| Milestone | Description | Amount (EUR) |
|-----------|-------------|--------------|
| M1 | Plugin architecture and extractor SDK | 6,000 |
| M2 | Regulatory depth and Article-level mapping | 5,000 |
| M3 | Interactive web dashboard | 6,000 |
| M4 | Batch scaling and organizational workflows | 5,000 |
| M5 | Carbon and environmental reporting | 4,000 |
| M6 | Documentation, outreach, integration | 4,000 |
| **Total** | | **EUR 30,000** |

All amounts cover developer time (single contributor). No hardware costs; CI runs on GitHub Actions free tier. Travel for M6 conference participation is included in the M6 line.

---

## Milestones and Timeline

| Month | Milestone | Tagged Release | Key Deliverable |
|-------|-----------|---------------|-----------------|
| 1-2 | M1: Plugin architecture | v1.3.0 | `pkg/plugin/`, 4 migrated extractors |
| 2-3 | M2: Regulatory depth | v1.4.0 | Article-level EU AI Act, NIST subcategories |
| 3-4 | M3: Web dashboard | v1.5.0 | Functional Next.js dashboard, e2e tests |
| 4-5 | M4: Batch scaling | v1.6.0 | 100+ model benchmarks, audit schema v2 |
| 5 | M5: Carbon reporting | v1.7.0 | Structured carbon reports, region-aware |
| 6 | M6: Docs and outreach | v2.0.0 | Docs site, Go package listing, talk |

Start date: upon grant agreement signing (estimated May 2026).

---

## NGI Relevance

`mcg` directly serves the NGI Zero Commons Fund's goals of strengthening the open internet through trustworthy technology:

**Open-source AI transparency.** Model cards are the primary mechanism for making AI systems inspectable by non-experts. By automating their creation from existing experiment trackers, `mcg` removes the friction that keeps most models undocumented. Every output is machine-readable (JSON) and human-readable (Markdown, HTML, PDF), so the same artifact serves developers, auditors, and regulators.

**EU AI Act readiness.** The Act requires documentation for high-risk AI systems, but few open-source tools help practitioners meet those requirements. `mcg` doesn't replace legal counsel, but it flags gaps early and gives teams a structured starting point. Article-level mapping (M2) will make this specific enough to be useful in real compliance workflows.

**Interoperability and open standards.** The tool reads from 4 different experiment trackers and writes to 4 output formats. The plugin API (M1) will let anyone add new sources without forking. This reduces vendor lock-in: you can switch from W&B to MLflow and still get the same model card.

**Privacy and user control.** `mcg` runs locally by default. No data leaves the user's machine unless they choose to connect to a remote tracker. The API server mode uses static key auth and rate limiting, not cloud-hosted SaaS.

---

## Comparable Projects

| Project | Scope | Difference from mcg |
|---------|-------|---------------------|
| HuggingFace `modelcard` | Python library for HF-hosted model cards | Single source (HF only), no compliance checks, no multi-format export |
| Google Model Card Toolkit | Python toolkit for structured model cards | Tied to TFX/ML Metadata, no EU AI Act mapping, no CLI pipeline |
| Fiddler AI | Commercial model monitoring with card features | Proprietary, SaaS-only, no self-hosted option |
| VerifyML | Python library for model documentation | Limited to documentation, no extraction from trackers, no regulatory mapping |
| IBM AI FactSheets | Enterprise model documentation | Closed source, tied to IBM Cloud Pak |

`mcg` is the only open-source tool that covers the full loop: multi-source extraction, fairness analysis, regulatory compliance checking, and multi-format export, all from a single Go binary with no Python runtime dependency for core operations (Python is optional, used only for the fairness bridge and carbon estimation).

---

## Supporting Materials Checklist

Before submitting, confirm the following are accessible:

- [ ] Public GitHub repository: https://github.com/ogulcanaydogan/AI-Model-Card-Generator
- [ ] Apache 2.0 LICENSE file in repository root
- [ ] README with build instructions, CLI usage, and API documentation
- [ ] CHANGELOG.md with release history (v1.0.0 through v1.2.0)
- [ ] 3 CI workflows passing (ci.yml, release.yml, scorecard.yml)
- [ ] 18 Go test files with `go test ./...` passing
- [ ] Example artifacts in `examples/` directory
- [ ] Web UI skeleton in `web/` directory
- [ ] JSON schema at `schemas/model-card.v1.json`

---

## Submission Steps

1. Go to https://nlnet.nl/propose/
2. Select **NGI Zero Commons Fund**
3. Fill in the form fields using the header table and abstract above
4. For "Describe the work" use the Description of Work section (background + milestones)
5. For "Budget" enter EUR 30,000 and reference the budget table
6. For "Relevance" use the NGI Relevance section
7. For "Comparable efforts" use the Comparable Projects table
8. Attach or link to the GitHub repository
9. Submit before **2026-04-01**

---

*Last updated: 2026-03-13*
