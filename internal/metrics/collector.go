package metrics

import (
	"context"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/cboxdk/fpm-exporter/internal/laravel"
	"github.com/cboxdk/fpm-exporter/internal/phpfpm"
	"github.com/cboxdk/fpm-exporter/internal/server"
)

func GetMetrics(ctx context.Context, cfg *config.Config) (*Metrics, error) {
	out := &Metrics{
		Timestamp: time.Now(),
		Errors:    make(map[string]string),
	}

	systemInfoData := server.DetectSystem()
	out.Server = systemInfoData.SystemInfo
	for k, v := range systemInfoData.Errors {
		out.Errors[k] = v
	}

	if cfg.PHPFpm.Enabled {
		fpmResults, err := phpfpm.GetMetrics(ctx, cfg)
		if err != nil {
			out.Errors["fpm"] = err.Error()
		} else {
			out.Fpm = fpmResults
		}
	}

	if len(cfg.Laravel) > 0 {
		data, errs := laravel.Collect(ctx, cfg)
		for key, msg := range errs {
			out.Errors[key] = msg
		}
		out.Laravel = make(map[string]*laravel.LaravelMetrics)
		for name, metrics := range data {
			m := metrics // capture loop variable
			out.Laravel[name] = &m
		}
	}

	return out, nil
}
