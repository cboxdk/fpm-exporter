package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gophpeek/phpeek-fpm-exporter/internal/config"
	"github.com/gophpeek/phpeek-fpm-exporter/internal/logging"
	"github.com/gophpeek/phpeek-fpm-exporter/internal/phpfpm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Version that is being reported by the CLI
var Version string

var Config *config.Config

var (
	laravelShorthand   string
	laravelSiteFlags   []string
	laravelConfigFile  string
)

var rootCmd = &cobra.Command{
	Use:   "phpeek-fpm-exporter",
	Short: "PHPeek PHP-FPM Exporter for monitoring PHP-FPM",
	Long:  `phpeek-fpm-exporter is a lightweight PHP-FPM metrics exporter for Prometheus`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Read config file if specified
		if path := viper.GetString("config"); path != "" {
			viper.SetConfigFile(path)
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
		}

		loaded, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Parse Laravel sites from all sources (priority: CLI > env > config file)
		sites, err := parseLaravelSites()
		if err != nil {
			return fmt.Errorf("failed to parse Laravel configuration: %w", err)
		}
		if len(sites) > 0 {
			loaded.Laravel = sites
		}

		// Handle log level (priority: flag > config > debug)
		if lvl, _ := cmd.Flags().GetString("log-level"); lvl != "" {
			loaded.Logging.Level = lvl
		} else if viper.GetBool("debug") || loaded.Debug {
			loaded.Logging.Level = "debug"
		}

		Config = loaded

		logging.Init(Config.Logging)
		logging.L().Debug("PHPeek Logging initialized", "level", Config.Logging.Level)
		logging.L().Debug("PHPeek Loaded config", "config", Config)

		// phpfpm autodiscover
		if Config.PHPFpm.Enabled && Config.PHPFpm.Autodiscover {
			var discovered []phpfpm.DiscoveredFPM
			var err error

			for i := 0; i < Config.PHPFpm.Retries; i++ {
				discovered, err = phpfpm.DiscoverFPMProcesses()
				if err == nil && len(discovered) > 0 {
					break
				}

				logging.L().Debug("PHPeek PHP-FPM autodiscover attempt failed", "attempt", i+1, "error", err)
				time.Sleep(time.Duration(Config.PHPFpm.RetryDelay) * time.Second)
			}

			if err != nil {
				logging.L().Error("PHPeek PHP-FPM Autodiscover failed after retries", "error", err)
			} else if len(discovered) == 0 {
				logging.L().Error("PHPeek PHP-FPM Autodiscover succeeded but no FPM pools found")
			} else {
				logging.L().Debug("PHPeek Discovered PHP-FPM Processes", "pools", discovered)
				for _, d := range discovered {
					Config.PHPFpm.Pools = append(Config.PHPFpm.Pools, config.FPMPoolConfig{
						Socket:       d.Socket,
						StatusSocket: d.StatusSocket,
						StatusPath:   d.StatusPath,
						ConfigPath:   d.ConfigPath,
						Binary:       d.Binary,
						CliBinary:    d.CliBinary,
					})
				}
			}
		}

		return nil
	},
}

// parseLaravelSites parses Laravel configuration from all sources
// Priority: CLI flags > Environment variables > Config file
func parseLaravelSites() ([]config.LaravelConfig, error) {
	var sites []config.LaravelConfig

	// 1. Parse from config file if specified
	if laravelConfigFile != "" {
		fileSites, err := parseConfigFile(laravelConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Laravel config file: %w", err)
		}
		sites = append(sites, fileSites...)
	}

	// 2. Parse from environment variable
	if envSites := os.Getenv("PHPEEK_LARAVEL_SITES"); envSites != "" {
		var envParsed []config.LaravelConfig
		if err := json.Unmarshal([]byte(envSites), &envParsed); err != nil {
			return nil, fmt.Errorf("failed to parse PHPEEK_LARAVEL_SITES: %w", err)
		}
		sites = mergeSites(sites, envParsed)
	}

	if envConfig := os.Getenv("PHPEEK_LARAVEL_CONFIG"); envConfig != "" && laravelConfigFile == "" {
		fileSites, err := parseConfigFile(envConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PHPEEK_LARAVEL_CONFIG file: %w", err)
		}
		sites = mergeSites(sites, fileSites)
	}

	// 3. Parse from shorthand CLI flag
	if laravelShorthand != "" {
		shortSite, err := parseShorthand(laravelShorthand)
		if err != nil {
			return nil, fmt.Errorf("failed to parse --laravel shorthand: %w", err)
		}
		sites = mergeSites(sites, []config.LaravelConfig{shortSite})
	}

	// 4. Parse from repeatable site flags (highest priority)
	if len(laravelSiteFlags) > 0 {
		flagSites, err := parseRepeatableFlags(laravelSiteFlags)
		if err != nil {
			return nil, fmt.Errorf("failed to parse --laravel-site flags: %w", err)
		}
		sites = mergeSites(sites, flagSites)
	}

	// Validate all sites
	if err := validateSites(sites); err != nil {
		return nil, err
	}

	return sites, nil
}

