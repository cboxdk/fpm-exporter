package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/cboxdk/fpm-exporter/internal/logging"
	"github.com/cboxdk/fpm-exporter/internal/phpfpm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// defaultSiteName labels a Laravel site the operator did not name.
const defaultSiteName = "App"

// Version that is being reported by the CLI
var Version string

var Config *config.Config

var (
	laravelShorthand  string
	laravelSiteFlags  []string
	laravelConfigFile string
)

var rootCmd = &cobra.Command{
	Use:   "fpm-exporter",
	Short: "Cbox FPM Exporter for monitoring PHP-FPM",
	Long:  `fpm-exporter is a lightweight PHP-FPM metrics exporter for Prometheus`,
	// A bad config path is not a usage error: printing twelve lines of flags
	// after it buries the actual message. cobra also printed the error itself,
	// so every failure appeared twice under two different prefixes.
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Read config file if specified
		if path := viper.GetString("config"); path != "" {
			viper.SetConfigFile(path)
			if err := viper.ReadInConfig(); err != nil {
				return fmt.Errorf("failed to read config file %s: %w", path, err)
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
		logging.L().Debug("Cbox Logging initialized", "level", Config.Logging.Level)
		logging.L().Debug("Cbox Loaded config", "config", Config)

		// Warned after logging is configured, so it lands in the operator's
		// chosen format. The documented quickstart --
		// `--laravel MyApp:/var/www/html` -- sets neither queues nor app info,
		// so it configures a site that collects nothing and used to say nothing
		// about it.
		for _, site := range Config.Laravel {
			if len(site.Queues) == 0 && !site.EnableAppInfo {
				logging.L().Warn("Cbox Laravel site will collect no metrics",
					"site", site.Name,
					"hint", "set appinfo=true (enable_app_info in YAML) or configure queues")
			}
		}

		// Autodiscovery is collection work: it scans the process table, runs
		// `php-fpm -tt` per master and probes PHP binaries. `version` used to
		// pay all of it -- a measured 11 seconds, ending in an ERROR about a
		// subsystem the command does not use.
		if !discoversPools(cmd) {
			return nil
		}

		// phpfpm autodiscover
		if Config.PHPFpm.Enabled && Config.PHPFpm.Autodiscover {
			var discovered []phpfpm.DiscoveredFPM
			var err error

			for i := 0; i < Config.PHPFpm.Retries; i++ {
				discovered, err = phpfpm.DiscoverFPMProcesses()
				if err == nil && len(discovered) > 0 {
					break
				}

				logging.L().Debug("Cbox PHP-FPM autodiscover attempt failed", "attempt", i+1, "error", err)
				time.Sleep(time.Duration(Config.PHPFpm.RetryDelay) * time.Second)
			}

			if err != nil {
				logging.L().Error("Cbox PHP-FPM Autodiscover failed after retries", "error", err)
			} else if len(discovered) == 0 {
				logging.L().Error("Cbox PHP-FPM Autodiscover succeeded but no FPM pools found")
			} else {
				logging.L().Debug("Cbox Discovered PHP-FPM Processes", "pools", discovered)
				for _, d := range discovered {
					Config.PHPFpm.Pools = append(Config.PHPFpm.Pools, config.FPMPoolConfig{
						Name:         d.Name,
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

// discoversPools reports whether a command actually collects metrics, and so
// needs the pool list resolved before it runs.
func discoversPools(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "serve" {
			return true
		}
	}
	return false
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
	if envSites := os.Getenv("CBOX_LARAVEL_SITES"); envSites != "" {
		var envParsed []config.LaravelConfig
		if err := json.Unmarshal([]byte(envSites), &envParsed); err != nil {
			return nil, fmt.Errorf("failed to parse CBOX_LARAVEL_SITES: %w", err)
		}
		sites = mergeSites(sites, envParsed)
	}

	if envConfig := os.Getenv("CBOX_LARAVEL_CONFIG"); envConfig != "" && laravelConfigFile == "" {
		fileSites, err := parseConfigFile(envConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CBOX_LARAVEL_CONFIG file: %w", err)
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
		name = defaultSiteName
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

// parseRepeatableFlags parses --laravel-site key=value flags into sites.
//
// A site ends where the next one begins, and only `name` or a repeated `path`
// can begin one. Writing the keys in the natural path-first order used to
// silently produce a site called "App" with everything after the dropped `name`
// discarded -- and `site` is a Prometheus label, so dashboards keyed on the name
// you asked for came back empty with no error anywhere.
func parseRepeatableFlags(flags []string) ([]config.LaravelConfig, error) {
	var sites []config.LaravelConfig

	newSite := func() config.LaravelConfig {
		return config.LaravelConfig{Queues: map[string][]string{}}
	}

	currentSite := newSite()
	// Tracks whether the current group has seen a name, so a name arriving
	// after a path can be reported rather than silently reassigned.
	named := false

	flush := func() {
		if currentSite.Path != "" || currentSite.Name != "" || len(currentSite.Queues) > 0 {
			sites = append(sites, currentSite)
		}
		currentSite = newSite()
		named = false
	}

	for _, flag := range flags {
		kv := strings.SplitN(flag, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid --laravel-site format (expected key=value): %s", flag)
		}

		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		// Nested queue config: queues.redis=default,emails
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
			if named {
				// A second name starts the next site.
				flush()
			} else if currentSite.Path != "" {
				return nil, fmt.Errorf(
					"--laravel-site name=%s came after path=%s; put name first, "+
						"or the keys cannot be grouped into sites unambiguously", val, currentSite.Path)
			}
			currentSite.Name = val
			named = true
		case "path":
			if currentSite.Path != "" {
				// A second path starts the next site.
				flush()
			}
			currentSite.Path = val
		case "appinfo":
			currentSite.EnableAppInfo = val == "true" || val == "1"
		default:
			return nil, fmt.Errorf("unknown Laravel config key: %s", key)
		}
	}

	flush()

	for i := range sites {
		if sites[i].Path == "" {
			return nil, fmt.Errorf("--laravel-site group %q has no path", sites[i].Name)
		}
		if sites[i].Name == "" {
			sites[i].Name = defaultSiteName
		}
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
			site.Name = defaultSiteName
		}
		siteMap[site.Name] = site
	}

	// Override with new sites, field by field. Replacing the whole struct meant
	// that adding a seemingly redundant `--laravel App:/same/path` on top of a
	// working config file silently deleted every queue it was monitoring.
	for _, site := range override {
		if site.Name == "" {
			site.Name = defaultSiteName
		}
		if existing, ok := siteMap[site.Name]; ok {
			siteMap[site.Name] = mergeSite(existing, site)
			continue
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

// mergeSite layers a higher-precedence site over a lower one, keeping any field
// the higher one did not set.
func mergeSite(base, override config.LaravelConfig) config.LaravelConfig {
	merged := base

	if override.Path != "" {
		merged.Path = override.Path
	}
	if override.EnableAppInfo {
		merged.EnableAppInfo = true
	}
	if override.PHPConfig != nil {
		merged.PHPConfig = override.PHPConfig
	}
	if override.Timeout > 0 {
		merged.Timeout = override.Timeout
	}
	if len(override.Queues) > 0 {
		merged.Queues = override.Queues
	}

	return merged
}

// validateSites validates all Laravel sites, reporting everything wrong at once
// rather than one failure per run -- with several sites and a config error in
// each, fixing them one at a time means one restart per typo.
func validateSites(sites []config.LaravelConfig) error {
	seenNames := map[string]bool{}
	var problems []error

	for i, site := range sites {
		if site.Name == "" {
			problems = append(problems, fmt.Errorf("laravel site at index %d has no name", i))
			continue
		}

		if seenNames[site.Name] {
			problems = append(problems, fmt.Errorf("duplicate laravel site name: %s", site.Name))
			continue
		}
		seenNames[site.Name] = true

		if site.Path == "" {
			problems = append(problems, fmt.Errorf("laravel site %q has no path", site.Name))
			continue
		}

		if _, err := os.Stat(site.Path); os.IsNotExist(err) {
			problems = append(problems, fmt.Errorf("laravel site %q path does not exist: %s", site.Name, site.Path))
			continue
		}

		artisanPath := filepath.Join(site.Path, "artisan")
		if _, err := os.Stat(artisanPath); os.IsNotExist(err) {
			problems = append(problems, fmt.Errorf("laravel site %q path does not contain an artisan file: %s", site.Name, site.Path))
		}
	}

	return errors.Join(problems...)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra's own printing is silenced, so this is the single place an
		// error reaches the operator.
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
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

	viper.SetEnvPrefix("CBOX")
	viper.AutomaticEnv()
}
