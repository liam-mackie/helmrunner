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
