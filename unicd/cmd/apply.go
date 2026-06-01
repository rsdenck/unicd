package cmd

import (
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply IaC configuration from file",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "file",
		Short: "Apply a YAML/JSON configuration (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func init() {
	rootCmd.AddCommand(newApplyCmd())
}
