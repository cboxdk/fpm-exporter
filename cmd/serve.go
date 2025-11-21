package cmd

import (
	"github.com/gophpeek/phpeek-fpm-exporter/internal/logging"

	"github.com/gophpeek/phpeek-fpm-exporter/internal/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start agent HTTP server with metrics and control endpoints",
	Run: func(cmd *cobra.Command, args []string) {
		logging.L().Info("PHPeek Starting")
		serve.StartPrometheusServer(Config)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
