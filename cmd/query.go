package cmd

import (
	"github.com/spf13/cobra"
)

func newQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "VMware Cloud Director API query",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run a query (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func init() {
	rootCmd.AddCommand(newQueryCmd())
}
