package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "unicd",
	Short: "Unifique Cloud Director CLI - vCD Provider",
	Long:  `Unicd is a complete VMware Cloud Director CLI provider for managing vCD resources.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.EnableCommandSorting = false
	rootCmd.PersistentFlags().Bool("no-banner", false, "Suppress the ASCII banner")
	rootCmd.PersistentFlags().MarkHidden("no-banner")
}
