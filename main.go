package main

import (
	"fmt"

	"github.com/cboxdk/fpm-exporter/cmd"
	"github.com/cboxdk/fpm-exporter/internal/serve"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
	// Exported as fpm_exporter_build_info so a partial rollout is visible in
	// Prometheus rather than over ssh.
	serve.Version = version
	cmd.Execute()
}
