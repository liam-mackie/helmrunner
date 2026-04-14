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
	Values    map[string]any `yaml:"values"`
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
