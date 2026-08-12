package cmd

import (
	"github.com/cboxdk/fpm-exporter/internal/logging"

	"github.com/cboxdk/fpm-exporter/internal/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start agent HTTP server with metrics and control endpoints",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.L().Info("Cbox Starting")
		return serve.StartPrometheusServer(Config)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
