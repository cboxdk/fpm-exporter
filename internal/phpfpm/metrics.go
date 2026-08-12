package phpfpm

import (
	"context"
	"fmt"
	"github.com/cboxdk/fpm-exporter/internal/logging"
	"slices"
	"strings"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/elasticphphq/fcgx"
)

type PoolProcess struct {
	PID               int     `json:"pid"`
	State             string  `json:"state"`
	StartTime         int64   `json:"start time"`
	StartSince        int64   `json:"start since"`
	Requests          int64   `json:"requests"`
	RequestDuration   int64   `json:"request duration"`
	RequestMethod     string  `json:"request method"`
	RequestURI        string  `json:"request uri"`
	ContentLength     int64   `json:"content length"`
	User              string  `json:"user"`
	Script            string  `json:"script"`
	LastRequestCPU    float64 `json:"last request cpu"`
	LastRequestMemory float64 `json:"last request memory"`
	CurrentRSS        int64   `json:"current_rss"`
}

type Pool struct {
	Address             string            `json:"address"`
	Path                string            `json:"path"`
	Name                string            `json:"pool"`
	ProcessManager      string            `json:"process manager"`
	StartTime           int64             `json:"start time"`
	StartSince          int64             `json:"start since"`
	AcceptedConnections int64             `json:"accepted conn"`
	ListenQueue         int64             `json:"listen queue"`
	MaxListenQueue      int64             `json:"max listen queue"`
	ListenQueueLength   int64             `json:"listen queue len"`
	IdleProcesses       int64             `json:"idle processes"`
	ActiveProcesses     int64             `json:"active processes"`
	TotalProcesses      int64             `json:"total processes"`
	MaxActiveProcesses  int64             `json:"max active processes"`
	MaxChildrenReached  int64             `json:"max children reached"`
	SlowRequests        int64             `json:"slow requests"`
	MemoryPeak          int64             `json:"memory peak"`
	Processes           []PoolProcess     `json:"processes"`
	ProcessesCpu        *float64          `json:"processes_cpu"`
	ProcessesMemory     *float64          `json:"processes_memory"`
	Config              map[string]string `json:"config,omitempty"`
	OpcacheStatus       OpcacheStatus     `json:"opcache_status,omitempty"`
	PhpInfo             Info              `json:"php_info,omitempty"`
}

type Result struct {
	Timestamp time.Time
	Pools     map[string]Pool
	Global    map[string]string `json:"global_config,omitempty"`
}

// PoolOutcome is what happened to one configured pool during a scrape: either a
// Result, or the error that prevented one. Failures are values rather than log
// lines so the collector can emit up=0 for a pool that did not answer -- a pool
// that silently vanishes from the output is indistinguishable from a pool that
// was removed from the configuration.
type PoolOutcome struct {
	// Name is the pool's configured or discovered name, used to label a
	// failure. A successful scrape prefers the name PHP-FPM itself reports.
	Name   string
	Socket string
	Result *Result
	Err    error
}

// GetMetrics scrapes every configured pool. It returns one outcome per pool, in
// configuration order, and an error only when nothing at all could be collected.
func GetMetrics(ctx context.Context, cfg *config.Config) ([]PoolOutcome, error) {
	outcomes := make([]PoolOutcome, 0, len(cfg.PHPFpm.Pools))

	for _, poolCfg := range cfg.PHPFpm.Pools {
		outcome := collectPool(ctx, poolCfg)
		if outcome.Err != nil {
			logging.L().Warn("Cbox Pool scrape failed",
				"pool", outcome.Name, "socket", outcome.Socket, "err", outcome.Err)
		}
		outcomes = append(outcomes, outcome)
	}

	if len(outcomes) > 0 && !slices.ContainsFunc(outcomes, func(o PoolOutcome) bool { return o.Err == nil }) {
		return outcomes, fmt.Errorf("all %d configured PHP-FPM pools failed to scrape", len(outcomes))
	}

	return outcomes, nil
}

