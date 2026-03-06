package generators

import (
	"bytes"
	"html"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	"github.com/yuin/goldmark"
)

const defaultMarkdownTemplate = `# Model Card: {{ .Metadata.Name }}

## Metadata
- Owner: {{ .Metadata.Owner }}
- License: {{ .Metadata.License }}
- Tags: {{ range $i, $t := .Metadata.Tags }}{{ if $i }}, {{ end }}{{ $t }}{{ end }}
- Intended Use: {{ .Metadata.IntendedUse }}
- Limitations: {{ .Metadata.Limitations }}

## Performance
- Accuracy: {{ printf "%.4f" .Performance.Accuracy }}
- Precision: {{ printf "%.4f" .Performance.Precision }}
- Recall: {{ printf "%.4f" .Performance.Recall }}
- F1: {{ printf "%.4f" .Performance.F1 }}
- AUC: {{ printf "%.4f" .Performance.AUC }}

## Fairness
- Demographic Parity Difference: {{ printf "%.4f" .Fairness.DemographicParityDiff }}
- Equalized Odds Difference: {{ printf "%.4f" .Fairness.EqualizedOddsDiff }}

## Carbon / Sustainability
{{ if .Carbon }}- Estimated kgCO2e: {{ printf "%.6f" .Carbon.EstimatedKgCO2e }}
- Method: {{ .Carbon.Method }}
{{ else }}- Estimated kgCO2e: n/a
- Method: unavailable
{{ end }}

## Risk Assessment
{{ range .RiskAssessment.KnownRisks }}- {{ . }}
{{ end }}

## Compliance
{{ range .Compliance }}
### {{ .Framework }}
- Score: {{ printf "%.2f" .Score }}
- Status: {{ .Status }}
{{ range .Findings }}- Finding: {{ . }}
{{ end }}{{ range .RequiredGaps }}- Required Gap: {{ . }}
{{ end }}{{ range .RecommendedActions }}- Recommendation: {{ . }}
{{ end }}
{{ end }}
`

func renderMarkdown(card core.ModelCard, templatePath string) (string, error) {
	tmplString := defaultMarkdownTemplate
	if strings.TrimSpace(templatePath) != "" {
		if _, err := os.Stat(templatePath); err == nil {
			data, err := os.ReadFile(filepath.Clean(templatePath))
			if err != nil {
				return "", err
			}
			tmplString = string(data)
		}
	}

	tmpl, err := template.New("card").Parse(tmplString)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, card); err != nil {
		return "", err
	}
	return out.String(), nil
}

func renderHTML(card core.ModelCard, templatePath string) (string, error) {
	md, err := renderMarkdown(card, templatePath)
	if err != nil {
		return "", err
	}

	var htmlBuf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &htmlBuf); err != nil {
		return "", err
	}

	doc := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Model Card</title>
<style>
body { font-family: "Helvetica Neue", Arial, sans-serif; margin: 2rem; color: #1f2937; line-height: 1.5; }
h1, h2, h3 { color: #0f172a; }
code, pre { background: #f1f5f9; padding: 0.2rem 0.4rem; border-radius: 4px; }
blockquote { border-left: 3px solid #94a3b8; margin: 0; padding-left: 1rem; color: #334155; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #cbd5e1; padding: 0.5rem; }
</style>
</head>
<body>
` + htmlBuf.String() + `
</body>
</html>`

	return html.UnescapeString(doc), nil
}
