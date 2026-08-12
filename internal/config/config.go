package config

import (
	"fmt"
	"strings"

	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Debug   bool            `mapstructure:"debug"`
	Logging LoggingBlock    `mapstructure:"logging"`
	PHPFpm  FPMConfig       `mapstructure:"phpfpm"`
	PHP     PHPConfig       `mapstructure:"php"`
	Monitor MonitorConfig   `mapstructure:"monitor"`
	Laravel []LaravelConfig `mapstructure:"laravel"`
}

type LoggingBlock struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // text, json
	Color  bool   `mapstructure:"color"`  // enable ANSI colors in text format
}

// PHPConfig is decoded by viper (mapstructure) from the main config file, and by
// yaml/json when it arrives through --laravel-config or CBOX_LARAVEL_SITES.
type PHPConfig struct {
	Binary  string `mapstructure:"binary" yaml:"binary" json:"binary"`
	IniPath string `mapstructure:"ini_path" yaml:"ini_path" json:"ini_path"`
}

type FPMConfig struct {
	Enabled      bool            `mapstructure:"enabled"`
	Autodiscover bool            `mapstructure:"autodiscover"`
	Retries      int             `mapstructure:"retries"`
	RetryDelay   int             `mapstructure:"retry_delay"`
	Pools        []FPMPoolConfig `mapstructure:"pools"`
}

type FPMPoolConfig struct {
	// Name identifies the pool in metrics when the pool itself could not be
	// reached. A successful scrape uses the name PHP-FPM reports; this is the
	// fallback so a failing pool is still labelled with something meaningful.
	Name              string `mapstructure:"name"`
	Socket            string `mapstructure:"socket"`
	StatusSocket      string `mapstructure:"status_socket"`
	StatusPath        string `mapstructure:"status_path"`
	StatusPathEnabled bool   `mapstructure:"status_path_enabled"`
	ConfigPath        string `mapstructure:"config_path"`
	Binary            string `mapstructure:"binary"`
	CliBinary         string `mapstructure:"cli_binary"`
	// Timeout bounds the FastCGI dial for this pool. Defaults to 3s.
	Timeout time.Duration `mapstructure:"timeout"`
}

// normalize fills in the defaults a hand-written pool is entitled to assume.
// Autodiscovery already sets both sockets; manual configuration did not, so the
// documented example -- socket plus status_path -- collected nothing at all,
// because collection dials StatusSocket exclusively.
func (p *FPMPoolConfig) normalize() {
	if p.StatusSocket == "" {
		p.StatusSocket = p.Socket
	}
	if p.StatusPath == "" {
		p.StatusPath = "/status"
	}
}

// LaravelConfig is decoded from three different sources, each with its own tag
// convention: the main config file via viper (mapstructure), a --laravel-config /
// CBOX_LARAVEL_CONFIG file via yaml, and CBOX_LARAVEL_SITES via json. All three
// sets of tags must stay in sync, or a documented key is silently dropped on the
// sources whose tag is missing.
type LaravelConfig struct {
	Name          string              `mapstructure:"name" yaml:"name" json:"name"`                                  // Optional name for identification
	Path          string              `mapstructure:"path" yaml:"path" json:"path"`                                  // Root path to Laravel app
	EnableAppInfo bool                `mapstructure:"enable_app_info" yaml:"enable_app_info" json:"enable_app_info"` // Collect `php artisan about` metrics
	PHPConfig     *PHPConfig          `mapstructure:"php_config" yaml:"php_config" json:"php_config"`                // Optional override of global PHP config
	Queues        map[string][]string `mapstructure:"queues" yaml:"queues" json:"queues"`                            // Map of connection name to list of queue names
	// Timeout bounds each artisan invocation for this site. Falls back to
	// DefaultLaravelTimeout.
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" json:"timeout"`
}

// DefaultLaravelTimeout bounds a single artisan invocation. Booting a Laravel
// app is slow, so this is generous compared to the FPM socket timeouts.
const DefaultLaravelTimeout = 10 * time.Second

type MonitorConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
	EnableJson bool   `mapstructure:"enable_json"`
	// ScrapeTimeout bounds a single /metrics or /json collection. Keep it below
	// Prometheus' own scrape_timeout.
	ScrapeTimeout time.Duration `mapstructure:"scrape_timeout"`
}

// envBoundKeys are the configuration keys settable through CBOX_* environment
// variables. Each one is covered by a test asserting the documented variable
// actually lands, so the table here and the table in the docs cannot drift.
var envBoundKeys = []string{
	"debug",
	"logging.level",
	"logging.format",
	"logging.color",
	"monitor.listen_addr",
	"monitor.enable_json",
	"monitor.scrape_timeout",
	"php.binary",
	"php.ini_path",
	"phpfpm.enabled",
	"phpfpm.autodiscover",
	"phpfpm.retries",
	"phpfpm.retry_delay",
}

func Load() (*Config, error) {
	viper.SetDefault("debug", false)

	viper.SetDefault("phpfpm.enabled", true)
	viper.SetDefault("phpfpm.autodiscover", true)
	viper.SetDefault("phpfpm.retries", 5)
	viper.SetDefault("phpfpm.retry_delay", 2)
	viper.SetDefault("phpfpm.pools", []FPMPoolConfig{})

	viper.SetDefault("php.binary", "php")

	viper.SetDefault("monitor.listen_addr", ":9114")
	viper.SetDefault("monitor.enable_json", true)
	viper.SetDefault("monitor.scrape_timeout", "15s")

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.color", true)

	viper.SetDefault("laravel", []LaravelConfig{})
	// No default queue config, expected to be provided per site if needed

	viper.SetEnvPrefix("CBOX")
	viper.AutomaticEnv()

	// AutomaticEnv alone looks up the nested key monitor.listen_addr as
	// CBOX_MONITOR.LISTEN_ADDR, which is not a settable variable name -- so
	// every documented nested variable silently did nothing. The replacer maps
	// the dots, and the explicit binds make each key work through Unmarshal,
	// which AutomaticEnv does not reliably do on its own.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	for _, key := range envBoundKeys {
		if err := viper.BindEnv(key); err != nil {
			return nil, fmt.Errorf("failed to bind %s: %w", key, err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	for i := range cfg.PHPFpm.Pools {
		cfg.PHPFpm.Pools[i].normalize()
	}

	return &cfg, nil
}
