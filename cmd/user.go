package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/client"
	"github.com/spf13/cobra"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List users via CloudAPI",
		RunE:  runUserList,
	})

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show user details",
		RunE:  runUserShow,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(showCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create user",
		RunE:  runUserCreate,
	}
	createCmd.Flags().String("name", "", "Username")
	createCmd.Flags().String("password", "", "Password")
	createCmd.Flags().String("role", "", "Role name")
	createCmd.Flags().String("email", "", "Email address")
	createCmd.Flags().Bool("enabled", false, "Enable user")
	createCmd.Flags().String("full-name", "", "Full name")
	createCmd.Flags().String("description", "", "Description")
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete user",
		RunE:  runUserDelete,
		Args:  cobra.ExactArgs(1),
	}
	cmd.AddCommand(deleteCmd)

	return cmd
}

func getAdminOrgOrExit(cl *client.VCDClient) (*govcd.AdminOrg, error) {
	return cl.VCDClient.GetAdminOrgByName(cl.OrgName)
}

func runUserShow(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := getAdminOrgOrExit(cl)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	user, err := adminOrg.GetUserByName(args[0], true)
	if err != nil {
		return fmt.Errorf("finding user %q: %w", args[0], err)
	}
	u := user.User
	fmt.Printf("Name:         %s\n", u.Name)
	fmt.Printf("ID:           %s\n", u.ID)
	fmt.Printf("Full Name:    %s\n", u.FullName)
	fmt.Printf("Email:        %s\n", u.EmailAddress)
	fmt.Printf("Telephone:    %s\n", u.Telephone)
	fmt.Printf("Enabled:      %t\n", u.IsEnabled)
	fmt.Printf("Locked:       %t\n", u.IsLocked)
	fmt.Printf("External:     %t\n", u.IsExternal)
	fmt.Printf("Provider:     %s\n", u.ProviderType)
	fmt.Printf("Role:         %s\n", user.GetRoleName())
	if u.Role != nil {
		fmt.Printf("Role HREF:    %s\n", u.Role.HREF)
	}
	if u.DeployedVmQuota != 0 {
		fmt.Printf("Deployed VM Quota: %d\n", u.DeployedVmQuota)
	}
	if u.StoredVmQuota != 0 {
		fmt.Printf("Stored VM Quota:  %d\n", u.StoredVmQuota)
	}
	return nil
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	password, _ := cmd.Flags().GetString("password")
	role, _ := cmd.Flags().GetString("role")
	email, _ := cmd.Flags().GetString("email")
	enabled, _ := cmd.Flags().GetBool("enabled")
	fullName, _ := cmd.Flags().GetString("full-name")
	description, _ := cmd.Flags().GetString("description")

	if name == "" || password == "" || role == "" {
		return fmt.Errorf("--name, --password, and --role are required")
	}

	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := getAdminOrgOrExit(cl)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}

	userData := govcd.OrgUserConfiguration{
		Name:         name,
		Password:     password,
		RoleName:     role,
		IsEnabled:    enabled,
		FullName:     fullName,
		Description:  description,
		EmailAddress: email,
	}
	_, err = adminOrg.CreateUserSimple(userData)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	fmt.Printf("User %q created\n", name)
	return nil
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := getAdminOrgOrExit(cl)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	user, err := adminOrg.GetUserByName(args[0], true)
	if err != nil {
		return fmt.Errorf("finding user %q: %w", args[0], err)
	}
	if err := user.Delete(false); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	fmt.Printf("User %q deleted\n", args[0])
	return nil
}

func runUserList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := getAdminOrgOrExit(cl)
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	if err := adminOrg.Refresh(); err != nil {
		return fmt.Errorf("refreshing admin org: %w", err)
	}
	users := adminOrg.AdminOrg.Users.User
	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}
	for _, ref := range users {
		u, err := adminOrg.GetUserByHref(ref.HREF)
		if err != nil {
			fmt.Printf("%-28s (error loading details: %v)\n", ref.Name, err)
			continue
		}
		fmt.Printf("%-28s %-24s %-7t %-8s %s\n", u.User.Name, u.GetRoleName(), u.User.IsEnabled, u.User.ProviderType, u.User.EmailAddress)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newUserCmd())
}