// parseShorthand parses the shorthand format: "name:path" or just "path"
func parseShorthand(shorthand string) (config.LaravelConfig, error) {
	parts := strings.SplitN(shorthand, ":", 2)

	var name, path string
	if len(parts) == 2 {
		name = parts[0]
		path = parts[1]
	} else {
		name = "App"
		path = parts[0]
	}

	if path == "" {
		return config.LaravelConfig{}, fmt.Errorf("path cannot be empty")
	}

	return config.LaravelConfig{
		Name:   name,
		Path:   path,
		Queues: map[string][]string{},
	}, nil
}

// parseRepeatableFlags parses --laravel-site key=value flags
func parseRepeatableFlags(flags []string) ([]config.LaravelConfig, error) {
	// Group flags by site (assuming sequential flags belong to same site)
	var sites []config.LaravelConfig
	currentSite := config.LaravelConfig{
		Queues: map[string][]string{},
	}

	for _, flag := range flags {
		kv := strings.SplitN(flag, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid --laravel-site format (expected key=value): %s", flag)
		}

		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		// Handle nested queue config: queues.redis=default,emails
		if strings.HasPrefix(key, "queues.") {
			connection := strings.TrimPrefix(key, "queues.")
			queueNames := strings.Split(val, ",")
			for i, q := range queueNames {
				queueNames[i] = strings.TrimSpace(q)
			}
			if currentSite.Queues == nil {
				currentSite.Queues = map[string][]string{}
			}
			currentSite.Queues[connection] = queueNames
			continue
		}

		switch key {
		case "name":
			// If we encounter a new "name" and current site has data, save it
			if currentSite.Path != "" {
				sites = append(sites, currentSite)
				currentSite = config.LaravelConfig{
					Queues: map[string][]string{},
				}
			}
			currentSite.Name = val
		case "path":
			currentSite.Path = val
		case "appinfo":
			currentSite.EnableAppInfo = val == "true" || val == "1"
		default:
			return nil, fmt.Errorf("unknown Laravel config key: %s", key)
		}
	}

	// Add last site if it has data
	if currentSite.Path != "" {
		sites = append(sites, currentSite)
	}

	return sites, nil
}

// parseConfigFile loads Laravel sites from a YAML file
func parseConfigFile(path string) ([]config.LaravelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg struct {
		Laravel []config.LaravelConfig `yaml:"laravel"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return cfg.Laravel, nil
}

// mergeSites merges two site lists, with 'override' taking precedence by name
func mergeSites(base, override []config.LaravelConfig) []config.LaravelConfig {
	siteMap := make(map[string]config.LaravelConfig)

	// Add base sites
	for _, site := range base {
		if site.Name == "" {
			site.Name = "App"
		}
		siteMap[site.Name] = site
	}

	// Override with new sites
	for _, site := range override {
		if site.Name == "" {
			site.Name = "App"
		}
		siteMap[site.Name] = site
	}

	// Convert back to slice
	result := make([]config.LaravelConfig, 0, len(siteMap))
	for _, site := range siteMap {
		result = append(result, site)
	}

	return result
}

// validateSites validates all Laravel sites
func validateSites(sites []config.LaravelConfig) error {
	seenNames := map[string]bool{}

	for i, site := range sites {
		// Ensure name is set
		if site.Name == "" {
			return fmt.Errorf("Laravel site at index %d missing name", i)
		}

		// Check for duplicate names
		if seenNames[site.Name] {
			return fmt.Errorf("duplicate Laravel site name: %s", site.Name)
		}
		seenNames[site.Name] = true

		// Validate path is set
		if site.Path == "" {
			return fmt.Errorf("Laravel site '%s' missing path", site.Name)
		}

		// Validate path exists
		if _, err := os.Stat(site.Path); os.IsNotExist(err) {
			return fmt.Errorf("Laravel site '%s' path does not exist: %s", site.Name, site.Path)
		}

		// Check if path looks like a Laravel app (has artisan)
		artisanPath := filepath.Join(site.Path, "artisan")
		if _, err := os.Stat(artisanPath); os.IsNotExist(err) {
			return fmt.Errorf("Laravel site '%s' path does not contain artisan file: %s", site.Name, site.Path)
		}
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Command execution failed:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().Bool("debug", false, "Debug mode")
	rootCmd.PersistentFlags().String("config", "", "config file path")
	rootCmd.PersistentFlags().Bool("autodiscover", true, "Autodiscover php-fpm pools")
	rootCmd.PersistentFlags().String("log-level", "", "Override log level (e.g. debug, info, warn)")

	// Laravel configuration flags
	rootCmd.PersistentFlags().StringVar(&laravelShorthand, "laravel", "", "Laravel site shorthand (name:path or just path)")
	rootCmd.PersistentFlags().StringArrayVar(&laravelSiteFlags, "laravel-site", nil, "Laravel site parameter (key=value). Repeat for multiple params")
	rootCmd.PersistentFlags().StringVar(&laravelConfigFile, "laravel-config", "", "Path to YAML file with Laravel site configurations")

	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	_ = viper.BindPFlag("phpfpm.autodiscover", rootCmd.PersistentFlags().Lookup("autodiscover"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))

	viper.SetEnvPrefix("PHPEEK")
	viper.AutomaticEnv()
}
