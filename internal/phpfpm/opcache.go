package phpfpm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cboxdk/fpm-exporter/internal/config"
	"github.com/elasticphphq/fcgx"
	"os"
	"path/filepath"
	"sync"
)

type OpcacheStatus struct {
	Enabled     bool   `json:"opcache_enabled"`
	MemoryUsage Memory `json:"memory_usage"`
	Statistics  Stats  `json:"opcache_statistics"`
}

type Memory struct {
	UsedMemory       uint64  `json:"used_memory"`
	FreeMemory       uint64  `json:"free_memory"`
	WastedMemory     uint64  `json:"wasted_memory"`
	CurrentWastedPct float64 `json:"current_wasted_percentage"`
}

type Stats struct {
	NumCachedScripts uint64  `json:"num_cached_scripts"`
	Hits             uint64  `json:"hits"`
	Misses           uint64  `json:"misses"`
	BlacklistMisses  uint64  `json:"blacklist_misses"`
	OomRestarts      uint64  `json:"oom_restarts"`
	HashRestarts     uint64  `json:"hash_restarts"`
	ManualRestarts   uint64  `json:"manual_restarts"`
	HitRate          float64 `json:"opcache_hit_rate"`
}

const opcacheScriptContent = `<?php
error_reporting(0);
ini_set('display_errors', 0);
header("Status: 200 OK");
header("Content-Type: application/json");
echo json_encode(opcache_get_status());
exit;`

var (
	opcacheScriptOnce sync.Once
	opcacheScriptPath string
	opcacheScriptErr  error
)

// opcacheScript writes the status script PHP-FPM is asked to execute, and returns
// its path. The file is created once per process under a random name with O_EXCL,
// so a local user cannot pre-create (or symlink) the path we are about to hand to
// PHP-FPM and have their own code executed as the pool user. It is deliberately
// world-readable: PHP-FPM usually runs as a different user than the exporter, and
// the contents are fixed and non-secret.
func opcacheScript() (string, error) {
	opcacheScriptOnce.Do(func() {
		f, err := os.CreateTemp("", "fpm-exporter-opcache-*.php")
		if err != nil {
			opcacheScriptErr = fmt.Errorf("failed to create PHP script: %w", err)
			return
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && opcacheScriptErr == nil {
				opcacheScriptErr = fmt.Errorf("failed to close PHP script: %w", cerr)
			}
		}()

		if _, err := f.WriteString(opcacheScriptContent); err != nil {
			_ = os.Remove(f.Name())
			opcacheScriptErr = fmt.Errorf("failed to write PHP script: %w", err)
			return
		}

		if err := f.Chmod(0644); err != nil {
			_ = os.Remove(f.Name())
			opcacheScriptErr = fmt.Errorf("failed to chmod PHP script: %w", err)
			return
		}

		opcacheScriptPath = f.Name()
	})

	return opcacheScriptPath, opcacheScriptErr
}

// RemoveOpcacheScript deletes the generated status script. Safe to call when no
// script was ever written.
func RemoveOpcacheScript() {
	if opcacheScriptPath != "" {
		_ = os.Remove(opcacheScriptPath)
	}
}

func GetOpcacheStatus(ctx context.Context, cfg config.FPMPoolConfig) (*OpcacheStatus, error) {
	tmpPath, err := opcacheScript()
	if err != nil {
		return nil, err
	}

	scheme, address, _, err := ParseAddress(cfg.StatusSocket, "")
	if err != nil {
		return nil, fmt.Errorf("invalid socket: %w", err)
	}

	client, err := fcgx.DialContext(ctx, scheme, address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial FPM: %w", err)
	}
	defer func() { _ = client.Close() }()

	scriptPath := tmpPath
	env := map[string]string{
		"SCRIPT_FILENAME": scriptPath,
		"SCRIPT_NAME":     "/" + filepath.Base(scriptPath),
		"SERVER_SOFTWARE": "fpm-exporter",
		"REMOTE_ADDR":     "127.0.0.1",
	}

	resp, err := client.Get(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("fcgi GET failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := fcgx.ReadBody(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read opcache response: %w", err)
	}

	var status OpcacheStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse opcache JSON: %w", err)
	}

	return &status, nil
}
