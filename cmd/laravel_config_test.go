package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gophpeek/phpeek-fpm-exporter/internal/config"
)

func TestParseShorthand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantPath  string
		wantError bool
	}{
		{
			name:      "Name and path",
			input:     "MyApp:/var/www/html",
			wantName:  "MyApp",
			wantPath:  "/var/www/html",
			wantError: false,
		},
		{
			name:      "Path only (default name)",
			input:     "/var/www/html",
			wantName:  "App",
			wantPath:  "/var/www/html",
			wantError: false,
		},
		{
			name:      "Empty path",
			input:     "MyApp:",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site, err := parseShorthand(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("parseShorthand() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if site.Name != tt.wantName {
					t.Errorf("parseShorthand() name = %v, want %v", site.Name, tt.wantName)
				}
				if site.Path != tt.wantPath {
					t.Errorf("parseShorthand() path = %v, want %v", site.Path, tt.wantPath)
				}
			}
		})
	}
}

func TestParseRepeatableFlags(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantSites int
		wantError bool
	}{
		{
			name: "Single site with basic params",
			input: []string{
				"name=App",
				"path=/var/www/html",
			},
			wantSites: 1,
			wantError: false,
		},
		{
			name: "Single site with queues",
			input: []string{
				"name=App",
				"path=/var/www/html",
				"queues.redis=default,emails",
			},
			wantSites: 1,
			wantError: false,
		},
		{
			name: "Multiple sites",
			input: []string{
				"name=App",
				"path=/var/www/html",
				"name=Admin",
				"path=/var/www/admin",
			},
			wantSites: 2,
			wantError: false,
		},
		{
			name: "Invalid format",
			input: []string{
				"invalid_no_equals",
			},
			wantError: true,
		},
		{
			name: "Unknown key",
			input: []string{
				"unknown_key=value",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sites, err := parseRepeatableFlags(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("parseRepeatableFlags() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && len(sites) != tt.wantSites {
				t.Errorf("parseRepeatableFlags() got %d sites, want %d", len(sites), tt.wantSites)
			}
		})
	}
}

func TestParseRepeatableFlagsQueues(t *testing.T) {
	flags := []string{
		"name=App",
		"path=/var/www/html",
		"queues.redis=default,emails",
		"queues.database=jobs",
	}

	sites, err := parseRepeatableFlags(flags)
	if err != nil {
		t.Fatalf("parseRepeatableFlags() unexpected error: %v", err)
	}

	if len(sites) != 1 {
		t.Fatalf("Expected 1 site, got %d", len(sites))
	}

	site := sites[0]
	if len(site.Queues) != 2 {
		t.Errorf("Expected 2 queue connections, got %d", len(site.Queues))
	}

	redisQueues, ok := site.Queues["redis"]
	if !ok {
		t.Error("Expected redis connection in queues")
	} else if len(redisQueues) != 2 {
		t.Errorf("Expected 2 redis queues, got %d", len(redisQueues))
	} else {
		if redisQueues[0] != "default" || redisQueues[1] != "emails" {
			t.Errorf("Unexpected redis queue names: %v", redisQueues)
		}
	}

	dbQueues, ok := site.Queues["database"]
	if !ok {
		t.Error("Expected database connection in queues")
	} else if len(dbQueues) != 1 || dbQueues[0] != "jobs" {
		t.Errorf("Unexpected database queues: %v", dbQueues)
	}
}

