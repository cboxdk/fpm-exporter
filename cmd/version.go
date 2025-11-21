package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		// Use fmt.Printf for version output (standard for CLI tools)
		fmt.Printf("PHPeek PHP-FPM Exporter version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
