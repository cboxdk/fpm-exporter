package cmd

import (
	"github.com/cboxdk/fpm-exporter/internal/logging"

	"github.com/cboxdk/fpm-exporter/internal/serve"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// --web.listen-address is what every exporter chart, unit file and manifest
	// template in the ecosystem expects. Until now the address was reachable
	// only through a config file.
	serveCmd.Flags().String("web.listen-address", "", "Address to serve metrics on (overrides monitor.listen_addr)")
	_ = viper.BindPFlag("monitor.listen_addr", serveCmd.Flags().Lookup("web.listen-address"))

	rootCmd.AddCommand(serveCmd)
}
