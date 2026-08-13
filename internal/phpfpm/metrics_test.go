package phpfpm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/cboxdk/fpm-exporter/internal/logging"
)

func TestPoolProcess_Structure(t *testing.T) {
	// Test PoolProcess structure
	process := PoolProcess{
		PID:               1234,
		State:             "Idle",
		StartTime:         1640995200, // Unix timestamp
		StartSince:        3600,       // 1 hour
		Requests:          150,
		RequestDuration:   5000, // 5 seconds in microseconds
		RequestMethod:     "GET",
		RequestURI:        "/api/users",
		ContentLength:     1024,
		User:              "www-data",
		Script:            "/var/www/app/index.php",
		LastRequestCPU:    0.05,
		LastRequestMemory: 1048576, // 1MB
		CurrentRSS:        2097152, // 2MB
	}

	// Verify all fields are correctly typed and accessible
	if process.PID != 1234 {
		t.Errorf("Expected PID to be int with value 1234")
	}

	if process.State != "Idle" {
		t.Errorf("Expected State to be string with value 'Idle'")
	}

	if process.StartTime != 1640995200 {
		t.Errorf("Expected StartTime to be int64 with value 1640995200")
	}

	if process.StartSince != 3600 {
		t.Errorf("Expected StartSince to be int64 with value 3600")
	}

	if process.Requests != 150 {
		t.Errorf("Expected Requests to be int64 with value 150")
	}

	if process.LastRequestCPU != 0.05 {
		t.Errorf("Expected LastRequestCPU to be float64 with value 0.05")
	}

	if process.LastRequestMemory != 1048576 {
		t.Errorf("Expected LastRequestMemory to be float64 with value 1048576")
	}

	if process.CurrentRSS != 2097152 {
		t.Errorf("Expected CurrentRSS to be int64 with value 2097152")
	}
}

func TestPool_Structure(t *testing.T) {
	// Test Pool structure with all fields
	pool := Pool{
		Address:             "unix:///var/run/php-fpm.sock",
		Path:                "/status",
		Name:                "www",
		ProcessManager:      "dynamic",
		StartTime:           1640995200,
		StartSince:          7200,
		AcceptedConnections: 5000,
		ListenQueue:         0,
		MaxListenQueue:      128,
		ListenQueueLength:   0,
		IdleProcesses:       5,
		ActiveProcesses:     3,
		TotalProcesses:      8,
		MaxActiveProcesses:  10,
		MaxChildrenReached:  2,
		SlowRequests:        1,
		MemoryPeak:          10485760, // 10MB
		Processes:           []PoolProcess{},
		ProcessesCpu:        ptr(0.15),
		ProcessesMemory:     ptr(8388608.0), // 8MB
		Config: map[string]string{
			"pm":                   "dynamic",
			"pm.max_children":      "20",
			"pm.start_servers":     "5",
			"pm.min_spare_servers": "5",
			"pm.max_spare_servers": "15",
		},
		OpcacheStatus: OpcacheStatus{
			Enabled: true,
			MemoryUsage: Memory{
				UsedMemory: 1048576,
			},
		},
		PhpInfo: Info{
			Version:    "PHP 8.2.10",
			Extensions: []string{"Core", "json"},
		},
	}

	// Verify structure and types
	if pool.Address != "unix:///var/run/php-fpm.sock" {
		t.Errorf("Expected Address to be string")
	}

	if pool.Name != "www" {
		t.Errorf("Expected Name to be string")
	}

	if pool.IdleProcesses != 5 {
		t.Errorf("Expected IdleProcesses to be int64")
	}

	if pool.ActiveProcesses != 3 {
		t.Errorf("Expected ActiveProcesses to be int64")
	}

	if pool.ProcessesCpu == nil || *pool.ProcessesCpu != 0.15 {
		t.Errorf("Expected ProcessesCpu to be *float64")
	}

	if pool.ProcessesMemory == nil || *pool.ProcessesMemory != 8388608.0 {
		t.Errorf("Expected ProcessesMemory to be *float64")
	}

	if len(pool.Config) != 5 {
		t.Errorf("Expected Config to be map[string]string with 5 entries")
	}

	if pool.Config["pm"] != "dynamic" {
		t.Errorf("Expected Config values to be accessible")
	}

	if !pool.OpcacheStatus.Enabled {
		t.Errorf("Expected OpcacheStatus to be embedded struct")
	}

	if pool.PhpInfo.Version != "PHP 8.2.10" {
		t.Errorf("Expected PhpInfo to be embedded struct")
	}
}

