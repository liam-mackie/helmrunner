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
