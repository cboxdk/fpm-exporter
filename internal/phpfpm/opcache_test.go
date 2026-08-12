package phpfpm

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/cboxdk/fpm-exporter/internal/logging"
)

func TestOpcacheStatus_Structure(t *testing.T) {
	// Test OpcacheStatus structure
	status := OpcacheStatus{
		Enabled: true,
		MemoryUsage: Memory{
			UsedMemory:       1024000,
			FreeMemory:       512000,
			WastedMemory:     1000,
			CurrentWastedPct: 0.1,
		},
		Statistics: Stats{
			NumCachedScripts: 150,
			Hits:             50000,
			Misses:           1000,
			BlacklistMisses:  5,
			OomRestarts:      0,
			HashRestarts:     0,
			ManualRestarts:   1,
			HitRate:          98.5,
		},
	}

	// Verify structure
	if !status.Enabled {
		t.Errorf("Expected Enabled to be true")
	}

	if status.MemoryUsage.UsedMemory != 1024000 {
		t.Errorf("Expected UsedMemory to be 1024000, got %d", status.MemoryUsage.UsedMemory)
	}

	if status.MemoryUsage.FreeMemory != 512000 {
		t.Errorf("Expected FreeMemory to be 512000, got %d", status.MemoryUsage.FreeMemory)
	}

	if status.MemoryUsage.WastedMemory != 1000 {
		t.Errorf("Expected WastedMemory to be 1000, got %d", status.MemoryUsage.WastedMemory)
	}

	if status.MemoryUsage.CurrentWastedPct != 0.1 {
		t.Errorf("Expected CurrentWastedPct to be 0.1, got %f", status.MemoryUsage.CurrentWastedPct)
	}

	if status.Statistics.NumCachedScripts != 150 {
		t.Errorf("Expected NumCachedScripts to be 150, got %d", status.Statistics.NumCachedScripts)
	}

	if status.Statistics.Hits != 50000 {
		t.Errorf("Expected Hits to be 50000, got %d", status.Statistics.Hits)
	}

	if status.Statistics.Misses != 1000 {
		t.Errorf("Expected Misses to be 1000, got %d", status.Statistics.Misses)
	}

	if status.Statistics.HitRate != 98.5 {
		t.Errorf("Expected HitRate to be 98.5, got %f", status.Statistics.HitRate)
	}
}

func TestMemory_Structure(t *testing.T) {
	// Test Memory structure with various values
	memory := Memory{
		UsedMemory:       2048000,
		FreeMemory:       1024000,
		WastedMemory:     5000,
		CurrentWastedPct: 0.25,
	}

	// Verify all fields are uint64 or float64 as expected
	if memory.UsedMemory != 2048000 {
		t.Errorf("Expected UsedMemory to be uint64 with value 2048000")
	}

	if memory.FreeMemory != 1024000 {
		t.Errorf("Expected FreeMemory to be uint64 with value 1024000")
	}

	if memory.WastedMemory != 5000 {
		t.Errorf("Expected WastedMemory to be uint64 with value 5000")
	}

	if memory.CurrentWastedPct != 0.25 {
		t.Errorf("Expected CurrentWastedPct to be float64 with value 0.25")
	}
}

func TestStats_Structure(t *testing.T) {
	// Test Stats structure with various values
	stats := Stats{
		NumCachedScripts: 500,
		Hits:             1000000,
		Misses:           10000,
		BlacklistMisses:  50,
		OomRestarts:      2,
		HashRestarts:     1,
		ManualRestarts:   3,
		HitRate:          99.0,
	}

	// Verify all fields are uint64 or float64 as expected
	if stats.NumCachedScripts != 500 {
		t.Errorf("Expected NumCachedScripts to be uint64 with value 500")
	}

	if stats.Hits != 1000000 {
		t.Errorf("Expected Hits to be uint64 with value 1000000")
	}

	if stats.Misses != 10000 {
		t.Errorf("Expected Misses to be uint64 with value 10000")
	}

	if stats.BlacklistMisses != 50 {
		t.Errorf("Expected BlacklistMisses to be uint64 with value 50")
	}

	if stats.OomRestarts != 2 {
		t.Errorf("Expected OomRestarts to be uint64 with value 2")
	}

	if stats.HashRestarts != 1 {
		t.Errorf("Expected HashRestarts to be uint64 with value 1")
	}

	if stats.ManualRestarts != 3 {
		t.Errorf("Expected ManualRestarts to be uint64 with value 3")
	}

	if stats.HitRate != 99.0 {
		t.Errorf("Expected HitRate to be float64 with value 99.0")
	}
}

