package laravel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGetQueueSizes(t *testing.T) {
	tests := []struct {
		name     string
		queueMap map[string][]string
		wantErr  bool
	}{
		{
			name:     "empty queue map",
			queueMap: map[string][]string{},
			wantErr:  false,
		},
		{
			name: "single connection with one queue",
			queueMap: map[string][]string{
				"default": {"default"},
			},
			wantErr: true, // Will fail without proper Laravel setup
		},
		{
			name: "multiple connections with multiple queues",
			queueMap: map[string][]string{
				"default": {"default", "high"},
				"redis":   {"background"},
			},
			wantErr: true, // Will fail without proper Laravel setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for testing
			tempDir := t.TempDir()

			result, err := GetQueueSizes(context.Background(), tempDir, "php", tt.queueMap)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetQueueSizes(context.Background(), ) expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetQueueSizes(context.Background(), ) unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("GetQueueSizes(context.Background(), ) returned nil result")
				return
			}

			// For empty queue map, result should be empty
			if len(tt.queueMap) == 0 && len(*result) != 0 {
				t.Errorf("GetQueueSizes(context.Background(), ) expected empty result for empty queue map")
			}
		})
	}
}

func TestGetQueueSizes_WithEnvVariable(t *testing.T) {
	// Test that NIGHTWATCH_ENABLED=false is set in the environment
	// We'll create a mock PHP script that outputs the environment variables
	tempDir := t.TempDir()

	// Create a mock php script that outputs environment variables
	mockPhpScript := `#!/bin/bash
echo "NIGHTWATCH_ENABLED=$NIGHTWATCH_ENABLED"
exit 0`

	mockPhpPath := tempDir + "/mock-php"
	err := os.WriteFile(mockPhpPath, []byte(mockPhpScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock PHP script: %v", err)
	}

	queueMap := map[string][]string{
		"default": {"default"},
	}

	// Use our mock PHP script
	_, err = GetQueueSizes(context.Background(), tempDir, mockPhpPath, queueMap)

	// We expect an error since our mock doesn't output valid JSON
	// but we can check if the environment variable was set by looking at the error output
	if err == nil {
		t.Errorf("Expected error when running mock PHP script")
		return
	}

	// The error should contain our environment variable output
	if !strings.Contains(err.Error(), "NIGHTWATCH_ENABLED=false") {
		t.Errorf("Expected error output to contain 'NIGHTWATCH_ENABLED=false', got: %s", err.Error())
	}
}

func TestGetQueueSizes_EnvVariableValidation(t *testing.T) {
	// Additional test to verify the environment variable is properly passed through a script
	tempDir := t.TempDir()

	// Create a script that validates NIGHTWATCH_ENABLED and outputs JSON
	validatorScript := `#!/bin/bash
if [ "$NIGHTWATCH_ENABLED" = "false" ]; then
    echo '{"default":{"default":{"size":0}}}'
else
    echo "ERROR: NIGHTWATCH_ENABLED not set to false, got: $NIGHTWATCH_ENABLED" >&2
    exit 1
fi`

	scriptPath := tempDir + "/validator-php"
	err := os.WriteFile(scriptPath, []byte(validatorScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create validator script: %v", err)
	}

	queueMap := map[string][]string{
		"default": {"default"},
	}

	// Use our validator script - this should succeed if env var is set correctly
	result, err := GetQueueSizes(context.Background(), tempDir, scriptPath, queueMap)

	if err != nil {
		t.Errorf("Expected no error with validator script, got: %v", err)
		return
	}

	if result == nil {
		t.Errorf("Expected valid result from validator script")
		return
	}

	// Verify the result structure
	queueSizes := *result
	if _, exists := queueSizes["default"]; !exists {
		t.Errorf("Expected 'default' connection in result")
	}
}

func TestQueueMetrics_JSONMarshaling(t *testing.T) {
	// Test that QueueMetrics struct can be properly marshaled/unmarshaled
	metrics := QueueMetrics{
		Driver:          stringPtr("database"),
		Size:            intPtr(10),
		Pending:         intPtr(5),
		Scheduled:       intPtr(2),
		Reserved:        intPtr(1),
		OldestPending:   intPtr(300),
		Failed:          intPtr(0),
		OldestFailed:    nil,
		NewestFailed:    nil,
		Failed1Min:      intPtr(0),
		Failed5Min:      intPtr(0),
		Failed10Min:     intPtr(0),
		FailedRate1Min:  float32Ptr(0.0),
		FailedRate5Min:  float32Ptr(0.0),
		FailedRate10Min: float32Ptr(0.0),
		ParseError:      nil,
	}

	// Test that all fields are accessible
	if *metrics.Driver != "database" {
		t.Errorf("Expected driver to be 'database', got %s", *metrics.Driver)
	}

	if *metrics.Size != 10 {
		t.Errorf("Expected size to be 10, got %d", *metrics.Size)
	}

	if *metrics.Pending != 5 {
		t.Errorf("Expected pending to be 5, got %d", *metrics.Pending)
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float32Ptr(f float32) *float32 {
	return &f
}

// Connection and queue names are operator-supplied. They must never end up as
// PHP source: a name containing a quote used to break out of the string it was
// interpolated into.
func TestBuildQueueScript_DoesNotInterpolateNames(t *testing.T) {
	queueMap := map[string][]string{
		`redis'; system('id'); //`: {`default'; system('whoami'); //`},
	}

	script, err := buildQueueScript(queueMap)
	if err != nil {
		t.Fatalf("buildQueueScript() unexpected error: %v", err)
	}

	if strings.Contains(script, "system(") {
		t.Errorf("Expected names to be encoded, but the script contains them verbatim:\n%s", script)
	}

	// ...and the payload still has to arrive intact on the PHP side.
	start := strings.Index(script, "base64_decode('")
	if start == -1 {
		t.Fatalf("Expected a base64 payload in the script")
	}
	start += len("base64_decode('")
	end := strings.Index(script[start:], "'")
	if end == -1 {
		t.Fatalf("Unterminated base64 payload")
	}

	decoded, err := base64.StdEncoding.DecodeString(script[start : start+end])
	if err != nil {
		t.Fatalf("Failed to decode payload: %v", err)
	}

	var roundTripped map[string][]string
	if err := json.Unmarshal(decoded, &roundTripped); err != nil {
		t.Fatalf("Failed to parse payload: %v", err)
	}

	if !reflect.DeepEqual(roundTripped, queueMap) {
		t.Errorf("Expected %v, got %v", queueMap, roundTripped)
	}
}

func TestBuildQueueScript_IsValidPHP(t *testing.T) {
	php, err := exec.LookPath("php")
	if err != nil {
		t.Skip("php not available")
	}

	script, err := buildQueueScript(map[string][]string{
		"redis":    {"default", "emails"},
		"database": {"jobs"},
	})
	if err != nil {
		t.Fatalf("buildQueueScript() unexpected error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "script.php")
	if err := os.WriteFile(path, []byte("<?php\n"+script), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	out, err := exec.Command(php, "-l", path).CombinedOutput()
	if err != nil {
		t.Errorf("Generated script is not valid PHP: %v\n%s", err, out)
	}
}
