package cmd

import (
	"github.com/spf13/cobra"
)

func newClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Advanced cluster operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List clusters (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create cluster (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func init() {
	rootCmd.AddCommand(newClusterCmd())
}
