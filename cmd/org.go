package cmd

import (
	"fmt"
	"os"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Organization operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List organizations",
		RunE: runOrgList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show organization details",
		RunE: runOrgShow,
	})
	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "User operations",
	}
	usersCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List users (not implemented)",
		RunE: stubCmd,
	})
	cmd.AddCommand(usersCmd)

	rolesCmd := &cobra.Command{
		Use:   "roles",
		Short: "Role operations",
	}
	rolesCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List roles (not implemented)",
		RunE: stubCmd,
	})
	cmd.AddCommand(rolesCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "permissions",
		Short: "Show permissions (not implemented)",
		RunE: stubCmd,
	})
	return cmd
}

func getClientFromContext() (*client.VCDClient, *config.Context, error) {
	user := os.Getenv("UNICD_USER")
	pass := os.Getenv("UNICD_PASS")
	host := os.Getenv("UNICD_HOST")
	org := os.Getenv("UNICD_ORG")
	apiVer := os.Getenv("UNICD_API_VERSION")
	if apiVer == "" {
		apiVer = "37.0"
	}

	if user == "" || pass == "" || host == "" || org == "" {
		tomlCfg, err := config.LoadToml()
		if err == nil && tomlCfg != nil {
			if user == "" {
				user = tomlCfg.User
			}
			if pass == "" {
				pass = tomlCfg.Pass
			}
			if host == "" {
				host = tomlCfg.Host
			}
			if org == "" {
				org = tomlCfg.Org
			}
		}
	}

	if user != "" && pass != "" && host != "" && org != "" {
		cl, err := client.NewClient(host, user, pass, org, apiVer, true)
		if err != nil {
			return nil, nil, fmt.Errorf("env auth failed: %w", err)
		}
		cl.OrgName = org
		ctx := &config.Context{
			Name:       org,
			Host:       host,
			Org:        org,
			Username:   user,
			APIVersion: apiVer,
		}
		tomlCfg, _ := config.LoadToml()
		if tomlCfg != nil {
			ctx.VDC = tomlCfg.VDC
		}
		cl.VDCName = ctx.VDC
		return cl, ctx, nil
	}

	// Fall back to session token
	ctx, err := config.GetCurrentContext()
	if err != nil {
		return nil, nil, fmt.Errorf("login first: run 'unicd init' or set env vars")
	}
	session, err := config.LoadSession()
	if err != nil {
		return nil, nil, fmt.Errorf("no session, run 'unicd init' first")
	}
	token := session["token"]
	authHeader := session["auth"]
	if authHeader == "" {
		authHeader = "x-vcloud-authorization"
	}
	cl, err := client.NewClientWithToken(ctx.Host, ctx.Org, authHeader, token, ctx.APIVersion, true)
	if err != nil {
		return nil, nil, fmt.Errorf("connecting: %w", err)
	}
	cl.VDCName = ctx.VDC
	return cl, ctx, nil
}

func stubCmd(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not implemented yet")
}

func runOrgList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	fmt.Printf("Name: %s | ID: %s\n", org.Org.Name, org.Org.ID)
	return nil
}

func runOrgShow(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	o := org.Org
	fmt.Printf("Name:        %s\n", o.Name)
	fmt.Printf("ID:          %s\n", o.ID)
	fmt.Printf("Description: %s\n", o.Description)
	fmt.Printf("Full Name:   %s\n", o.FullName)
	if o.Link != nil {
		for _, l := range o.Link {
			fmt.Printf("  Link: %s (%s)\n", l.Name, l.Rel)
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newOrgCmd())
}
