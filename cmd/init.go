package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var host string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize CLI with interactive setup",
		Long: `First-time setup command. Prompts for organization, VDC, user, and password.
Saves persistent configuration to ~/.unicd/config.toml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				host = "vcd-tio.unifique.cloud"
			}

			fmt.Print("Organization: ")
			var org string
			fmt.Scanln(&org)
			if org == "" {
				org = "DENCK_ORG"
				fmt.Printf("  [using default: %s]\n", org)
			}

			fmt.Print("VDC name: ")
			var vdc string
			fmt.Scanln(&vdc)
			if vdc == "" {
				return fmt.Errorf("VDC name is required")
			}

			fmt.Print("Username: ")
			var user string
			fmt.Scanln(&user)
			if user == "" {
				return fmt.Errorf("username is required")
			}

			fmt.Print("Password: ")
			var pass string
			fmt.Scanln(&pass)
			if pass == "" {
				return fmt.Errorf("password is required")
			}

			fmt.Println("\nAuthenticating...")
			cl, err := client.NewClient(host, user, pass, org, "37.0", true)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			tomlCfg := &config.TomlConfig{
				Org:  org,
				VDC:  vdc,
				User: user,
				Pass: pass,
				Host: host,
			}
			if err := config.SaveToml(tomlCfg); err != nil {
				return fmt.Errorf("saving config.toml: %w", err)
			}

			ctx := config.Context{
				Name:       org,
				Host:       host,
				Org:        org,
				VDC:        vdc,
				Username:   user,
				APIVersion: "37.0",
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			found := false
			for i, c := range cfg.Contexts {
				if c.Name == org {
					cfg.Contexts[i] = ctx
					found = true
					break
				}
			}
			if !found {
				cfg.Contexts = append(cfg.Contexts, ctx)
			}
			cfg.CurrentContext = org

			if err := config.Save(cfg); err != nil {
				return err
			}

			session := map[string]string{
				"token": cl.VCDClient.Client.VCDToken,
				"auth":  cl.VCDClient.Client.VCDAuthHeader,
			}
			if err := config.SaveSession(session); err != nil {
				return err
			}

			fmt.Printf("\nInitialized: %s @ %s (VDC: %s)\n", user, org, vdc)
			return nil
		},
	}
	cmd.Flags().StringVarP(&host, "host", "H", "", "vCD host (default: vcd-tio.unifique.cloud)")
	return cmd
}

func init() {
	rootCmd.AddCommand(newInitCmd())
}
