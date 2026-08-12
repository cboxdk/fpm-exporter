package laravel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/cboxdk/fpm-exporter/internal/logging"
)

type BoolString bool

func (b *BoolString) UnmarshalJSON(data []byte) error {
	var asBool bool
	if err := json.Unmarshal(data, &asBool); err == nil {
		*b = BoolString(asBool)
		return nil
	}
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		switch strings.ToLower(asString) {
		case "enabled", "true", "on", "yes", "cached":
			*b = true
		default:
			*b = false
		}
		return nil
	}
	return fmt.Errorf("invalid boolean value: %s", string(data))
}

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*s = []string{single}
		return nil
	}
	var slice []string
	if err := json.Unmarshal(data, &slice); err == nil {
		*s = slice
		return nil
	}
	return fmt.Errorf("invalid value for StringOrSlice: %s", string(data))
}

type AppInfo struct {
	Environment struct {
		ApplicationName *string     `json:"application_name"`
		LaravelVersion  *string     `json:"laravel_version"`
		PHPVersion      *string     `json:"php_version"`
		ComposerVersion *string     `json:"composer_version"`
		Environment     *string     `json:"environment"`
		DebugMode       *BoolString `json:"debug_mode"`
		URL             *string     `json:"url"`
		MaintenanceMode *BoolString `json:"maintenance_mode"`
		Timezone        *string     `json:"timezone"`
		Locale          *string     `json:"locale"`
	} `json:"environment"`

	Cache struct {
		Config BoolString `json:"config"`
		Events BoolString `json:"events"`
		Routes BoolString `json:"routes"`
		Views  BoolString `json:"views"`
	} `json:"cache"`

	Drivers struct {
		Broadcasting *string        `json:"broadcasting"`
		Cache        *string        `json:"cache"`
		Database     *string        `json:"database"`
		Logs         *StringOrSlice `json:"logs"`
		Mail         *string        `json:"mail"`
		Queue        *string        `json:"queue"`
		Session      *string        `json:"session"`
	} `json:"drivers"`

	Livewire *map[string]string `json:"livewire,omitempty"`
}

// `artisan about` is expensive, so results are cached -- but only for a while.
// A permanent cache meant that an app which was down when the exporter started
// never reported app info again until the process was restarted, and a single
// timeout poisoned the site for good.
const (
	appInfoTTL        = 5 * time.Minute
	appInfoFailureTTL = 30 * time.Second
)

type appInfoEntry struct {
	info      *AppInfo
	expiresAt time.Time
}

var (
	appInfoCache = make(map[string]appInfoEntry)
	cacheMutex   sync.RWMutex
)

func cacheAppInfo(key string, info *AppInfo, ttl time.Duration) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	appInfoCache[key] = appInfoEntry{info: info, expiresAt: time.Now().Add(ttl)}
}

func GetAppInfo(ctx context.Context, site config.LaravelConfig, phpBinary string) (*AppInfo, error) {
	if !site.EnableAppInfo {
		return nil, nil
	}

	if phpBinary == "" || site.Path == "" {
		return nil, fmt.Errorf("invalid input: phpBinary and path are required")
	}

	cacheKey := filepath.Clean(site.Path)

	cacheMutex.RLock()
	entry, ok := appInfoCache[cacheKey]
	cacheMutex.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		if entry.info == nil {
			return nil, fmt.Errorf("app info recently failed for %s, not retrying yet", cacheKey)
		}
		return entry.info, nil
	}

	logging.L().Debug("Cbox Uncached app info. Calling artisan about", "path", site.Path)

	cmd := exec.CommandContext(ctx, phpBinary, "-d", "error_reporting=E_ALL & ~E_DEPRECATED", "artisan", "about", "--json")

	// disable monitoring on scraping to prevent exhausting monitoring tools
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NIGHTWATCH_ENABLED=false")
	cmd.Env = append(cmd.Env, "TELESCOPE_ENABLED=false")
	cmd.Env = append(cmd.Env, "NEW_RELIC_ENABLED=false")
	cmd.Env = append(cmd.Env, "BUGSNAG_API_KEY=null")
	cmd.Env = append(cmd.Env, "SENTRY_LARAVEL_DSN=null")
	cmd.Env = append(cmd.Env, "ROLLBAR_TOKEN=null")

	cmd.Dir = cacheKey

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		cacheAppInfo(cacheKey, nil, appInfoFailureTTL)
		if ctx.Err() != nil {
			return nil, fmt.Errorf("artisan about timed out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("artisan about failed: %w\nOutput: %s", err, out.String())
	}

	var parsed AppInfo
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		cacheAppInfo(cacheKey, nil, appInfoFailureTTL)
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	cacheAppInfo(cacheKey, &parsed, appInfoTTL)

	return &parsed, nil
}
