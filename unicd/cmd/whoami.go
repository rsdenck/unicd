package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current user/session info",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := config.GetCurrentContext()
			if err != nil {
				return fmt.Errorf("not logged in: %w", err)
			}
			session, _ := config.LoadSession()
			fmt.Printf("User:     %s\n", ctx.Username)
			fmt.Printf("Org:      %s\n", ctx.Org)
			fmt.Printf("Host:     %s\n", ctx.Host)
			fmt.Printf("VDC:      %s\n", ctx.VDC)
			fmt.Printf("API:      %s\n", ctx.APIVersion)
			if session != nil {
				fmt.Printf("Token:    %s...\n", truncate(session["token"], 20))
			}
			return nil
		},
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func init() {
	rootCmd.AddCommand(newWhoamiCmd())
}
