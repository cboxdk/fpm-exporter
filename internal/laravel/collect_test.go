package laravel

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
)

func TestCollect(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		expectedSites  int
		expectedErrors int
	}{
		{
			name: "empty config",
			cfg: &config.Config{
				PHP:     config.PHPConfig{Binary: "php"},
				Laravel: []config.LaravelConfig{},
			},
			expectedSites:  0,
			expectedErrors: 0,
		},
		{
			name: "single site with default php binary",
			cfg: &config.Config{
				PHP: config.PHPConfig{Binary: "php"},
				Laravel: []config.LaravelConfig{
					{
						Name: "test-site",
						Path: "/tmp/test-laravel",
						Queues: map[string][]string{
							"default": {"default"},
						},
						EnableAppInfo: false,
					},
				},
			},
			expectedSites:  0, // No sites added when queue fails
			expectedErrors: 1, // Will fail without Laravel app
		},
		{
			name: "single site with custom php binary",
			cfg: &config.Config{
				PHP: config.PHPConfig{Binary: "php8.2"},
				Laravel: []config.LaravelConfig{
					{
						Name: "test-site",
						Path: "/tmp/test-laravel",
						PHPConfig: &config.PHPConfig{
							Binary: "php8.3",
						},
						Queues: map[string][]string{
							"redis": {"background", "emails"},
						},
						EnableAppInfo: true,
					},
				},
			},
			expectedSites:  0, // No sites added when queue fails
			expectedErrors: 1, // Queue error only (app info not called if queue fails)
		},
		{
			name: "multiple sites",
			cfg: &config.Config{
				PHP: config.PHPConfig{Binary: "php"},
				Laravel: []config.LaravelConfig{
					{
						Name: "site1",
						Path: "/tmp/site1",
						Queues: map[string][]string{
							"default": {"default"},
						},
						EnableAppInfo: false,
					},
					{
						Name: "site2",
						Path: "/tmp/site2",
						Queues: map[string][]string{
							"redis": {"high", "low"},
						},
						EnableAppInfo: true,
					},
				},
			},
			expectedSites:  0, // No sites added when queues fail
			expectedErrors: 2, // Both sites queue errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, errors := Collect(ctx, tt.cfg)

			if len(result) != tt.expectedSites {
				t.Errorf("Expected %d sites, got %d", tt.expectedSites, len(result))
			}

			if len(errors) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errors), errors)
			}

			// Only verify site presence if we expect sites to be present
			if tt.expectedSites > 0 {
				for _, site := range tt.cfg.Laravel {
					if _, exists := result[site.Name]; !exists {
						t.Errorf("Site %s not found in results", site.Name)
					}
				}
			}

			// Verify LaravelMetrics structure
			for siteName, metrics := range result {
				if metrics.Queues == nil {
					t.Errorf("Site %s has nil Queues", siteName)
				}
				// App info can be nil if not enabled or if it failed
			}
		})
	}
}

func TestLaravelMetrics_Structure(t *testing.T) {
	// Test that LaravelMetrics struct can be properly created
	queueSizes := &QueueSizes{}
	appInfo := &AppInfo{}

	metrics := LaravelMetrics{
		Queues: queueSizes,
		Info:   appInfo,
	}

	if metrics.Queues != queueSizes {
		t.Errorf("Expected Queues to be set correctly")
	}

	if metrics.Info != appInfo {
		t.Errorf("Expected Info to be set correctly")
	}

	// Test with nil values
	metricsWithNil := LaravelMetrics{
		Queues: nil,
		Info:   nil,
	}

	if metricsWithNil.Queues != nil {
		t.Errorf("Expected Queues to be nil")
	}

	if metricsWithNil.Info != nil {
		t.Errorf("Expected Info to be nil")
	}
}

func TestCollect_PHPBinarySelection(t *testing.T) {
	// Test that the correct PHP binary is selected based on configuration
	cfg := &config.Config{
		PHP: config.PHPConfig{Binary: "php8.1"},
		Laravel: []config.LaravelConfig{
			{
				Name: "site-with-default-php",
				Path: "/tmp/site1",
				Queues: map[string][]string{
					"default": {"default"},
				},
			},
			{
				Name: "site-with-custom-php",
				Path: "/tmp/site2",
				PHPConfig: &config.PHPConfig{
					Binary: "php8.3",
				},
				Queues: map[string][]string{
					"default": {"default"},
				},
			},
		},
	}

	ctx := context.Background()
	result, errors := Collect(ctx, cfg)

	// Should have 0 sites and 2 errors (queue failures prevent site addition)
	if len(result) != 0 {
		t.Errorf("Expected 0 sites, got %d", len(result))
	}

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}

// Each site boots its own PHP process, so collecting them in sequence made a
// scrape cost the sum of every site's boot time.
func TestCollect_RunsSitesConcurrently(t *testing.T) {
	const (
		sites     = 4
		sleepFor  = 300 * time.Millisecond
		tolerance = 3 * sleepFor // sequential would be 4x sleepFor
	)

	dir := t.TempDir()
	fakePHP := filepath.Join(dir, "fake-php")
	script := "#!/bin/sh\nsleep " + strconv.FormatFloat(sleepFor.Seconds(), 'f', 2, 64) + "\necho '{}'\n"
	if err := os.WriteFile(fakePHP, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to write fake php: %v", err)
	}

	cfg := &config.Config{PHP: config.PHPConfig{Binary: fakePHP}}
	for i := range sites {
		cfg.Laravel = append(cfg.Laravel, config.LaravelConfig{
			Name:   "site-" + strconv.Itoa(i),
			Path:   dir,
			Queues: map[string][]string{"redis": {"default"}},
		})
	}

	start := time.Now()
	result, errs := Collect(context.Background(), cfg)
	elapsed := time.Since(start)

	if len(errs) != 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	if len(result) != sites {
		t.Errorf("Expected %d sites, got %d", sites, len(result))
	}

	if elapsed > tolerance {
		t.Errorf("Expected sites to be collected concurrently; %d sites took %s", sites, elapsed)
	}
}

// One failing site must not cost the others their metrics.
func TestCollect_IsolatesSiteFailures(t *testing.T) {
	dir := t.TempDir()
	fakePHP := filepath.Join(dir, "fake-php")
	if err := os.WriteFile(fakePHP, []byte("#!/bin/sh\necho '{}'\n"), 0755); err != nil {
		t.Fatalf("Failed to write fake php: %v", err)
	}

	cfg := &config.Config{
		PHP: config.PHPConfig{Binary: fakePHP},
		Laravel: []config.LaravelConfig{
			{Name: "healthy", Path: dir, Queues: map[string][]string{"redis": {"default"}}},
			{Name: "broken", Path: dir, Queues: map[string][]string{"redis": {"default"}},
				PHPConfig: &config.PHPConfig{Binary: filepath.Join(dir, "does-not-exist")}},
		},
	}

	result, errs := Collect(context.Background(), cfg)

	if _, ok := result["healthy"]; !ok {
		t.Errorf("Expected the healthy site to still report, got %v", result)
	}

	if _, ok := result["broken"]; ok {
		t.Errorf("Expected no metrics for the broken site")
	}

	if _, ok := errs["laravel:broken"]; !ok {
		t.Errorf("Expected an error keyed to the broken site, got %v", errs)
	}
}
