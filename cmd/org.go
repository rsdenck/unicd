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
		Short: "Show organization info",
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
		Short: "List users",
		RunE: runOrgUsersList,
	})
	cmd.AddCommand(usersCmd)

	rolesCmd := &cobra.Command{
		Use:   "roles",
		Short: "Role operations",
	}
	rolesCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List roles",
		RunE: runOrgRolesList,
	})
	cmd.AddCommand(rolesCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "permissions",
		Short: "Show permissions per role",
		RunE: runOrgPermissions,
	})
	RegisterOrgDelCmd(cmd)
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

func runOrgUsersList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := cl.GetAdminOrg()
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	if adminOrg.AdminOrg.Users == nil || len(adminOrg.AdminOrg.Users.User) == 0 {
		fmt.Println("No users found")
		return nil
	}
	for _, u := range adminOrg.AdminOrg.Users.User {
		fmt.Printf("%-24s %s\n", u.Name, u.ID)
	}
	return nil
}

func runOrgRolesList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := cl.VCDClient.GetAdminOrgByName(cl.OrgName)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	roles, err := adminOrg.GetAllRoles(nil)
	if err != nil {
		return fmt.Errorf("listing roles: %w", err)
	}
	for _, r := range roles {
		fmt.Printf("%-24s %s\n", r.Role.Name, r.Role.ID)
	}
	return nil
}

func runOrgPermissions(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := cl.VCDClient.GetAdminOrgByName(cl.OrgName)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	roles, err := adminOrg.GetAllRoles(nil)
	if err != nil {
		return fmt.Errorf("listing roles: %w", err)
	}
	for _, r := range roles {
		fmt.Printf("Role: %s\n", r.Role.Name)
		rights, err := r.GetRights(nil)
		if err != nil {
			fmt.Printf("  rights: %v\n", err)
		} else if len(rights) == 0 {
			fmt.Println("  (no rights assigned)")
		} else {
			for _, rt := range rights {
				fmt.Printf("  - %s\n", rt.Name)
			}
		}
		fmt.Println()
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newOrgCmd())
}
