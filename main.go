package main

import (
	"fmt"
	"github.com/cboxdk/fpm-exporter/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
	cmd.Execute()
}
