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
	Values    map[string]any
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

func resolveValues(values map[string]any, vars map[string]string) (map[string]any, error) {
	if values == nil {
		return nil, nil
	}
	result := make(map[string]any, len(values))
	for k, v := range values {
		resolved, err := resolveValue(v, vars)
		if err != nil {
			return nil, fmt.Errorf("key %s: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}

func resolveValue(v any, vars map[string]string) (any, error) {
	switch val := v.(type) {
	case string:
		return resolveString(val, vars)
	case map[string]any:
		return resolveValues(val, vars)
	default:
		return v, nil
	}
}
