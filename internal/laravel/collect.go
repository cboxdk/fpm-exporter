package laravel

import (
	"context"
	"runtime"
	"sync"

	"github.com/cboxdk/fpm-exporter/internal/config"
)

type LaravelMetrics struct {
	Queues *QueueSizes `json:"queues"`
	Info   *AppInfo    `json:"app_info"`
}

type siteResult struct {
	name    string
	metrics LaravelMetrics
	errors  map[string]string
}

// Collect gathers Laravel queue metrics for all configured sites.
//
// Sites are collected concurrently. Each one boots a PHP process (two when app
// info is enabled), so collecting them in sequence made the scrape cost the sum
// of every site's boot time — with a handful of sites that alone can exceed the
// scrape budget.
func Collect(ctx context.Context, cfg *config.Config) (map[string]LaravelMetrics, map[string]string) {
	result := make(map[string]LaravelMetrics, len(cfg.Laravel))
	errors := make(map[string]string)

	results := make(chan siteResult, len(cfg.Laravel))
	var wg sync.WaitGroup

	// Bounded, because every site in flight is a PHP process resident in
	// memory. Unbounded fan-out would let a host with many sites fork a small
	// army on each scrape.
	slots := make(chan struct{}, max(1, runtime.NumCPU()))

	for _, site := range cfg.Laravel {
		wg.Add(1)
		go func(site config.LaravelConfig) {
			defer wg.Done()

			slots <- struct{}{}
			defer func() { <-slots }()

			results <- collectSite(ctx, cfg, site)
		}(site)
	}

	wg.Wait()
	close(results)

	for res := range results {
		for key, msg := range res.errors {
			errors[key] = msg
		}
		if res.name != "" {
			result[res.name] = res.metrics
		}
	}

	return result, errors
}

func collectSite(ctx context.Context, cfg *config.Config, site config.LaravelConfig) siteResult {
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
	defer cancel()

	res := siteResult{errors: map[string]string{}}

	queues, err := GetQueueSizes(siteCtx, site.Path, php, site.Queues)
	if err != nil {
		res.errors["laravel:"+site.Name] = err.Error()
		return res
	}

	info, err := GetAppInfo(siteCtx, site, php)
	if err != nil {
		res.errors["laravel:"+site.Name+":info"] = err.Error()
	}

	res.name = site.Name
	res.metrics = LaravelMetrics{Queues: queues, Info: info}

	return res
}
