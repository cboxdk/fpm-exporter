package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"github.com/cboxdk/fpm-exporter/internal/config"
)

// logger is read from every collection path, including goroutines, so it is
// swapped atomically rather than assigned. It is never nil: L() falling back to
// the slog default means a package that logs before Init cannot panic the
// process, which is otherwise a real hazard on error paths that only run in
// production.
var logger atomic.Pointer[slog.Logger]

func Init(cfg config.LoggingBlock) {
	var lvl slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}

	l := slog.New(handler)
	logger.Store(l)
	slog.SetDefault(l)
}

func L() *slog.Logger {
	if l := logger.Load(); l != nil {
		return l
	}
	return slog.Default()
}
