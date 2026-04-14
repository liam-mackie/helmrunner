# Helmrunner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go TUI application that reads Helm chart definitions from YAML files, lets users select and configure them interactively, then deploys or templates them via the Helm SDK.

**Architecture:** Three internal packages (`config`, `tui`, `helm`) behind a thin CLI entry point. Config handles YAML parsing and Go template resolution. TUI uses Bubble Tea with a state machine across four screens. Helm wraps the Helm Go SDK for upgrade-install and template operations.

**Tech Stack:** Go 1.23+, Bubble Tea/Bubbles/Lipgloss, Helm v3 SDK, gopkg.in/yaml.v3, GoReleaser, release-please, cosign, GitHub Actions

---

## File Structure

```
cmd/helmrunner/main.go              # CLI entry, flag parsing, orchestration
internal/config/config.go           # Definition struct, Load(), Validate()
internal/config/config_test.go      # Tests for loading and validation
internal/config/resolve.go          # ResolvedDefinition, Resolve() with text/template
internal/config/resolve_test.go     # Tests for variable resolution and templating
internal/tui/model.go               # Root Bubble Tea model, state machine, messages
internal/tui/select.go              # Screen 1: multi-select definition list
internal/tui/input.go               # Screen 2: variable input prompts
internal/tui/review.go              # Screen 3: review table with inline editing
internal/tui/execute.go             # Screen 4: execution progress display
internal/tui/styles.go              # Shared lipgloss styles
internal/helm/helm.go               # Install() and Template() using Helm SDK
internal/helm/helm_test.go          # Tests for chart source resolution
.github/workflows/ci.yml            # Lint, build, test
.github/workflows/release-please.yml # Auto release PR and tagging
.github/workflows/goreleaser.yml    # Cross-platform build, sign, attest
.goreleaser.yml                     # GoReleaser config
.golangci.yml                       # Linter config
go.mod                              # Module definition
go.sum                              # Dependency checksums
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/helmrunner/main.go`
- Create: `.golangci.yml`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/liammackie/g/helmrunner
go mod init github.com/liam-mackie/helmrunner
```

- [ ] **Step 2: Create minimal main.go**

Create `cmd/helmrunner/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("helmrunner")
	os.Exit(0)
}
```

- [ ] **Step 3: Create linter config**

Create `.golangci.yml`:

```yaml
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign

run:
  timeout: 5m
```

- [ ] **Step 4: Verify it builds and runs**

```bash
go build -o helmrunner ./cmd/helmrunner
./helmrunner
```

Expected: prints `helmrunner` and exits 0.

- [ ] **Step 5: Commit**

```bash
git add go.mod cmd/helmrunner/main.go .golangci.yml
git commit -m "feat: scaffold project with Go module and main entry point"
```

---

### Task 2: Config — Definition Struct and Loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test for Load()**

Create `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/...
```

Expected: compilation error — package doesn't exist yet.

- [ ] **Step 3: Implement config.go**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Variable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

type Chart struct {
	Source     string `yaml:"source"`
	Repository string `yaml:"repository"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
}

type Definition struct {
	Name      string                 `yaml:"name"`
	Release   string                 `yaml:"release"`
	Namespace string                 `yaml:"namespace"`
	Chart     Chart                  `yaml:"chart"`
	Variables []Variable             `yaml:"variables"`
	Values    map[string]interface{} `yaml:"values"`
}

func Load(dir string) ([]Definition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var defs []Definition
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}

		var def Definition
		if err := yaml.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}

		if def.Release == "" {
			def.Release = def.Name
		}
		if def.Namespace == "" {
			def.Namespace = "default"
		}

		if err := validate(def, entry.Name()); err != nil {
			return nil, err
		}

		defs = append(defs, def)
	}

	return defs, nil
}