func TestResult_Structure(t *testing.T) {
	// Test Result structure
	now := time.Now()
	result := Result{
		Timestamp: now,
		Pools: map[string]Pool{
			"www": {
				Name:            "www",
				IdleProcesses:   5,
				ActiveProcesses: 3,
			},
			"api": {
				Name:            "api",
				IdleProcesses:   2,
				ActiveProcesses: 1,
			},
		},
		Global: map[string]string{
			"pid":       "/var/run/php-fpm.pid",
			"error_log": "/var/log/php-fpm.log",
		},
	}

	// Verify structure
	if !result.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp to be time.Time")
	}

	if len(result.Pools) != 2 {
		t.Errorf("Expected Pools to be map[string]Pool with 2 entries")
	}

	wwwPool, exists := result.Pools["www"]
	if !exists {
		t.Fatalf("Expected 'www' pool to exist")
	}

	if wwwPool.Name != "www" {
		t.Errorf("Expected pool name to be accessible")
	}

	if len(result.Global) != 2 {
		t.Errorf("Expected Global to be map[string]string with 2 entries")
	}

	if result.Global["pid"] != "/var/run/php-fpm.pid" {
		t.Errorf("Expected Global values to be accessible")
	}
}

func TestGetMetrics_ErrorHandling(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	ctx := context.Background()

	// Test with empty config
	emptyConfig := &config.Config{
		PHPFpm: config.FPMConfig{
			Pools: []config.FPMPoolConfig{},
		},
	}

	results, err := GetMetrics(ctx, emptyConfig)
	if err != nil {
		t.Errorf("Expected no error with empty config, got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty results with empty config, got %d", len(results))
	}

	// Test with invalid socket
	invalidConfig := &config.Config{
		PHPFpm: config.FPMConfig{
			Pools: []config.FPMPoolConfig{
				{
					Socket:       "invalid-socket",
					StatusSocket: "invalid://socket/path",
					StatusPath:   "/status",
					Binary:       "/usr/sbin/php-fpm",
				},
			},
		},
	}

	// A pool that cannot be scraped is reported as a failed outcome, not
	// dropped: the collector needs it to emit up=0. When every configured pool
	// fails, that is a scrape failure.
	results, err = GetMetrics(ctx, invalidConfig)
	if err == nil {
		t.Errorf("Expected an error when every configured pool fails")
	}
	if len(results) != 1 {
		t.Fatalf("Expected one outcome per configured pool, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Errorf("Expected the invalid pool to carry its error")
	}

	// Test with non-existent socket
	nonExistentConfig := &config.Config{
		PHPFpm: config.FPMConfig{
			Pools: []config.FPMPoolConfig{
				{
					Socket:       "non-existent",
					StatusSocket: "unix:///non/existent/socket",
					StatusPath:   "/status",
					Binary:       "/usr/sbin/php-fpm",
				},
			},
		},
	}

	results, err = GetMetrics(ctx, nonExistentConfig)
	if err == nil {
		t.Errorf("Expected an error when the only configured pool is unreachable")
	}
	if len(results) != 1 || results[0].Err == nil {
		t.Errorf("Expected the unreachable pool to be reported with its error, got %+v", results)
	}
	if results[0].Result != nil {
		t.Errorf("Expected no Result for a failed pool")
	}
}

func TestGetMetricsForPool_ErrorHandling(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	ctx := context.Background()

	// Test with invalid socket format
	poolConfig := config.FPMPoolConfig{
		Socket:     "invalid-format",
		StatusPath: "/status",
	}

	_, err := GetMetricsForPool(ctx, poolConfig)
	if err == nil {
		t.Errorf("Expected error for invalid socket format")
	}

	if !strings.Contains(err.Error(), "invalid FPM socket address") {
		t.Errorf("Expected error to mention invalid socket address, got: %s", err.Error())
	}

	// Test with non-existent socket
	poolConfig2 := config.FPMPoolConfig{
		Socket:     "unix:///non/existent/socket",
		StatusPath: "/status",
	}

	_, err = GetMetricsForPool(ctx, poolConfig2)
	if err == nil {
		t.Errorf("Expected error for non-existent socket")
	}

	// Should be a FastCGI dial error
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "failed to dial fastcgi") && !strings.Contains(errStr, "invalid fpm socket address") {
		t.Errorf("Expected FastCGI dial error, got: %s", err.Error())
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name               string
		addr               string
		path               string
		expectedScheme     string
		expectedAddress    string
		expectedScriptPath string
		expectError        bool
	}{
		{
			name:               "unix socket with protocol",
			addr:               "unix:///var/run/php-fpm.sock",
			path:               "/status",
			expectedScheme:     "unix",
			expectedAddress:    "/var/run/php-fpm.sock",
			expectedScriptPath: "/status",
			expectError:        false,
		},
		{
			name:               "unix socket without protocol",
			addr:               "/var/run/php-fpm.sock",
			path:               "/status",
			expectedScheme:     "unix",
			expectedAddress:    "/var/run/php-fpm.sock",
			expectedScriptPath: "/status",
			expectError:        false,
		},
		{
			name:               "tcp socket with protocol",
			addr:               "tcp://127.0.0.1:9000",
			path:               "/status",
			expectedScheme:     "tcp",
			expectedAddress:    "127.0.0.1:9000",
			expectedScriptPath: "/status",
			expectError:        false,
		},
		{
			name:        "unsupported protocol",
			addr:        "http://example.com",
			path:        "/status",
			expectError: true,
		},
		{
			name:        "empty address",
			addr:        "",
			path:        "/status",
			expectError: true,
		},
		{
			name:        "invalid format",
			addr:        "invalid-format",
			path:        "/status",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, address, scriptPath, err := ParseAddress(tt.addr, tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					if scheme != tt.expectedScheme {
						t.Errorf("Expected scheme '%s', got '%s'", tt.expectedScheme, scheme)
					}
					if address != tt.expectedAddress {
						t.Errorf("Expected address '%s', got '%s'", tt.expectedAddress, address)
					}
					if scriptPath != tt.expectedScriptPath {
						t.Errorf("Expected scriptPath '%s', got '%s'", tt.expectedScriptPath, scriptPath)
					}
				}
			}
		})
	}
}

