package laravel

import (
	"context"
	"github.com/cboxdk/fpm-exporter/internal/config"
)

type LaravelMetrics struct {
	Queues *QueueSizes `json:"queues"`
	Info   *AppInfo    `json:"app_info"`
}

// Collect gathers Laravel queue metrics for all configured sites.
func Collect(ctx context.Context, cfg *config.Config) (map[string]LaravelMetrics, map[string]string) {
	result := make(map[string]LaravelMetrics)
	errors := make(map[string]string)

	for _, site := range cfg.Laravel {
		php := cfg.PHP.Binary
		if site.PHPConfig != nil && site.PHPConfig.Binary != "" {
			php = site.PHPConfig.Binary
		}

		// Booting Laravel can hang on an unreachable Redis or a locked table.
		// Without a bound, the artisan call outlives the scrape it belongs to
		// and requests pile up behind it.
		timeout := site.Timeout
		if timeout <= 0 {
			timeout = config.DefaultLaravelTimeout
		}
		siteCtx, cancel := context.WithTimeout(ctx, timeout)

		queues, err := GetQueueSizes(siteCtx, site.Path, php, site.Queues)
		if err != nil {
			cancel()
			errors["laravel:"+site.Name] = err.Error()
			continue
		}

		info, err := GetAppInfo(siteCtx, site, php)
		cancel()
		if err != nil {
			errors["laravel:"+site.Name+":info"] = err.Error()
		}

		if info != nil {
			result[site.Name] = LaravelMetrics{
				Queues: queues,
				Info:   info,
			}
		} else {
			result[site.Name] = LaravelMetrics{
				Queues: queues,
			}
		}
	}

	return result, errors
}
