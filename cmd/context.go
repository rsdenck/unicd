package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts (tenants/orgs)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List saved contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			for _, c := range cfg.Contexts {
				mark := " "
				if c.Name == cfg.CurrentContext {
					mark = "*"
				}
				fmt.Printf("%s %-20s %-20s %s\n", mark, c.Name, c.Org, c.Host)
			}
			return nil
		},
	})
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "Select a context as default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			for _, c := range cfg.Contexts {
				if c.Name == args[0] {
					cfg.CurrentContext = args[0]
					return config.Save(cfg)
				}
			}
			return fmt.Errorf("context %q not found", args[0])
		},
	}
	cmd.AddCommand(useCmd)
	return cmd
}

func init() {
	rootCmd.AddCommand(newContextCmd())
}