func TestPtr(t *testing.T) {
	// Test the ptr helper function
	intVal := 42
	intPtr := ptr(intVal)

	if intPtr == nil {
		t.Fatalf("Expected ptr to return non-nil pointer")
	}

	if *intPtr != intVal {
		t.Errorf("Expected ptr to return pointer to correct value")
	}

	// Test with different types
	stringVal := "test"
	stringPtr := ptr(stringVal)

	if stringPtr == nil {
		t.Fatalf("Expected ptr to work with string")
	}

	if *stringPtr != stringVal {
		t.Errorf("Expected ptr to return pointer to correct string value")
	}

	float64Val := 3.14
	float64Ptr := ptr(float64Val)

	if float64Ptr == nil {
		t.Fatalf("Expected ptr to work with float64")
	}

	if *float64Ptr != float64Val {
		t.Errorf("Expected ptr to return pointer to correct float64 value")
	}
}

func TestGetMetrics_PoolConfigParsing(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	// Test that config parsing logic works (even though actual FPM calls will fail)
	ctx := context.Background()

	cfg := &config.Config{
		PHPFpm: config.FPMConfig{
			Pools: []config.FPMPoolConfig{
				{
					Socket:       "pool1",
					StatusSocket: "unix:///var/run/pool1.sock",
					StatusPath:   "/status",
					Binary:       "/usr/sbin/php-fpm1",
					ConfigPath:   "/etc/php1/fpm.conf",
				},
				{
					Socket:       "pool2",
					StatusSocket: "tcp://127.0.0.1:9001",
					StatusPath:   "/fpm-status",
					Binary:       "/usr/sbin/php-fpm2",
					ConfigPath:   "/etc/php2/fpm.conf",
				},
			},
		},
	}

	// Both pools fail to connect, so this is a failed scrape — but every
	// configured pool must still come back as an outcome, in order, so the
	// collector can report each one down individually.
	results, err := GetMetrics(ctx, cfg)
	if err == nil {
		t.Errorf("Expected an error when both configured pools are unreachable")
	}

	if len(results) != 2 {
		t.Fatalf("Expected an outcome for each of the 2 configured pools, got %d", len(results))
	}
	for i, outcome := range results {
		if outcome.Err == nil {
			t.Errorf("Expected pool %d to carry an error", i)
		}
		if outcome.Socket == "" {
			t.Errorf("Expected pool %d to be labelled with its socket", i)
		}
	}
}