func TestOpcacheStatus_JSONMarshaling(t *testing.T) {
	// Test JSON marshaling and unmarshaling
	originalStatus := OpcacheStatus{
		Enabled: true,
		MemoryUsage: Memory{
			UsedMemory:       1024000,
			FreeMemory:       512000,
			WastedMemory:     1000,
			CurrentWastedPct: 0.1,
		},
		Statistics: Stats{
			NumCachedScripts: 150,
			Hits:             50000,
			Misses:           1000,
			BlacklistMisses:  5,
			OomRestarts:      0,
			HashRestarts:     0,
			ManualRestarts:   1,
			HitRate:          98.5,
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalStatus)
	if err != nil {
		t.Fatalf("Failed to marshal OpcacheStatus to JSON: %v", err)
	}

	// Unmarshal back
	var unmarshaledStatus OpcacheStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to OpcacheStatus: %v", err)
	}

	// Compare values
	if unmarshaledStatus.Enabled != originalStatus.Enabled {
		t.Errorf("Enabled mismatch after JSON round-trip")
	}

	if unmarshaledStatus.MemoryUsage.UsedMemory != originalStatus.MemoryUsage.UsedMemory {
		t.Errorf("UsedMemory mismatch after JSON round-trip")
	}

	if unmarshaledStatus.MemoryUsage.CurrentWastedPct != originalStatus.MemoryUsage.CurrentWastedPct {
		t.Errorf("CurrentWastedPct mismatch after JSON round-trip")
	}

	if unmarshaledStatus.Statistics.Hits != originalStatus.Statistics.Hits {
		t.Errorf("Hits mismatch after JSON round-trip")
	}

	if unmarshaledStatus.Statistics.HitRate != originalStatus.Statistics.HitRate {
		t.Errorf("HitRate mismatch after JSON round-trip")
	}
}

func TestOpcacheStatus_JSONTags(t *testing.T) {
	// Test that JSON tags are correctly defined
	status := OpcacheStatus{
		Enabled: true,
		MemoryUsage: Memory{
			UsedMemory:       1000000,
			FreeMemory:       2000000,
			WastedMemory:     5000,
			CurrentWastedPct: 0.5,
		},
		Statistics: Stats{
			NumCachedScripts: 100,
			Hits:             10000,
			Misses:           500,
			BlacklistMisses:  10,
			OomRestarts:      1,
			HashRestarts:     0,
			ManualRestarts:   2,
			HitRate:          95.0,
		},
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	jsonStr := string(jsonData)

	// Check that JSON contains expected field names (as per struct tags)
	expectedFields := []string{
		"opcache_enabled",
		"memory_usage",
		"opcache_statistics",
		"used_memory",
		"free_memory",
		"wasted_memory",
		"current_wasted_percentage",
		"num_cached_scripts",
		"hits",
		"misses",
		"blacklist_misses",
		"oom_restarts",
		"hash_restarts",
		"manual_restarts",
		"opcache_hit_rate",
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("Expected JSON to contain field '%s', but it was not found. JSON: %s", field, jsonStr)
		}
	}
}

func TestGetOpcacheStatus_ErrorHandling(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	ctx := context.Background()

	// Test with invalid socket
	cfg := config.FPMPoolConfig{
		StatusSocket: "invalid://socket/path",
	}

	_, err := GetOpcacheStatus(ctx, cfg)
	if err == nil {
		t.Errorf("Expected error for invalid socket")
	}

	if !strings.Contains(err.Error(), "invalid socket") {
		t.Errorf("Expected error to mention 'invalid socket', got: %s", err.Error())
	}

	// Test with non-existent socket
	cfg2 := config.FPMPoolConfig{
		StatusSocket: "unix:///non/existent/socket",
	}

	_, err = GetOpcacheStatus(ctx, cfg2)
	if err == nil {
		t.Errorf("Expected error for non-existent socket")
	}

	// Should be a dial error
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "dial") && !strings.Contains(errStr, "connect") && !strings.Contains(errStr, "fmp") {
		t.Errorf("Expected dial/connection error, got: %s", err.Error())
	}
}

func TestGetOpcacheStatus_ScriptCreation(t *testing.T) {
	// Initialize logging to prevent panic
	logging.Init(config.LoggingBlock{Level: "error", Format: "text"})

	ctx := context.Background()
	cfg := config.FPMPoolConfig{
		StatusSocket: "unix:///non/existent/socket",
	}

	// This will fail to connect, but should still have written the script
	_, err := GetOpcacheStatus(ctx, cfg)
	if err == nil {
		t.Errorf("Expected error due to non-existent socket")
	}

	path, err := opcacheScript()
	if err != nil {
		t.Fatalf("opcacheScript() unexpected error: %v", err)
	}

	if path == "" {
		t.Fatalf("Expected a script path to be set")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Expected opcache status script at %s: %v", path, err)
	}

	// PHP-FPM commonly runs as another user, so the script has to be readable —
	// but never writable by anyone but us.
	if perm := info.Mode().Perm(); perm != 0644 {
		t.Errorf("Expected script mode 0644, got %04o", perm)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read opcache script: %v", err)
	}

	scriptContent := string(content)
	expectedContent := []string{
		"<?php",
		"error_reporting(0)",
		"ini_set('display_errors', 0)",
		"header(\"Status: 200 OK\")",
		"header(\"Content-Type: application/json\")",
		"echo json_encode(opcache_get_status())",
		"exit;",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(scriptContent, expected) {
			t.Errorf("Expected script to contain '%s', but it was not found", expected)
		}
	}
}

