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

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show vApp details",
		Args:  cobra.ExactArgs(1),
		RunE:  runVappShow,
	}
	cmd.AddCommand(showCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create vApp",
		Args:  cobra.ExactArgs(1),
		RunE:  runVappCreate,
	}
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete vApp",
		Args:  cobra.ExactArgs(1),
		RunE:  runVappDelete,
	}
	cmd.AddCommand(deleteCmd)

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

func runVappShow(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	vapp, err := vdc.GetVAppByName(args[0], true)
	if err != nil {
		return fmt.Errorf("finding vApp %q: %w", args[0], err)
	}
	v := vapp.VApp
	fmt.Printf("Name:        %s\n", v.Name)
	fmt.Printf("ID:          %s\n", v.ID)
	fmt.Printf("Description: %s\n", v.Description)
	fmt.Printf("Status:      %d\n", v.Status)
	fmt.Printf("Deployed:    %t\n", v.Deployed)
	if v.Owner != nil {
		fmt.Printf("Owner:       %s\n", v.Owner.User.Name)
	}
	if v.DateCreated != "" {
		fmt.Printf("Created:     %s\n", v.DateCreated)
	}
	if v.Children != nil {
		fmt.Printf("VMs:         %d\n", len(v.Children.VM))
		for _, vm := range v.Children.VM {
			fmt.Printf("  - %s (%s)\n", vm.Name, vm.ID)
		}
	}
	if v.NetworkConfigSection != nil {
		for _, nc := range v.NetworkConfigSection.NetworkConfig {
			fmt.Printf("Network:     %s (%s)\n", nc.NetworkName, nc.Configuration.ParentNetwork.Name)
		}
	}
	return nil
}

func runVappCreate(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	vapp, err := vdc.CreateRawVApp(args[0], "")
	if err != nil {
		return fmt.Errorf("creating vApp: %w", err)
	}
	fmt.Printf("vApp %q created (ID: %s)\n", vapp.VApp.Name, vapp.VApp.ID)
	return nil
}

func runVappDelete(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	vapp, err := vdc.GetVAppByName(args[0], true)
	if err != nil {
		return fmt.Errorf("finding vApp %q: %w", args[0], err)
	}
	task, err := vapp.Delete()
	if err != nil {
		return fmt.Errorf("deleting vApp: %w", err)
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("vApp %q deleted\n", args[0])
	return nil
}

func init() {
	rootCmd.AddCommand(newVappCmd())
}
