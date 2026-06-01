package cmd

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/spf13/cobra"
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
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show user details (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create user (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete user (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func runUserList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	type userRecord struct {
		Name  string `xml:"name,attr"`
		Role  string `xml:"role,attr"`
		IsEnabled string `xml:"isEnabled,attr"`
		Email string `xml:"emailAddress,attr"`
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
