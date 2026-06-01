package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local session",
		RunE: func(cmd *cobra.Command, args []string) error {
			config.ClearSession()
			cfg, _ := config.Load()
			cfg.CurrentContext = ""
			config.Save(cfg)
			fmt.Println("Logged out")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newLogoutCmd())
}