// A local user must not be able to pre-create the path we hand to PHP-FPM and
// have their own code executed as the pool user.
func TestGetOpcacheStatus_DoesNotUsePlantedScript(t *testing.T) {
	plantedPath := "/tmp/cbox-opcache-status.php"
	plantedContent := `<?php echo "planted";`

	if err := os.WriteFile(plantedPath, []byte(plantedContent), 0644); err != nil {
		t.Fatalf("Failed to create planted script: %v", err)
	}
	defer func() { _ = os.Remove(plantedPath) }()

	path, err := opcacheScript()
	if err != nil {
		t.Fatalf("opcacheScript() unexpected error: %v", err)
	}

	if path == plantedPath {
		t.Fatalf("Expected a generated path, got the planted one: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}

	if strings.Contains(string(content), "planted") {
		t.Errorf("Expected the generated script, got planted content")
	}
}

func TestRemoveOpcacheScript(t *testing.T) {
	path, err := opcacheScript()
	if err != nil {
		t.Fatalf("opcacheScript() unexpected error: %v", err)
	}

	RemoveOpcacheScript()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected script at %s to be removed", path)
	}

	// The path is memoized for the process, so put it back for any test that
	// runs after this one.
	if err := os.WriteFile(path, []byte(opcacheScriptContent), 0644); err != nil {
		t.Fatalf("Failed to restore script: %v", err)
	}
}

func TestOpcacheStatus_EmptyValues(t *testing.T) {
	// Test with zero/empty values
	status := OpcacheStatus{
		Enabled: false,
		MemoryUsage: Memory{
			UsedMemory:       0,
			FreeMemory:       0,
			WastedMemory:     0,
			CurrentWastedPct: 0.0,
		},
		Statistics: Stats{
			NumCachedScripts: 0,
			Hits:             0,
			Misses:           0,
			BlacklistMisses:  0,
			OomRestarts:      0,
			HashRestarts:     0,
			ManualRestarts:   0,
			HitRate:          0.0,
		},
	}

	// Should be able to marshal/unmarshal even with zero values
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal empty status: %v", err)
	}

	var unmarshaledStatus OpcacheStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty status: %v", err)
	}

	// All values should be preserved
	if unmarshaledStatus.Enabled != false {
		t.Errorf("Expected Enabled to be false")
	}

	if unmarshaledStatus.MemoryUsage.UsedMemory != 0 {
		t.Errorf("Expected UsedMemory to be 0")
	}

	if unmarshaledStatus.Statistics.HitRate != 0.0 {
		t.Errorf("Expected HitRate to be 0.0")
	}
}

func TestOpcacheStatus_LargeValues(t *testing.T) {
	// Test with large values to ensure uint64 and float64 work correctly
	status := OpcacheStatus{
		Enabled: true,
		MemoryUsage: Memory{
			UsedMemory:       18446744073709551615, // Max uint64
			FreeMemory:       1844674407370955161,  // Large uint64
			WastedMemory:     184467440737095516,   // Large uint64
			CurrentWastedPct: 99.999999999999,      // Large float64
		},
		Statistics: Stats{
			NumCachedScripts: 18446744073709551615, // Max uint64
			Hits:             18446744073709551615, // Max uint64
			Misses:           1844674407370955161,  // Large uint64
			BlacklistMisses:  184467440737095516,   // Large uint64
			OomRestarts:      18446744073709551615, // Max uint64
			HashRestarts:     1844674407370955161,  // Large uint64
			ManualRestarts:   184467440737095516,   // Large uint64
			HitRate:          99.999999999999,      // Large float64
		},
	}

	// Should handle large values correctly
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status with large values: %v", err)
	}

	var unmarshaledStatus OpcacheStatus
	err = json.Unmarshal(jsonData, &unmarshaledStatus)
	if err != nil {
		t.Fatalf("Failed to unmarshal status with large values: %v", err)
	}

	// Verify large values are preserved
	if unmarshaledStatus.MemoryUsage.UsedMemory != 18446744073709551615 {
		t.Errorf("Large UsedMemory value not preserved")
	}

	if unmarshaledStatus.Statistics.Hits != 18446744073709551615 {
		t.Errorf("Large Hits value not preserved")
	}

	// Float64 precision might have minor differences, so check within a reasonable range
	expectedHitRate := 99.999999999999
	actualHitRate := unmarshaledStatus.Statistics.HitRate
	if actualHitRate < expectedHitRate-0.000001 || actualHitRate > expectedHitRate+0.000001 {
		t.Errorf("HitRate precision lost: expected ~%f, got %f", expectedHitRate, actualHitRate)
	}
}
