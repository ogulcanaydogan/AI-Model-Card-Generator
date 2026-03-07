package unit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yapay/ai-model-card-generator/pkg/core"
	cardtemplates "github.com/yapay/ai-model-card-generator/pkg/templates"
)

func TestTemplateInitFromBase(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "my_template.tmpl")
	if err := cardtemplates.InitTemplate("My Template", outPath, "minimal"); err != nil {
		t.Fatalf("init template: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read template file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Custom template: My Template") {
		t.Fatalf("missing custom header in template: %s", content)
	}
	if !strings.Contains(content, "## Metadata") {
		t.Fatalf("expected minimal base content to exist: %s", content)
	}
}

func TestTemplateValidate(t *testing.T) {
	validPath := filepath.Join(t.TempDir(), "valid.tmpl")
	if err := os.WriteFile(validPath, []byte("# {{ .Metadata.Name }}"), 0o644); err != nil {
		t.Fatalf("write valid template: %v", err)
	}
	if err := cardtemplates.ValidateTemplateFile(validPath); err != nil {
		t.Fatalf("valid template should pass: %v", err)
	}

	badParsePath := filepath.Join(t.TempDir(), "bad_parse.tmpl")
	if err := os.WriteFile(badParsePath, []byte("{{ if }}"), 0o644); err != nil {
		t.Fatalf("write bad parse template: %v", err)
	}
	if err := cardtemplates.ValidateTemplateFile(badParsePath); err == nil {
		t.Fatalf("expected parse error for invalid template")
	}

	badFieldPath := filepath.Join(t.TempDir(), "bad_field.tmpl")
	if err := os.WriteFile(badFieldPath, []byte("{{ .Metadata.DoesNotExist }}"), 0o644); err != nil {
		t.Fatalf("write bad field template: %v", err)
	}
	if err := cardtemplates.ValidateTemplateFile(badFieldPath); err == nil {
		t.Fatalf("expected execute error for invalid placeholder")
	}
}

func TestTemplatePreviewDeterministic(t *testing.T) {
	templatePath := filepath.Join(t.TempDir(), "preview.tmpl")
	if err := os.WriteFile(templatePath, []byte("# {{ .Metadata.Name }}\nVersion: {{ .Version }}\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	card := core.ModelCard{
		Version: "v1",
		Metadata: core.ModelMetadata{
			Name: "preview-model",
		},
	}
	rendered, err := cardtemplates.RenderTemplateFile(templatePath, card)
	if err != nil {
		t.Fatalf("render template preview: %v", err)
	}
	expected := "# preview-model\nVersion: v1\n"
	if rendered != expected {
		t.Fatalf("unexpected preview output:\nexpected:\n%s\ngot:\n%s", expected, rendered)
	}
}
