package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/liam-mackie/helmrunner/internal/config"
)

func TestLoad_ValidSourceChart(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: my-app
chart:
  source: oci://registry.example.com/charts/my-app
`
	if err := os.WriteFile(filepath.Join(dir, "my-app.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "my-app" {
		t.Errorf("expected name 'my-app', got %q", defs[0].Name)
	}
	if defs[0].Chart.Source != "oci://registry.example.com/charts/my-app" {
		t.Errorf("unexpected chart source: %q", defs[0].Chart.Source)
	}
}

func TestLoad_ValidRepoChart(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: nginx
chart:
  repository: https://charts.bitnami.com/bitnami
  name: nginx
  version: "1.2.3"
`
	if err := os.WriteFile(filepath.Join(dir, "nginx.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Chart.Repository != "https://charts.bitnami.com/bitnami" {
		t.Errorf("unexpected repository: %q", defs[0].Chart.Repository)
	}
	if defs[0].Chart.Name != "nginx" {
		t.Errorf("unexpected chart name: %q", defs[0].Chart.Name)
	}
	if defs[0].Chart.Version != "1.2.3" {
		t.Errorf("unexpected version: %q", defs[0].Chart.Version)
	}
}

func TestLoad_WithVariablesAndValues(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: my-app
release: "{{ .environment }}-my-app"
namespace: "{{ .environment }}"
chart:
  source: ./charts/my-app
variables:
  - name: environment
    description: "Target environment"
    default: "staging"
values:
  replicaCount: "{{ .replicas }}"
`
	if err := os.WriteFile(filepath.Join(dir, "app.yml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Release != "{{ .environment }}-my-app" {
		t.Errorf("unexpected release: %q", defs[0].Release)
	}
	if len(defs[0].Variables) != 1 {
		t.Fatalf("expected 1 variable, got %d", len(defs[0].Variables))
	}
	if defs[0].Variables[0].Default != "staging" {
		t.Errorf("unexpected default: %q", defs[0].Variables[0].Default)
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	yaml := `
name: my-app
chart:
  source: oci://example.com/chart
`
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	defs, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defs[0].Release != "my-app" {
		t.Errorf("expected release to default to name, got %q", defs[0].Release)
	}
	if defs[0].Namespace != "default" {
		t.Errorf("expected namespace to default to 'default', got %q", defs[0].Namespace)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing name",
			yaml: `
chart:
  source: oci://example.com/chart
`,
		},
		{
			name: "missing chart",
			yaml: `
name: my-app
`,
		},
		{
			name: "both source and repository",
			yaml: `
name: my-app
chart:
  source: oci://example.com/chart
  repository: https://example.com
  name: chart
`,
		},
		{
			name: "repository without chart name",
			yaml: `
name: my-app
chart:
  repository: https://example.com
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := config.Load(dir)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestLoad_SkipsNonYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	defs, err := config.Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(defs) != 0 {
		t.Fatalf("expected 0 definitions, got %d", len(defs))
	}
}
