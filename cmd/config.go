package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	})
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration value",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("use 'config set <key>=<value>' (not implemented)")
		},
	}
	cmd.AddCommand(setCmd)
	return cmd
}

func init() {
	rootCmd.AddCommand(newConfigCmd())
}