func validate(def Definition, filename string) error {
	if def.Name == "" {
		return fmt.Errorf("%s: name is required", filename)
	}

	hasSource := def.Chart.Source != ""
	hasRepo := def.Chart.Repository != ""

	if !hasSource && !hasRepo {
		return fmt.Errorf("%s: chart.source or chart.repository is required", filename)
	}
	if hasSource && hasRepo {
		return fmt.Errorf("%s: chart.source and chart.repository are mutually exclusive", filename)
	}
	if hasRepo && def.Chart.Name == "" {
		return fmt.Errorf("%s: chart.name is required when using chart.repository", filename)
	}

	return nil
}
```

- [ ] **Step 4: Install yaml dependency and run tests**

```bash
go get gopkg.in/yaml.v3
go test ./internal/config/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go go.mod go.sum
git commit -m "feat: add config loading and validation for YAML definitions"
```

---

### Task 3: Config — Variable Resolution and Templating

**Files:**
- Create: `internal/config/resolve.go`
- Create: `internal/config/resolve_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/config/resolve_test.go`:

```go
package config_test

import (
	"testing"

	"github.com/liam-mackie/helmrunner/internal/config"
)

func TestResolve_BasicTemplating(t *testing.T) {
	def := config.Definition{
		Name:      "my-app",
		Release:   "{{ .environment }}-my-app",
		Namespace: "{{ .environment }}",
		Chart:     config.Chart{Source: "oci://example.com/chart"},
		Values: map[string]interface{}{
			"replicaCount": "{{ .replicas }}",
			"nested": map[string]interface{}{
				"key": "static",
			},
		},
	}
	vars := map[string]string{
		"environment": "prod",
		"replicas":    "5",
	}

	resolved, err := config.Resolve(def, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Release != "prod-my-app" {
		t.Errorf("expected release 'prod-my-app', got %q", resolved.Release)
	}
	if resolved.Namespace != "prod" {
		t.Errorf("expected namespace 'prod', got %q", resolved.Namespace)
	}
	rc, ok := resolved.Values["replicaCount"].(string)
	if !ok || rc != "5" {
		t.Errorf("expected replicaCount '5', got %v", resolved.Values["replicaCount"])
	}
	nested, ok := resolved.Values["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", resolved.Values["nested"])
	}
	if nested["key"] != "static" {
		t.Errorf("expected static value preserved, got %v", nested["key"])
	}
}

