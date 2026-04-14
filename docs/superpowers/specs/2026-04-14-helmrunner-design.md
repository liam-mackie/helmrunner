# Helmrunner Design Spec

A TUI application for selecting and deploying Helm chart definitions from YAML files.

## Definition YAML Format

Each definition is a YAML file in the target directory (non-recursive, `*.yaml`/`*.yml`):

```yaml
name: my-app
release: "{{ .environment }}-my-app"   # templatable, overrideable in review
namespace: "{{ .environment }}"         # templatable, overrideable in review
chart:
  # Option 1: OCI or local path
  source: oci://registry.example.com/charts/my-app
  # Option 2: Helm repository
  # repository: https://charts.bitnami.com/bitnami
  # name: nginx
  # version: "1.2.3"
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
    tag: "{{ .image_tag }}"
  nested:
    key: "static-value"
```

### Validation Rules

- `name` is required
- `chart` is required, with either `chart.source` (OCI/local) or `chart.repository` + `chart.name` (Helm repo)
- `chart.source` and `chart.repository` are mutually exclusive
- `chart.version` is optional (only valid with `chart.repository`)
- `variables` is optional
- `values` is optional
- `release` defaults to `name` if not specified
- `namespace` defaults to `default` if not specified

### Templating

`release`, `namespace`, and `values` fields support Go `text/template` syntax with variables injected from the `variables` section. Template delimiters are `{{ }}`.

## CLI Interface

```
helmrunner [directory]           # interactive install mode (default: current directory)
helmrunner --template [directory] # interactive template mode, outputs YAML to stdout
```

## TUI Flow

### Screen 1: Definition Selection

Multi-select list using Bubbles `list` component. Displays definition `name` only. User toggles selection with space, confirms with enter.

### Screen 2: Variable Input

For each selected definition that has `variables`, prompts for each variable sequentially. Shows the variable description and default value. Enter accepts the default, or the user types a new value. Uses Bubbles `textinput` component.

### Screen 3: Review/Overview

Table showing each selected definition with columns: Name, Release, Namespace, Chart. Release and namespace are resolved (variables templated in). User can navigate to any row and edit the release or namespace inline. Enter confirms and proceeds, `q`/`esc` aborts.

### Screen 4: Execution (install mode only)

Shows progress for each definition as Helm operations run. Status per definition: pending / running / success / error. Errors display the Helm error message inline.

### Template Mode

Screens 1-3 are identical. After review confirmation, the TUI exits and prints rendered YAML to stdout. Multiple definitions are separated by `---`.

## Architecture

### Package Structure

```
cmd/helmrunner/main.go     # CLI entry point, flag parsing, orchestration
internal/config/            # YAML parsing, validation, variable resolution
internal/tui/               # Bubble Tea model, all TUI screens
internal/helm/              # Helm SDK wrapper for install and template
```

### `internal/config`

- `Definition` struct matching the YAML schema
- `Load(dir string) ([]Definition, error)` — reads all YAML/YML files, validates, returns definitions
- `Resolve(def Definition, vars map[string]string) (ResolvedDefinition, error)` — templates variables into values, release, and namespace using `text/template`

### `internal/tui`

- Bubble Tea model with state machine: `selecting` → `inputting` → `reviewing` → `executing`
- Uses Bubbles `list` for selection, `textinput` for variable input, `table` for review
- Returns `[]ResolvedDefinition` and chosen action

### `internal/helm`

- `Install(ctx context.Context, def ResolvedDefinition) error` — Helm SDK `action.Upgrade` with `Install: true`
- `Template(ctx context.Context, def ResolvedDefinition) (string, error)` — Helm SDK `action.Install` with `DryRun: true, ClientOnly: true`
- Handles chart loading for all three source types: OCI, Helm repository, local path

### Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — reusable TUI components
- `github.com/charmbracelet/lipgloss` — TUI styling
- `helm.sh/helm/v3` — Helm Go SDK
- `gopkg.in/yaml.v3` — YAML parsing

## CI/CD

### `ci.yml` — Push/PR to main

- Lint with `golangci-lint`
- Build
- Test

### `release-please.yml` — Push to main

- Uses `googleapis/release-please-action` with `go` release type
- Reads conventional commits to auto-generate changelog and bump version
- Creates/updates a release PR
- When merged, creates a git tag and GitHub Release

### `goreleaser.yml` — Tag creation

- Triggered by tags created by release-please
- Builds binaries for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Signs binaries with `cosign` (keyless/OIDC via GitHub Actions identity)
- Attests provenance with `actions/attest-build-provenance`
- Uploads artifacts to the GitHub Release

### Commit Convention

All commits use conventional commit format (`feat:`, `fix:`, `ci:`, `docs:`, `chore:`, `refactor:`). Release-please uses these to determine version bumps and generate changelogs.
