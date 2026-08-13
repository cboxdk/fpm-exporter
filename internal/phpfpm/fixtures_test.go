package phpfpm

import (
	"encoding/json"
	"os"
	"testing"
)

// These two tests are the ones that were missing: real output from a real
// PHP-FPM, parsed by the real parsers. The status payload's JSON tags are
// space-separated strings nobody would guess twice ("accepted conn", "listen
// queue len"), and the only test that named them assigned struct fields and
// read them back without ever unmarshalling. Renaming a tag to something
// tidier would have reported 0 forever with a green suite.

func TestParseFPMConfigOutput_RealFixture(t *testing.T) {
	output, err := os.ReadFile("testdata/php-fpm-tt.txt")
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	cfg, err := parseFPMConfigOutput(output)
	if err != nil {
		t.Fatalf("parseFPMConfigOutput() unexpected error: %v", err)
	}

	if len(cfg.Pools) != 2 {
		t.Fatalf("Expected the fixture's 2 pools, got %d: %v", len(cfg.Pools), keysOf(cfg.Pools))
	}

	www, ok := cfg.Pools["www"]
	if !ok {
		t.Fatalf("Expected a www pool, got %v", keysOf(cfg.Pools))
	}

	// Values the collector turns into metrics.
	for key, want := range map[string]string{
		"pm.max_children":           "5",
		"pm.start_servers":          "2",
		"pm.min_spare_servers":      "1",
		"pm.max_spare_servers":      "3",
		"pm.max_requests":           "500",
		"pm.status_path":            "/status",
		"request_terminate_timeout": "120s",
	} {
		if got := www[key]; got != want {
			t.Errorf("www[%s] = %q, want %q", key, got, want)
		}
	}

	// php-fpm reports unset strings as the literal "undefined".
	if got := cfg.Global["pid"]; got != "" {
		t.Errorf("Expected an undefined global to parse as empty, got %q", got)
	}

	// Discovery reads these two out of the same structure.
	if got := www["listen"]; got == "" {
		t.Errorf("Expected the listen socket to be parsed; discovery depends on it")
	}
	if _, ok := cfg.Pools["api"]; !ok {
		t.Errorf("Expected the second pool to be parsed independently")
	}

	// The parser keeps everything; the export filter is what drops secrets.
	if _, ok := www["env[DB_PASSWORD]"]; !ok {
		t.Errorf("Expected env[] to be parsed (and dropped later by exportableConfig)")
	}
	if _, ok := exportableConfig(www)["env[DB_PASSWORD]"]; ok {
		t.Errorf("Expected env[] never to survive the export filter")
	}
}

func TestPoolStatus_RealFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/status-full.json")
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	var pool Pool
	if err := json.Unmarshal(raw, &pool); err != nil {
		t.Fatalf("Failed to decode a real status payload: %v", err)
	}

	if pool.Name != "www" {
		t.Errorf("pool name = %q, want www", pool.Name)
	}
	if pool.ProcessManager != "dynamic" {
		t.Errorf("process manager = %q, want dynamic", pool.ProcessManager)
	}
	if pool.StartTime == 0 {
		t.Errorf("Expected `start time` to decode")
	}
	if pool.AcceptedConnections == 0 {
		t.Errorf("Expected `accepted conn` to decode; the exporter had served at least one request")
	}
	if pool.TotalProcesses == 0 {
		t.Errorf("Expected `total processes` to decode")
	}
	if pool.IdleProcesses+pool.ActiveProcesses == 0 {
		t.Errorf("Expected `idle processes`/`active processes` to decode")
	}
	if len(pool.Processes) == 0 {
		t.Fatalf("Expected the `full` process list to decode")
	}

	proc := pool.Processes[0]
	if proc.PID == 0 {
		t.Errorf("Expected a per-process pid")
	}
	if proc.State == "" {
		t.Errorf("Expected a per-process state")
	}
	if proc.RequestURI == "" {
		t.Errorf("Expected a per-process `request uri`")
	}
}

func TestPoolStatus_MalformedInputIsAnError(t *testing.T) {
	raw, err := os.ReadFile("testdata/status-full.json")
	if err != nil {
		t.Fatalf("Failed to read fixture: %v", err)
	}

	for name, input := range map[string][]byte{
		"empty":     {},
		"truncated": raw[:len(raw)/2],
		"html":      []byte("<html><body>502 Bad Gateway</body></html>"),
	} {
		t.Run(name, func(t *testing.T) {
			var pool Pool
			if err := json.Unmarshal(input, &pool); err == nil {
				t.Errorf("Expected malformed input to be an error, not a zero-valued pool")
			}
		})
	}
}

func keysOf(m map[string]map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
