package cmd

import (
	"encoding/xml"
	"fmt"
	"io"

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
	type userRecord struct {
		Name      string `xml:"name,attr"`
		Role      string `xml:"role,attr"`
		IsEnabled string `xml:"isEnabled,attr"`
		Email     string `xml:"emailAddress,attr"`
	}
	type queryRecords struct {
		User []userRecord `xml:"UserRecord"`
	}
	type queryResult struct {
		Records queryRecords `xml:"QueryResultRecords"`
	}

	href := cl.VCDClient.Client.VCDHREF
	href.Path = "/api/query"
	q := href.Query()
	q.Set("type", "orgUser")
	href.RawQuery = q.Encode()

	req := cl.VCDClient.Client.NewRequestWitNotEncodedParams(nil, nil, "GET", href, nil)
	resp, err := cl.VCDClient.Client.Http.Do(req)
	if err != nil {
		return fmt.Errorf("fetching users: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("fetching users: API error %d: %s", resp.StatusCode, string(data))
	}

	var result queryResult
	if err := xml.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("parsing users: %w\n%s", err, string(data))
	}
	if len(result.Records.User) == 0 {
		fmt.Println("No users found")
		return nil
	}
	for _, u := range result.Records.User {
		fmt.Printf("%-24s %-20s %-5s %s\n", u.Name, u.Role, u.IsEnabled, u.Email)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newUserCmd())
}