func TestResolve_NoTemplates(t *testing.T) {
	def := config.Definition{
		Name:      "simple",
		Release:   "simple",
		Namespace: "default",
		Chart:     config.Chart{Source: "./local"},
		Values: map[string]interface{}{
			"key": "value",
		},
	}

	resolved, err := config.Resolve(def, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.Release != "simple" {
		t.Errorf("unexpected release: %q", resolved.Release)
	}
	if resolved.Values["key"] != "value" {
		t.Errorf("unexpected value: %v", resolved.Values["key"])
	}
}

func TestResolve_InvalidTemplate(t *testing.T) {
	def := config.Definition{
		Name:    "bad",
		Release: "{{ .missing",
		Chart:   config.Chart{Source: "oci://example.com/chart"},
	}

	_, err := config.Resolve(def, nil)
	if err == nil {
		t.Fatal("expected template error, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/... -v
```

Expected: compilation error — `Resolve` and `ResolvedDefinition` don't exist.

- [ ] **Step 3: Implement resolve.go**

Create `internal/config/resolve.go`:

```go
package config

import (
	"bytes"
	"fmt"
	"text/template"
)

type ResolvedDefinition struct {
	Name      string
	Release   string
	Namespace string
	Chart     Chart
	Values    map[string]interface{}
}

func Resolve(def Definition, vars map[string]string) (ResolvedDefinition, error) {
	release, err := resolveString(def.Release, vars)
	if err != nil {
		return ResolvedDefinition{}, fmt.Errorf("resolving release: %w", err)
	}

	namespace, err := resolveString(def.Namespace, vars)
	if err != nil {
		return ResolvedDefinition{}, fmt.Errorf("resolving namespace: %w", err)
	}

	values, err := resolveValues(def.Values, vars)
	if err != nil {
		return ResolvedDefinition{}, fmt.Errorf("resolving values: %w", err)
	}

	return ResolvedDefinition{
		Name:      def.Name,
		Release:   release,
		Namespace: namespace,
		Chart:     def.Chart,
		Values:    values,
	}, nil
}

func resolveString(s string, vars map[string]string) (string, error) {
	tmpl, err := template.New("").Parse(s)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func resolveValues(values map[string]interface{}, vars map[string]string) (map[string]interface{}, error) {
	if values == nil {
		return nil, nil
	}
	result := make(map[string]interface{}, len(values))
	for k, v := range values {
		resolved, err := resolveValue(v, vars)
		if err != nil {
			return nil, fmt.Errorf("key %s: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}

func resolveValue(v interface{}, vars map[string]string) (interface{}, error) {
	switch val := v.(type) {
	case string:
		return resolveString(val, vars)
	case map[string]interface{}:
		return resolveValues(val, vars)
	default:
		return v, nil
	}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/config/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/resolve.go internal/config/resolve_test.go
git commit -m "feat: add variable resolution and templating for definitions"
```

---

### Task 4: Helm SDK Wrapper

**Files:**
- Create: `internal/helm/helm.go`
- Create: `internal/helm/helm_test.go`

- [ ] **Step 1: Write a test for chart source string generation**

Create `internal/helm/helm_test.go`:

```go
package helm

import (
	"testing"

	"github.com/liam-mackie/helmrunner/internal/config"
)

func TestChartRef(t *testing.T) {
	tests := []struct {
		name  string
		chart config.Chart
		want  string
	}{
		{
			name:  "OCI source",
			chart: config.Chart{Source: "oci://registry.example.com/charts/my-app"},
			want:  "oci://registry.example.com/charts/my-app",
		},
		{
			name:  "local source",
			chart: config.Chart{Source: "./charts/my-app"},
			want:  "./charts/my-app",
		},
		{
			name:  "repository chart",
			chart: config.Chart{Repository: "https://charts.bitnami.com/bitnami", Name: "nginx"},
			want:  "nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chartRef(tt.chart)
			if got != tt.want {
				t.Errorf("chartRef() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/helm/... -v
```

Expected: compilation error — package doesn't exist.

- [ ] **Step 3: Implement helm.go**

Create `internal/helm/helm.go`:

```go
package helm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

func chartRef(chart config.Chart) string {
	if chart.Source != "" {
		return chart.Source
	}
	return chart.Name
}

func newConfig(namespace string) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {}); err != nil {
		return nil, fmt.Errorf("initializing helm config: %w", err)
	}
	return cfg, nil
}

func locateChart(cfg *action.Configuration, chart config.Chart) (string, error) {
	ref := chartRef(chart)

	if chart.Source != "" && strings.HasPrefix(chart.Source, "oci://") {
		registryClient, err := registry.NewClient()
		if err != nil {
			return "", fmt.Errorf("creating registry client: %w", err)
		}
		cfg.RegistryClient = registryClient

		pull := action.NewPullWithOpts(action.WithConfig(cfg))
		pull.Settings = cli.New()
		dir, err := os.MkdirTemp("", "helmrunner-*")
		if err != nil {
			return "", fmt.Errorf("creating temp dir: %w", err)
		}
		pull.DestDir = dir
		pull.Untar = true
		pull.UntarDir = dir

		_, err = pull.Run(ref)
		if err != nil {
			return "", fmt.Errorf("pulling OCI chart: %w", err)
		}

		// Find the extracted chart directory
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", fmt.Errorf("reading temp dir: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				return fmt.Sprintf("%s/%s", dir, entry.Name()), nil
			}
		}
		return "", fmt.Errorf("no chart directory found after pull")
	}

	if chart.Repository != "" {
		install := action.NewInstall(cfg)
		install.RepoURL = chart.Repository
		install.Version = chart.Version
		cp, err := install.ChartPathOptions.LocateChart(ref, cli.New())
		if err != nil {
			return "", fmt.Errorf("locating chart: %w", err)
		}
		return cp, nil
	}

	// Local path
	return ref, nil
}

func Install(ctx context.Context, def config.ResolvedDefinition) error {
	cfg, err := newConfig(def.Namespace)
	if err != nil {
		return err
	}

	chartPath, err := locateChart(cfg, def.Chart)
	if err != nil {
		return err
	}

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("loading chart: %w", err)
	}

	upgrade := action.NewUpgrade(cfg)
	upgrade.Install = true
	upgrade.Namespace = def.Namespace

	_, err = upgrade.RunWithContext(ctx, def.Release, chartObj, def.Values)
	if err != nil {
		return fmt.Errorf("helm upgrade --install: %w", err)
	}

	return nil
}

func Template(ctx context.Context, def config.ResolvedDefinition) (string, error) {
	cfg, err := newConfig(def.Namespace)
	if err != nil {
		return "", err
	}

	chartPath, err := locateChart(cfg, def.Chart)
	if err != nil {
		return "", err
	}

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("loading chart: %w", err)
	}

	install := action.NewInstall(cfg)
	install.DryRun = true
	install.ClientOnly = true
	install.ReleaseName = def.Release
	install.Namespace = def.Namespace
	install.Replace = true

	rel, err := install.RunWithContext(ctx, chartObj, def.Values)
	if err != nil {
		return "", fmt.Errorf("helm template: %w", err)
	}

	return rel.Manifest, nil
}
```

- [ ] **Step 4: Get helm dependency and run tests**

```bash
go get helm.sh/helm/v3
go test ./internal/helm/... -v
```

Expected: `TestChartRef` passes. (Install/Template can't be unit-tested without a real cluster/chart, so we test the helper.)

- [ ] **Step 5: Commit**

```bash
git add internal/helm/helm.go internal/helm/helm_test.go go.mod go.sum
git commit -m "feat: add Helm SDK wrapper for install and template operations"
```

---

### Task 5: TUI — Styles and Root Model

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/model.go`

- [ ] **Step 1: Create shared styles**

Create `internal/tui/styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			PaddingBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
)
```

- [ ] **Step 2: Create root model with state machine**

Create `internal/tui/model.go`:

```go
package tui

import (
	"github.com/liam-mackie/helmrunner/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateSelecting state = iota
	stateInputting
	stateReviewing
	stateExecuting
	stateDone
)

type Result struct {
	Definitions []config.ResolvedDefinition
	Aborted     bool
}

type Model struct {
	state        state
	definitions  []config.Definition
	templateMode bool

	// Sub-models
	selectModel  selectModel
	inputModel   inputModel
	reviewModel  reviewModel
	executeModel executeModel

	result Result
	width  int
	height int
}

func New(defs []config.Definition, templateMode bool) Model {
	m := Model{
		state:        stateSelecting,
		definitions:  defs,
		templateMode: templateMode,
	}
	m.selectModel = newSelectModel(defs)
	return m
}

func (m Model) Init() tea.Cmd {
	return m.selectModel.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.result.Aborted = true
			return m, tea.Quit
		}
	}

	switch m.state {
	case stateSelecting:
		return m.updateSelecting(msg)
	case stateInputting:
		return m.updateInputting(msg)
	case stateReviewing:
		return m.updateReviewing(msg)
	case stateExecuting:
		return m.updateExecuting(msg)
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateSelecting:
		return m.selectModel.View()
	case stateInputting:
		return m.inputModel.View()
	case stateReviewing:
		return m.reviewModel.View()
	case stateExecuting:
		return m.executeModel.View()
	case stateDone:
		return ""
	}
	return ""
}

// GetResult returns the result after the TUI exits.
func (m Model) GetResult() Result {
	return m.result
}

// Messages for state transitions
type selectDoneMsg struct {
	selected []config.Definition
}

type inputDoneMsg struct {
	varSets []map[string]string
}

type reviewDoneMsg struct {
	definitions []config.ResolvedDefinition
}

type executeDoneMsg struct{}

func (m Model) updateSelecting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.selectModel, cmd = m.selectModel.Update(msg)

	if done, ok := msg.(selectDoneMsg); ok {
		if len(done.selected) == 0 {
			m.result.Aborted = true
			return m, tea.Quit
		}
		m.inputModel = newInputModel(done.selected)
		m.state = stateInputting
		return m, m.inputModel.Init()
	}

	return m, cmd
}

func (m Model) updateInputting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.inputModel, cmd = m.inputModel.Update(msg)

	if done, ok := msg.(inputDoneMsg); ok {
		selected := m.inputModel.definitions
		resolved := make([]config.ResolvedDefinition, len(selected))
		for i, def := range selected {
			r, err := config.Resolve(def, done.varSets[i])
			if err != nil {
				// Show error in review with unresolved values
				resolved[i] = config.ResolvedDefinition{
					Name:      def.Name,
					Release:   def.Release,
					Namespace: def.Namespace,
					Chart:     def.Chart,
					Values:    def.Values,
				}
				continue
			}
			resolved[i] = r
		}
		m.reviewModel = newReviewModel(resolved, m.templateMode)
		m.state = stateReviewing
		return m, nil
	}

	return m, cmd
}

func (m Model) updateReviewing(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.reviewModel, cmd = m.reviewModel.Update(msg)

	if done, ok := msg.(reviewDoneMsg); ok {
		if m.templateMode {
			m.result.Definitions = done.definitions
			m.state = stateDone
			return m, tea.Quit
		}
		m.executeModel = newExecuteModel(done.definitions)
		m.state = stateExecuting
		return m, m.executeModel.Init()
	}

	return m, cmd
}

func (m Model) updateExecuting(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.executeModel, cmd = m.executeModel.Update(msg)

	if _, ok := msg.(executeDoneMsg); ok {
		m.state = stateDone
		return m, tea.Quit
	}

	return m, cmd
}
```

- [ ] **Step 3: Get bubbletea dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```

- [ ] **Step 4: Verify it compiles** (it won't until sub-models exist — that's expected, move to next task)

- [ ] **Step 5: Commit** (defer until sub-models compile)

---

### Task 6: TUI — Selection Screen

**Files:**
- Create: `internal/tui/select.go`

- [ ] **Step 1: Implement the selection screen**

Create `internal/tui/select.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type selectModel struct {
	definitions []config.Definition
	cursor      int
	selected    map[int]bool
}

func newSelectModel(defs []config.Definition) selectModel {
	return selectModel{
		definitions: defs,
		selected:    make(map[int]bool),
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (selectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.definitions)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "enter":
			var selected []config.Definition
			for i, def := range m.definitions {
				if m.selected[i] {
					selected = append(selected, def)
				}
			}
			return m, func() tea.Msg {
				return selectDoneMsg{selected: selected}
			}
		case "q", "esc":
			return m, func() tea.Msg {
				return selectDoneMsg{selected: nil}
			}
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select definitions to deploy"))
	b.WriteString("\n\n")

	for i, def := range m.definitions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checked := "[ ]"
		if m.selected[i] {
			checked = "[x]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, checked, def.Name)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("space: toggle • enter: confirm • q: quit"))

	return b.String()
}
```

- [ ] **Step 2: Move on to next screen** (no independent test — TUI tested by running the app)

---

### Task 7: TUI — Variable Input Screen

**Files:**
- Create: `internal/tui/input.go`

- [ ] **Step 1: Implement the variable input screen**

Create `internal/tui/input.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type inputModel struct {
	definitions []config.Definition
	defIndex    int
	varIndex    int
	varSets     []map[string]string
	textInput   textinput.Model
	done        bool
}

func newInputModel(defs []config.Definition) inputModel {
	varSets := make([]map[string]string, len(defs))
	for i := range varSets {
		varSets[i] = make(map[string]string)
	}

	m := inputModel{
		definitions: defs,
		varSets:     varSets,
	}
	m.advanceToNextVariable()
	return m
}

func (m *inputModel) advanceToNextVariable() {
	for m.defIndex < len(m.definitions) {
		def := m.definitions[m.defIndex]
		if m.varIndex < len(def.Variables) {
			v := def.Variables[m.varIndex]
			ti := textinput.New()
			ti.Placeholder = v.Default
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			return
		}
		m.defIndex++
		m.varIndex = 0
	}
	m.done = true
}

func (m inputModel) currentVariable() config.Variable {
	return m.definitions[m.defIndex].Variables[m.varIndex]
}

func (m inputModel) Init() tea.Cmd {
	if m.done {
		return func() tea.Msg {
			return inputDoneMsg{varSets: m.varSets}
		}
	}
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	if m.done {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			v := m.currentVariable()
			value := m.textInput.Value()
			if value == "" {
				value = v.Default
			}
			m.varSets[m.defIndex][v.Name] = value
			m.varIndex++
			m.advanceToNextVariable()
			if m.done {
				return m, func() tea.Msg {
					return inputDoneMsg{varSets: m.varSets}
				}
			}
			return m, textinput.Blink
		case "esc", "q":
			// Can't abort from here easily, ctrl+c still works
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	def := m.definitions[m.defIndex]
	v := m.currentVariable()

	b.WriteString(titleStyle.Render(fmt.Sprintf("Variables for: %s", def.Name)))
	b.WriteString("\n\n")

	b.WriteString(promptStyle.Render(v.Description))
	b.WriteString("\n")
	if v.Default != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("default: %s", v.Default)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("enter: accept • ctrl+c: abort"))

	return b.String()
}
```

---

### Task 8: TUI — Review Screen

**Files:**
- Create: `internal/tui/review.go`

- [ ] **Step 1: Implement the review screen with inline editing**

Create `internal/tui/review.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type editField int

const (
	editNone editField = iota
	editRelease
	editNamespace
)

type reviewModel struct {
	definitions  []config.ResolvedDefinition
	cursor       int
	editing      editField
	textInput    textinput.Model
	templateMode bool
}

func newReviewModel(defs []config.ResolvedDefinition, templateMode bool) reviewModel {
	return reviewModel{
		definitions:  defs,
		templateMode: templateMode,
	}
}

func (m reviewModel) Update(msg tea.Msg) (reviewModel, tea.Cmd) {
	if m.editing != editNone {
		return m.updateEditing(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.definitions)-1 {
				m.cursor++
			}
		case "r":
			ti := textinput.New()
			ti.SetValue(m.definitions[m.cursor].Release)
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			m.editing = editRelease
			return m, textinput.Blink
		case "n":
			ti := textinput.New()
			ti.SetValue(m.definitions[m.cursor].Namespace)
			ti.Focus()
			ti.CharLimit = 256
			ti.Width = 40
			m.textInput = ti
			m.editing = editNamespace
			return m, textinput.Blink
		case "enter":
			return m, func() tea.Msg {
				return reviewDoneMsg{definitions: m.definitions}
			}
		case "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m reviewModel) updateEditing(msg tea.Msg) (reviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			value := m.textInput.Value()
			if m.editing == editRelease {
				m.definitions[m.cursor].Release = value
			} else {
				m.definitions[m.cursor].Namespace = value
			}
			m.editing = editNone
			return m, nil
		case "esc":
			m.editing = editNone
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m reviewModel) chartString(chart config.Chart) string {
	if chart.Source != "" {
		return chart.Source
	}
	s := chart.Name
	if chart.Version != "" {
		s += "@" + chart.Version
	}
	return s
}

func (m reviewModel) View() string {
	var b strings.Builder

	action := "deploy"
	if m.templateMode {
		action = "template"
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("Review — %s the following:", action)))
	b.WriteString("\n\n")

	// Column headers
	b.WriteString(fmt.Sprintf("  %-20s %-25s %-20s %s\n",
		dimStyle.Render("NAME"),
		dimStyle.Render("RELEASE"),
		dimStyle.Render("NAMESPACE"),
		dimStyle.Render("CHART")))
	b.WriteString("\n")

	for i, def := range m.definitions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%-20s %-25s %-20s %s",
			cursor, def.Name, def.Release, def.Namespace, m.chartString(def.Chart))

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	if m.editing != editNone {
		b.WriteString("\n")
		label := "Release"
		if m.editing == editNamespace {
			label = "Namespace"
		}
		b.WriteString(promptStyle.Render(fmt.Sprintf("Edit %s: ", label)))
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("r: edit release • n: edit namespace • enter: confirm • q: quit"))

	return b.String()
}
```

---

### Task 9: TUI — Execution Screen

**Files:**
- Create: `internal/tui/execute.go`

- [ ] **Step 1: Implement the execution screen**

Create `internal/tui/execute.go`:

```go
package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/liam-mackie/helmrunner/internal/helm"

	tea "github.com/charmbracelet/bubbletea"
)

type deployStatus int

const (
	statusPending deployStatus = iota
	statusRunning
	statusSuccess
	statusError
)

type deployResult struct {
	index int
	err   error
}

type executeModel struct {
	definitions []config.ResolvedDefinition
	statuses    []deployStatus
	errors      []string
	current     int
	allDone     bool
}

func newExecuteModel(defs []config.ResolvedDefinition) executeModel {
	return executeModel{
		definitions: defs,
		statuses:    make([]deployStatus, len(defs)),
		errors:      make([]string, len(defs)),
		current:     0,
	}
}

func (m executeModel) Init() tea.Cmd {
	if len(m.definitions) == 0 {
		return func() tea.Msg { return executeDoneMsg{} }
	}
	return m.runNext(0)
}

func (m executeModel) runNext(index int) tea.Cmd {
	def := m.definitions[index]
	return func() tea.Msg {
		err := helm.Install(context.Background(), def)
		return deployResult{index: index, err: err}
	}
}

func (m executeModel) Update(msg tea.Msg) (executeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deployResult:
		if msg.err != nil {
			m.statuses[msg.index] = statusError
			m.errors[msg.index] = msg.err.Error()
		} else {
			m.statuses[msg.index] = statusSuccess
		}

		next := msg.index + 1
		if next < len(m.definitions) {
			m.current = next
			m.statuses[next] = statusRunning
			return m, m.runNext(next)
		}

		m.allDone = true
		return m, nil

	case tea.KeyMsg:
		if m.allDone && (msg.String() == "enter" || msg.String() == "q") {
			return m, func() tea.Msg { return executeDoneMsg{} }
		}
	}

	return m, nil
}

func (m executeModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Deploying..."))
	b.WriteString("\n\n")

	for i, def := range m.definitions {
		var icon string
		switch m.statuses[i] {
		case statusPending:
			icon = dimStyle.Render("○")
		case statusRunning:
			icon = selectedStyle.Render("●")
		case statusSuccess:
			icon = successStyle.Render("✓")
		case statusError:
			icon = errorStyle.Render("✗")
		}

		line := fmt.Sprintf(" %s %s → %s/%s", icon, def.Name, def.Namespace, def.Release)
		b.WriteString(line + "\n")

		if m.statuses[i] == statusError {
			b.WriteString("   " + errorStyle.Render(m.errors[i]) + "\n")
		}
	}

	if m.allDone {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("enter/q: exit"))
	}

	return b.String()
}
```

---

### Task 10: Wire Up Main and Compile All TUI

**Files:**
- Modify: `cmd/helmrunner/main.go`

- [ ] **Step 1: Wire up main.go**

Replace `cmd/helmrunner/main.go` with:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/liam-mackie/helmrunner/internal/config"
	"github.com/liam-mackie/helmrunner/internal/helm"
	"github.com/liam-mackie/helmrunner/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	templateMode := flag.Bool("template", false, "render templates to stdout instead of installing")
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	defs, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(defs) == 0 {
		fmt.Fprintf(os.Stderr, "No definitions found in %s\n", dir)
		os.Exit(1)
	}

	model := tui.New(defs, *templateMode)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result := finalModel.(tui.Model).GetResult()
	if result.Aborted {
		os.Exit(0)
	}

	if *templateMode {
		var outputs []string
		for _, def := range result.Definitions {
			rendered, err := helm.Template(context.Background(), def)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error templating %s: %v\n", def.Name, err)
				os.Exit(1)
			}
			outputs = append(outputs, rendered)
		}
		fmt.Print(strings.Join(outputs, "\n---\n"))
	}
}
```

- [ ] **Step 2: Build and verify compilation**

```bash
go mod tidy
go build ./cmd/helmrunner
```

Expected: builds successfully.

- [ ] **Step 3: Commit all TUI files and wired main**

```bash
git add cmd/helmrunner/main.go internal/tui/ go.mod go.sum
git commit -m "feat: add TUI with selection, variable input, review, and execution screens"
```

---

### Task 11: GitHub Actions — CI

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  contents: read

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Build
        run: go build ./cmd/helmrunner

      - name: Test
        run: go test ./... -v
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add CI workflow with lint, build, and test"
```

---

### Task 12: GitHub Actions — Release Please

**Files:**
- Create: `.github/workflows/release-please.yml`

- [ ] **Step 1: Create release-please workflow**

Create `.github/workflows/release-please.yml`:

```yaml
name: Release Please

on:
  push:
    branches: [main]

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.release.outputs.release_created }}
      tag_name: ${{ steps.release.outputs.tag_name }}
    steps:
      - uses: googleapis/release-please-action@v4
        id: release
        with:
          release-type: go
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release-please.yml
git commit -m "ci: add release-please for automated versioning and changelog"
```

---

### Task 13: GitHub Actions — GoReleaser with Signing and Attestation

**Files:**
- Create: `.goreleaser.yml`
- Create: `.github/workflows/goreleaser.yml`

- [ ] **Step 1: Create GoReleaser config**

Create `.goreleaser.yml`:

```yaml
version: 2

builds:
  - main: ./cmd/helmrunner
    binary: helmrunner
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: checksums.txt

snapshot:
  version_template: "{{ incpatch .Version }}-next"

changelog:
  disable: true
```

- [ ] **Step 2: Create GoReleaser workflow**

Create `.github/workflows/goreleaser.yml`:

```yaml
name: GoReleaser

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write
  packages: write
  id-token: write
  attestations: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: sigstore/cosign-installer@v3

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Sign checksums
        run: cosign sign-blob --yes checksums.txt --output-signature checksums.txt.sig --output-certificate checksums.txt.pem
        working-directory: dist

      - name: Upload signatures
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          gh release upload ${{ github.ref_name }} dist/checksums.txt.sig dist/checksums.txt.pem

      - name: Attest build provenance
        uses: actions/attest-build-provenance@v2
        with:
          subject-path: dist/helmrunner_*
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yml .github/workflows/goreleaser.yml
git commit -m "ci: add GoReleaser with cosign signing and build attestation"
```

---

### Task 14: Final Build Verification and Example Definition

**Files:**
- Create: `examples/my-app.yaml`

- [ ] **Step 1: Create example definition**

Create `examples/my-app.yaml`:

```yaml
name: my-app
release: "{{ .environment }}-my-app"
namespace: "{{ .environment }}"
chart:
  source: oci://registry.example.com/charts/my-app
variables:
  - name: environment
    description: "Target environment"
    default: "staging"
  - name: replicas
    description: "Number of replicas"
    default: "3"
values:
  replicaCount: "{{ .replicas }}"
  image:
    tag: latest
```

- [ ] **Step 2: Create a second example**

Create `examples/nginx.yaml`:

```yaml
name: nginx
chart:
  repository: https://charts.bitnami.com/bitnami
  name: nginx
  version: "18.1.0"
values:
  service:
    type: ClusterIP
```

- [ ] **Step 3: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests pass.

- [ ] **Step 4: Build final binary**

```bash
go build -o helmrunner ./cmd/helmrunner
```

Expected: builds successfully.

- [ ] **Step 5: Commit**

```bash
git add examples/
git commit -m "docs: add example definition files"
```

- [ ] **Step 6: Push to GitHub**

```bash
git push origin main
```
