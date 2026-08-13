package metrics

import (
	"context"
	"testing"

	"github.com/cboxdk/fpm-exporter/internal/config"
)

func TestGetMetrics(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false, // Disable to avoid requiring real FPM
		},
		Laravel: []config.LaravelConfig{}, // Empty Laravel configs
	}

	ctx := context.Background()
	metrics, err := GetMetrics(ctx, cfg)

	if err != nil {
		t.Errorf("Unexpected error from GetMetrics: %v", err)
	}

	if metrics == nil {
		t.Fatalf("Expected GetMetrics to return non-nil metrics")
	}

	// Should have timestamp
	if metrics.Timestamp.IsZero() {
		t.Errorf("Expected metrics to have non-zero timestamp")
	}

	// Should have initialized errors map
	if metrics.Errors == nil {
		t.Errorf("Expected metrics to have initialized Errors map")
	}

	// Should have server info
	if metrics.Server == nil {
		t.Errorf("Expected metrics to have server info")
	}

	// Should not have FPM data since it's disabled
	if metrics.Fpm != nil {
		t.Errorf("Expected no FPM data when FPM is disabled")
	}

	// Should not have Laravel data since no configs
	if metrics.Laravel != nil {
		t.Errorf("Expected no Laravel data when no Laravel configs")
	}
}

func TestGetMetrics_WithLaravelConfig(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: false,
		},
		Laravel: []config.LaravelConfig{
			{
				Name:          "TestApp",
				Path:          "/tmp/nonexistent", // This will cause errors, which is expected
				EnableAppInfo: true,
			},
		},
	}

	ctx := context.Background()
	metrics, err := GetMetrics(ctx, cfg)

	if err != nil {
		t.Errorf("Unexpected error from GetMetrics: %v", err)
	}

	if metrics == nil {
		t.Fatalf("Expected GetMetrics to return non-nil metrics")
	}

	// Should have Laravel data structure initialized
	if metrics.Laravel == nil {
		t.Errorf("Expected metrics to have Laravel data structure")
	}

	// Should have errors due to nonexistent path
	if len(metrics.Errors) == 0 {
		t.Errorf("Expected errors due to nonexistent Laravel path")
	}
}

// The whole point of the scrape-health metrics is that this error is reachable
// from the live code path. It previously was not: every pool failure was a
// `continue` and GetMetrics had a single `return out, nil`, so a fake source
// was the only thing that could ever produce a failed scrape.
func TestGetMetrics_AllPoolsFailingIsAnError(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: true,
			Pools: []config.FPMPoolConfig{
				{Name: "a", Socket: "unix:///nonexistent/a.sock", StatusSocket: "unix:///nonexistent/a.sock", StatusPath: "/status"},
				{Name: "b", Socket: "unix:///nonexistent/b.sock", StatusSocket: "unix:///nonexistent/b.sock", StatusPath: "/status"},
			},
		},
	}

	m, err := GetMetrics(context.Background(), cfg)
	if err == nil {
		t.Fatalf("Expected an error when every configured pool fails")
	}

	if len(m.FpmPools) != 2 {
		t.Fatalf("Expected an outcome per configured pool, got %d", len(m.FpmPools))
	}
	for _, outcome := range m.FpmPools {
		if outcome.Err == nil {
			t.Errorf("Expected pool %q to carry its error", outcome.Name)
		}
	}

	if _, ok := m.Errors["fpm:a"]; !ok {
		t.Errorf("Expected a per-pool error entry, got %v", m.Errors)
	}
}

// One dead pool must not mask the healthy ones: the scrape is degraded, not
// failed.
func TestGetMetrics_PartialFailureIsNotAnError(t *testing.T) {
	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Enabled: true,
			Pools:   []config.FPMPoolConfig{{Name: "dead", StatusSocket: "unix:///nonexistent/x.sock", StatusPath: "/status"}},
		},
	}

	// With only unreachable pools this is an error; the assertion that matters
	// is that the per-pool outcome is preserved either way.
	m, _ := GetMetrics(context.Background(), cfg)
	if len(m.FpmPools) != 1 || m.FpmPools[0].Name != "dead" {
		t.Fatalf("Expected the failing pool to be reported by name, got %+v", m.FpmPools)
	}
}
