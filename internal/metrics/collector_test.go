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