// collectPool scrapes a single pool. Extracted from GetMetrics so its client and
// response body close at the end of the pool rather than at the end of the
// whole scrape.
func collectPool(ctx context.Context, poolCfg config.FPMPoolConfig) PoolOutcome {
	outcome := PoolOutcome{Name: poolName(poolCfg), Socket: poolCfg.StatusSocket}

	result := &Result{
		Timestamp: time.Now(),
		Pools:     make(map[string]Pool),
		Global:    make(map[string]string),
	}

	scheme, address, path, err := ParseAddress(poolCfg.StatusSocket, poolCfg.StatusPath)
	if err != nil {
		outcome.Err = fmt.Errorf("invalid FPM socket address: %w", err)
		return outcome
	}

	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout(poolCfg))
	logging.L().Debug("Cbox Dialing FastCGI", "scheme", scheme, "address", address, "status_path", path)
	client, err := fcgx.DialContext(dialCtx, scheme, address)
	cancel()
	if err != nil {
		outcome.Err = fmt.Errorf("failed to dial FastCGI: %w", err)
		return outcome
	}
	defer func() { _ = client.Close() }()

	env := map[string]string{
		"SCRIPT_FILENAME": path,
		"SCRIPT_NAME":     path,
		"SERVER_SOFTWARE": "fpm-exporter",
		"REMOTE_ADDR":     "127.0.0.1",
		"QUERY_STRING":    "json&full",
	}
	logging.L().Debug("Cbox Sending FCGI request", "env", env)

	resp, err := client.Get(ctx, env)
	if err != nil {
		outcome.Err = fmt.Errorf("fcgi GET failed: %w", err)
		return outcome
	}
	defer func() { _ = resp.Body.Close() }()

	var pool Pool
	if err := fcgx.ReadJSON(resp, &pool); err != nil {
		outcome.Err = fmt.Errorf("failed to parse FPM status JSON: %w", err)
		return outcome
	}

	pool.Address = address
	pool.Path = path
	if conf, err := ParseFPMConfig(poolCfg.Binary, poolCfg.ConfigPath); err == nil {
		for section, values := range conf.Pools {
			if strings.EqualFold(section, pool.Name) {
				pool.Config = exportableConfig(values)
			}
		}
		result.Global = exportableConfig(conf.Global)
	}

	recountProcesses(&pool, poolCfg.StatusPath)

	phpStatus, err := GetPHPStats(ctx, poolCfg)
	if err == nil && phpStatus != nil {
		pool.PhpInfo = *phpStatus
	} else {
		logging.L().Debug("Cbox failed to get PHP info", "error", err)
	}

	opcacheStatus, err := GetOpcacheStatus(ctx, poolCfg)
	if err == nil && opcacheStatus != nil {
		pool.OpcacheStatus = *opcacheStatus
	} else {
		logging.L().Debug("Cbox failed to get Opcache info", "error", err)
	}

	result.Pools[pool.Name] = pool
	result.Pools[pool.Name] = pool
	// Only adopt the name PHP-FPM reports when the operator did not choose one;
	// a configured name has to survive the pool going down and coming back.
	if poolCfg.Name == "" && pool.Name != "" {
		outcome.Name = pool.Name
	}
	outcome.Result = result

	return outcome

}

// recountProcesses derives the pool's process counts and per-request averages
// from the process list, rather than trusting the summary fields, and strips
// the query string from each request URI.
//
// Extracted so it can be tested: the previous test copied this logic into
// itself and asserted on the copy, so no production statement ran.
func recountProcesses(pool *Pool, statusPath string) {
	var totalCPU, totalMem float64
	var count int

	// PHP-FPM reports the full request URI of each worker, query string
	// included -- on a real site that is a rolling sample of production URLs
	// with their tokens in them, and /json republishes it unauthenticated. The
	// path is enough for the filter below and for debugging.
	for i := range pool.Processes {
		if q := strings.IndexByte(pool.Processes[i].RequestURI, '?'); q >= 0 {
			pool.Processes[i].RequestURI = pool.Processes[i].RequestURI[:q]
		}
	}
	var activeCount, idleCount int64

	for _, proc := range pool.Processes {
		// Count processes by state
		switch strings.ToLower(proc.State) {
		case "running", "reading headers", "info", "finishing", "ending":
			activeCount++
		case "idle":
			idleCount++
		}

		// CPU/memory calculation, excluding the exporter's own traffic: the
		// status call and the opcache probe. Counting those skews the averages
		// worst on idle pools, where they are most of the traffic.
		if !strings.HasPrefix(proc.RequestURI, statusPath) &&
			!strings.HasPrefix(proc.RequestURI, "/"+opcacheScriptPrefix) {

			totalCPU += float64(proc.LastRequestCPU)
			totalMem += float64(proc.LastRequestMemory)
			count++
		}
	}

	// Recalculate process counts from actual process list
	pool.ActiveProcesses = activeCount
	pool.IdleProcesses = idleCount
	pool.TotalProcesses = int64(len(pool.Processes))

	if count > 0 {
		pool.ProcessesCpu = ptr(totalCPU / float64(count))
		pool.ProcessesMemory = ptr(totalMem / float64(count))
	}

}

