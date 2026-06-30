package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVappCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vapp",
		Short: "vApp operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List vApps",
		RunE:  runVappList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show vApp details (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create vApp (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete vApp (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func runVappList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	refs := vdc.GetVappList()
	if len(refs) == 0 {
		fmt.Println("No vApps found")
		return nil
	}
	for _, ref := range refs {
		fmt.Printf("%-30s %s\n", ref.Name, ref.HREF)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newVappCmd())
}
