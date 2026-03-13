# OpenSSF Best Practices Badge — Submission Guide

> **URL:** https://www.bestpractices.dev/
> **Time:** ~30 minutes
> **Cost:** Free
> **What you get:** CII Best Practices badge for README + credibility for grant applications

---

## Pre-Submission Checklist

| Criterion | Status | Evidence |
|-----------|--------|----------|
| OSS license (approved by OSI) | Apache 2.0 | `LICENSE` |
| Project website with basic info | GitHub README | `README.md` |
| Version control (public repo) | GitHub | `github.com/ogulcanaydogan/AI-Model-Card-Generator` |
| Unique version numbering | SemVer | `CHANGELOG.md` |
| Release notes | Per-version changelogs | `CHANGELOG.md` |
| Bug reporting process | GitHub Issues | Issues tab |
| Build system | Makefile + go build | `Makefile`, `go.mod` |
| Automated test suite | Go tests | `.github/workflows/ci.yml` |
| New functionality tested | CI enforces tests | CI pipeline |
| Warning flags enabled | golangci-lint | CI lint jobs |
| HTTPS for project sites | GitHub uses HTTPS | Default |
| English documentation | All docs in English | `docs/` |
| Vulnerability reporting process | Private disclosure policy | `SECURITY.md` |
| Working build system | CI green | `.github/workflows/ci.yml` |

---

## Step-by-Step Submission

### 1. Go to bestpractices.dev
Navigate to https://www.bestpractices.dev/ and click **"Get Your Badge Now"**.

### 2. Sign in with GitHub
Use your GitHub account (`ogulcanaydogan`).

### 3. Add your project
Enter the repository URL:
```
https://github.com/ogulcanaydogan/AI-Model-Card-Generator
```

### 4. Fill out the form

#### Basics
- **Project name:** AI-Model-Card-Generator (mcg)
- **Description:** Automated model card generation for responsible AI and EU AI Act readiness. Go-first CLI pipeline with HuggingFace, Weights & Biases, and MLflow extraction. Exports to Markdown, JSON, HTML, PDF.
- **Project URL:** https://github.com/ogulcanaydogan/AI-Model-Card-Generator
- **License:** Apache-2.0
- **Documentation URL:** https://github.com/ogulcanaydogan/AI-Model-Card-Generator/tree/main/docs

#### Change Control
- **Public version control:** Yes (GitHub)
- **Unique version numbering:** Yes (SemVer)
- **Release notes:** Yes (CHANGELOG.md)

#### Reporting
- **Bug reporting process:** Yes (GitHub Issues)
- **Vulnerability reporting process:** Yes → point to `SECURITY.md`

#### Quality
- **Working build system:** Yes (Makefile, go build)
- **Automated test suite:** Yes (Go tests, Node tests, Python tests)
- **New functionality testing policy:** Yes (CI enforces tests)
- **Test coverage:** Run `go test -cover ./...` for exact percentage

#### Security
- **Secure development knowledge:** Yes (NIST AI RMF integration)
- **Use basic good cryptographic practices:** N/A (not a crypto project)
- **Static analysis:** Yes (golangci-lint, go vet)

#### Analysis
- **Static analysis in CI:** Yes → golangci-lint
- **Dynamic analysis:** Yes → go test -race

### 5. Submit and add badge to README

After submission, add the badge:
```markdown
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/XXXXX/badge)](https://www.bestpractices.dev/projects/XXXXX)
```
