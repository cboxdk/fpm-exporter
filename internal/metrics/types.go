package metrics

import (
	"github.com/gophpeek/phpeek-fpm-exporter/internal/laravel"
	"github.com/gophpeek/phpeek-fpm-exporter/internal/phpfpm"
	"github.com/gophpeek/phpeek-fpm-exporter/internal/server"
	"time"
)

type Metrics struct {
	Timestamp time.Time
	Server    *server.SystemInfo
	Fpm       map[string]*phpfpm.Result
	Laravel   map[string]*laravel.LaravelMetrics `json:"laravel,omitempty"`
	Errors    map[string]string
}
