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
			if err := config.ClearSession(); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.CurrentContext = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println("Logged out")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newLogoutCmd())
}
