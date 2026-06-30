package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd() *cobra.Command {
	var user, pass, host, org string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with vCD",
		RunE: func(cmd *cobra.Command, args []string) error {
			if user == "" {
				user = os.Getenv("UNICD_USER")
			}
			if user == "" {
				fmt.Print("Username: ")
				fmt.Scanln(&user)
			}
			if pass == "" {
				pass = os.Getenv("UNICD_PASS")
			}
			if pass == "" {
				fmt.Print("Password: ")
				raw, err := term.ReadPassword(syscall.Stdin)
				if err != nil {
					return fmt.Errorf("reading password: %w", err)
				}
				pass = string(raw)
				fmt.Println()
			}
			if host == "" {
				host = os.Getenv("UNICD_HOST")
			}
			if host == "" {
				host = "vcd-tio.unifique.cloud"
			}
			if org == "" {
				org = os.Getenv("UNICD_ORG")
			}
			if org == "" {
				org = "DENCK_ORG"
			}

			cl, err := client.NewClient(host, user, pass, org, "37.0", true)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			ctx := config.Context{
				Name:       org,
				Host:       host,
				Org:        org,
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

			fmt.Printf("Logged in as %s @ %s (%s)\n", user, org, host)
			return nil
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "Username")
	cmd.Flags().StringVarP(&pass, "pass", "p", "", "Password")
	cmd.Flags().StringVarP(&host, "host", "H", "", "vCD host")
	cmd.Flags().StringVarP(&org, "org", "o", "", "Organization")
	return cmd
}

func init() {
	rootCmd.AddCommand(newLoginCmd())
}