func TestPool_JSONTags(t *testing.T) {
	// Test that Pool struct has proper JSON tags by checking field access
	pool := Pool{
		Name:                "test-pool",
		ActiveProcesses:     5,
		IdleProcesses:       3,
		TotalProcesses:      8,
		MaxChildrenReached:  1,
		SlowRequests:        0,
		AcceptedConnections: 1000,
	}

	// These fields should be accessible and properly typed
	if pool.Name != "test-pool" {
		t.Errorf("Name field access failed")
	}

	if pool.ActiveProcesses != 5 {
		t.Errorf("ActiveProcesses field access failed")
	}

	if pool.IdleProcesses != 3 {
		t.Errorf("IdleProcesses field access failed")
	}

	// Test that we can create pools with various process states
	processes := []PoolProcess{
		{
			PID:   1001,
			State: "Running",
		},
		{
			PID:   1002,
			State: "Idle",
		},
	}

	pool.Processes = processes

	if len(pool.Processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(pool.Processes))
	}

	if pool.Processes[0].State != "Running" {
		t.Errorf("Expected first process to be Running")
	}

	if pool.Processes[1].State != "Idle" {
		t.Errorf("Expected second process to be Idle")
	}
}

func TestResult_TimestampHandling(t *testing.T) {
	// Test that Result properly handles timestamps
	now := time.Now()
	result := &Result{
		Timestamp: now,
		Pools:     make(map[string]Pool),
		Global:    make(map[string]string),
	}

	// Timestamp should be preserved
	if !result.Timestamp.Equal(now) {
		t.Errorf("Timestamp not preserved correctly")
	}

	// Test with zero time
	zeroResult := &Result{
		Timestamp: time.Time{},
		Pools:     make(map[string]Pool),
		Global:    make(map[string]string),
	}

	if !zeroResult.Timestamp.IsZero() {
		t.Errorf("Zero timestamp not handled correctly")
	}

	// Test timestamp comparison
	later := now.Add(time.Hour)
	laterResult := &Result{
		Timestamp: later,
		Pools:     make(map[string]Pool),
		Global:    make(map[string]string),
	}

	if !laterResult.Timestamp.After(result.Timestamp) {
		t.Errorf("Timestamp comparison failed")
	}
}

func TestRecountProcesses(t *testing.T) {
	pool := Pool{Processes: []PoolProcess{
		{PID: 1, State: "Running", RequestURI: "/checkout?token=secret", LastRequestCPU: 10, LastRequestMemory: 100},
		{PID: 2, State: "Idle", RequestURI: "/home", LastRequestCPU: 20, LastRequestMemory: 200},
		{PID: 3, State: "Reading headers", RequestURI: "/api/v1", LastRequestCPU: 30, LastRequestMemory: 300},
		// The exporter's own traffic must not count towards the averages.
		{PID: 4, State: "Running", RequestURI: "/status?json&full", LastRequestCPU: 999, LastRequestMemory: 999},
		{PID: 5, State: "Running", RequestURI: "/" + opcacheScriptPrefix + "123.php", LastRequestCPU: 999, LastRequestMemory: 999},
	}}

	recountProcesses(&pool, "/status")

	if pool.ActiveProcesses != 4 {
		t.Errorf("Expected 4 active (running, reading headers), got %d", pool.ActiveProcesses)
	}
	if pool.IdleProcesses != 1 {
		t.Errorf("Expected 1 idle, got %d", pool.IdleProcesses)
	}
	if pool.TotalProcesses != 5 {
		t.Errorf("Expected 5 total, got %d", pool.TotalProcesses)
	}

	// (10+20+30)/3 — the two self-inflicted requests are excluded.
	if pool.ProcessesCpu == nil || *pool.ProcessesCpu != 20 {
		t.Errorf("Expected the exporter's own requests to be excluded from the CPU average, got %v", pool.ProcessesCpu)
	}
	if pool.ProcessesMemory == nil || *pool.ProcessesMemory != 200 {
		t.Errorf("Expected the exporter's own requests to be excluded from the memory average, got %v", pool.ProcessesMemory)
	}

	if got := pool.Processes[0].RequestURI; got != "/checkout" {
		t.Errorf("Expected the query string to be stripped, got %q", got)
	}
}

// A pool where every process is the exporter's own must not divide by zero.
func TestRecountProcesses_AllExcluded(t *testing.T) {
	pool := Pool{Processes: []PoolProcess{
		{PID: 1, State: "Running", RequestURI: "/status", LastRequestCPU: 5},
	}}

	recountProcesses(&pool, "/status")

	if pool.ProcessesCpu != nil {
		t.Errorf("Expected no CPU average when every request was excluded, got %v", *pool.ProcessesCpu)
	}
}
