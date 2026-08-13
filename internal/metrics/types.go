package metrics

import (
	"github.com/cboxdk/fpm-exporter/internal/laravel"
	"github.com/cboxdk/fpm-exporter/internal/phpfpm"
	"github.com/cboxdk/fpm-exporter/internal/server"
	"time"
)

type Metrics struct {
	Timestamp time.Time
	Server    *server.SystemInfo
	Fpm       map[string]*phpfpm.Result
	// FpmPools carries the per-pool outcome, including the pools that failed.
	// Kept off the JSON payload: Fpm is the published shape.
	FpmPools []phpfpm.PoolOutcome               `json:"-"`
	Laravel  map[string]*laravel.LaravelMetrics `json:"laravel,omitempty"`
	Errors   map[string]string
}