// exportedConfigKeys are the only pool-config settings that leave this process.
// `php-fpm -tt` dumps the effective configuration, which routinely carries
// secrets -- env[DB_PASSWORD], env[APP_KEY], php_admin_value[...] -- and the
// whole map used to be serialised on the unauthenticated /json endpoint. These
// eleven are exactly the keys the collector turns into metrics.
var exportedConfigKeys = map[string]bool{
	"pm.max_children":           true,
	"pm.max_requests":           true,
	"pm.max_spare_servers":      true,
	"pm.max_spawn_rate":         true,
	"pm.min_spare_servers":      true,
	"pm.process_idle_timeout":   true,
	"pm.start_servers":          true,
	"request_slowlog_timeout":   true,
	"request_terminate_timeout": true,
	"rlimit_core":               true,
	"rlimit_files":              true,
}

// exportableConfig copies through only the settings we publish. An allow-list,
// not a deny-list: a new PHP-FPM directive holding a credential must not become
// a leak because nobody remembered to add it to a block list.
func exportableConfig(values map[string]string) map[string]string {
	exportable := make(map[string]string, len(exportedConfigKeys))
	for key, value := range values {
		if exportedConfigKeys[strings.ToLower(strings.TrimSpace(key))] {
			exportable[key] = value
		}
	}
	return exportable
}

// poolName prefers the configured name and falls back to the socket, so a pool
// that fails before PHP-FPM ever answers still carries a usable label.
func poolName(poolCfg config.FPMPoolConfig) string {
	if poolCfg.Name != "" {
		return poolCfg.Name
	}
	if poolCfg.StatusSocket != "" {
		return poolCfg.StatusSocket
	}
	return poolCfg.Socket
}

func dialTimeout(poolCfg config.FPMPoolConfig) time.Duration {
	if poolCfg.Timeout > 0 {
		return poolCfg.Timeout
	}
	return 3 * time.Second
}

func GetMetricsForPool(ctx context.Context, pool config.FPMPoolConfig) (*Result, error) {
	scheme, address, path, err := ParseAddress(pool.StatusSocket, pool.StatusPath)
	if err != nil {
		return nil, fmt.Errorf("invalid FPM socket address: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FastCGI: %w", err)
	}
	defer func() { _ = client.Close() }()

	env := map[string]string{
		"SCRIPT_FILENAME": path,
		"SCRIPT_NAME":     path,
		"SERVER_SOFTWARE": "fpm-exporter",
		"REMOTE_ADDR":     "127.0.0.1",
		"QUERY_STRING":    "json&full",
	}

	resp, err := client.Get(ctx, env)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var poolData Pool
	err = fcgx.ReadJSON(resp, &poolData)

	if err != nil {
		return nil, fmt.Errorf("failed to parse FPM JSON: %w", err)
	}

	// Recalculate process counts from actual process list
	var activeCount, idleCount int64
	for _, proc := range poolData.Processes {
		switch strings.ToLower(proc.State) {
		case "running", "reading headers", "info", "finishing", "ending":
			activeCount++
		case "idle":
			idleCount++
		}
	}

	poolData.ActiveProcesses = activeCount
	poolData.IdleProcesses = idleCount
	poolData.TotalProcesses = int64(len(poolData.Processes))

	return &Result{
		Timestamp: time.Now(),
		Pools:     map[string]Pool{poolData.Name: poolData},
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}

func ParseAddress(addr string, path string) (scheme, address, scriptPath string, err error) {
	if strings.HasPrefix(addr, "unix://") {
		return "unix", strings.TrimPrefix(addr, "unix://"), path, nil
	}
	if strings.HasPrefix(addr, "tcp://") {
		return "tcp", strings.TrimPrefix(addr, "tcp://"), path, nil
	}
	if strings.HasPrefix(addr, "/") {
		return "unix", addr, path, nil
	}
	return "", "", "", fmt.Errorf("unsupported socket format: %s", addr)
}
