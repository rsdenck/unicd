package cmd

import (
	"fmt"

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
