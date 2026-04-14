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
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {}); err != nil {
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
		cp, err := install.LocateChart(ref, cli.New())
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