func TestParseConfigFile(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "laravel.yaml")

	yamlContent := `laravel:
  - name: App
    path: /var/www/html
    queues:
      redis:
        - default
        - emails
  - name: Admin
    path: /var/www/admin
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	sites, err := parseConfigFile(configPath)
	if err != nil {
		t.Fatalf("parseConfigFile() unexpected error: %v", err)
	}

	if len(sites) != 2 {
		t.Errorf("Expected 2 sites, got %d", len(sites))
	}

	if sites[0].Name != "App" || sites[0].Path != "/var/www/html" {
		t.Errorf("Unexpected site 0: %+v", sites[0])
	}

	if sites[1].Name != "Admin" || sites[1].Path != "/var/www/admin" {
		t.Errorf("Unexpected site 1: %+v", sites[1])
	}
}

func TestMergeSites(t *testing.T) {
	base := []config.LaravelConfig{
		{Name: "App", Path: "/var/www/html"},
		{Name: "Admin", Path: "/var/www/admin"},
	}

	override := []config.LaravelConfig{
		{Name: "App", Path: "/custom/path"}, // Override existing
		{Name: "Api", Path: "/var/www/api"}, // Add new
	}

	result := mergeSites(base, override)

	// Should have 3 sites total (App overridden, Admin kept, Api added)
	if len(result) != 3 {
		t.Fatalf("Expected 3 sites, got %d", len(result))
	}

	// Check that App was overridden
	found := false
	for _, site := range result {
		if site.Name == "App" {
			found = true
			if site.Path != "/custom/path" {
				t.Errorf("App site not overridden, path = %s", site.Path)
			}
		}
	}
	if !found {
		t.Error("App site not found in result")
	}
}

func TestValidateSites(t *testing.T) {
	// Create temp directories for testing
	tmpDir := t.TempDir()
	validPath := filepath.Join(tmpDir, "app1")
	_ = os.MkdirAll(validPath, 0755)
	_ = os.WriteFile(filepath.Join(validPath, "artisan"), []byte("#!/usr/bin/env php"), 0644)

	missingArtisan := filepath.Join(tmpDir, "app2")
	_ = os.MkdirAll(missingArtisan, 0755)

	tests := []struct {
		name      string
		sites     []config.LaravelConfig
		wantError bool
	}{
		{
			name: "Valid site",
			sites: []config.LaravelConfig{
				{Name: "App", Path: validPath},
			},
			wantError: false,
		},
		{
			name: "Missing name",
			sites: []config.LaravelConfig{
				{Name: "", Path: validPath},
			},
			wantError: true,
		},
		{
			name: "Missing path",
			sites: []config.LaravelConfig{
				{Name: "App", Path: ""},
			},
			wantError: true,
		},
		{
			name: "Path does not exist",
			sites: []config.LaravelConfig{
				{Name: "App", Path: "/nonexistent/path"},
			},
			wantError: true,
		},
		{
			name: "Path missing artisan",
			sites: []config.LaravelConfig{
				{Name: "App", Path: missingArtisan},
			},
			wantError: true,
		},
		{
			name: "Duplicate names",
			sites: []config.LaravelConfig{
				{Name: "App", Path: validPath},
				{Name: "App", Path: validPath},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSites(tt.sites)
			if (err != nil) != tt.wantError {
				t.Errorf("validateSites() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestParseLaravelSites_EnvVar(t *testing.T) {
	// Save and restore original env vars
	originalSites := os.Getenv("PHPEEK_LARAVEL_SITES")
	defer func() {
		if originalSites != "" {
			os.Setenv("PHPEEK_LARAVEL_SITES", originalSites)
		} else {
			os.Unsetenv("PHPEEK_LARAVEL_SITES")
		}
	}()

	// Create temp directory for testing
	tmpDir := t.TempDir()
	appPath := filepath.Join(tmpDir, "app")
	_ = os.MkdirAll(appPath, 0755)
	_ = os.WriteFile(filepath.Join(appPath, "artisan"), []byte("#!/usr/bin/env php"), 0644)

	sites := []config.LaravelConfig{
		{Name: "App", Path: appPath, Queues: map[string][]string{}},
	}

	jsonData, err := json.Marshal(sites)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	os.Setenv("PHPEEK_LARAVEL_SITES", string(jsonData))

	// Reset CLI flags
	laravelShorthand = ""
	laravelSiteFlags = nil
	laravelConfigFile = ""

	result, err := parseLaravelSites()
	if err != nil {
		t.Fatalf("parseLaravelSites() unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 site from env var, got %d", len(result))
	}
}

func TestParseLaravelSites_Priority(t *testing.T) {
	// Create temp paths
	tmpDir := t.TempDir()
	appPath := filepath.Join(tmpDir, "app")
	_ = os.MkdirAll(appPath, 0755)
	_ = os.WriteFile(filepath.Join(appPath, "artisan"), []byte("#!/usr/bin/env php"), 0644)

	// Create config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `laravel:
  - name: App
    path: ` + appPath + `
    queues:
      redis:
        - from_file
`
	_ = os.WriteFile(configPath, []byte(yamlContent), 0644)

	// Set config file
	laravelConfigFile = configPath

	// Set CLI flags (should override file)
	laravelSiteFlags = []string{
		"name=App",
		"path=" + appPath,
		"queues.redis=from_cli",
	}

	result, err := parseLaravelSites()
	if err != nil {
		t.Fatalf("parseLaravelSites() unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 site, got %d", len(result))
	}

	// CLI should override file
	if result[0].Queues["redis"][0] != "from_cli" {
		t.Errorf("CLI flags did not override config file, got queue: %v", result[0].Queues["redis"])
	}

	// Reset
	laravelConfigFile = ""
	laravelSiteFlags = nil
}
