package cmd

import (
	"fmt"

	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"github.com/spf13/cobra"
)

func newVDCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vdc",
		Short: "VDC operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List VDCs",
		RunE: runVDCList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show VDC details",
		RunE: runVDCShow,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create VDC (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete VDC (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Update VDC (not implemented)",
		RunE:  stubCmd,
	})

	vappCmd := &cobra.Command{
		Use:   "vapp",
		Short: "vApp operations",
	}
	vappCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List vApps",
		RunE:  runVdcVAppList,
	})
	vappCmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete a vApp by name",
		RunE:  runVdcVAppDelete,
	})
	cmd.AddCommand(vappCmd)

	storageCmd := &cobra.Command{
		Use:   "storage",
		Short: "Storage policy operations",
	}
	storageCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List storage policies",
		RunE:  runVDCStorageList,
	})
	cmd.AddCommand(storageCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "resource",
		Short: "Show VDC resource allocation",
		RunE:  stubCmd,
	})
	return cmd
}

func runVdcVAppList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	vms, err := vdc.QueryVmList(types.VmQueryFilterAll)
	if err != nil {
		return err
	}
	vappMap := map[string]int{}
	for _, vm := range vms {
		vappMap[vm.ContainerName]++
	}
	for name, count := range vappMap {
		fmt.Printf("%-30s VMs:%d\n", name, count)
	}
	return nil
}

func runVdcVAppDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("vApp name is required")
	}
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

func runVDCList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	vdcs, err := org.QueryOrgVdcList()
	if err != nil {
		return err
	}
	for _, v := range vdcs {
		vmCount := 0
		if v.NumberOfVMs != nil {
			vmCount = *v.NumberOfVMs
		}
		fmt.Printf("%-30s | Status: %s | VMs: %d\n", v.Name, v.Status, vmCount)
	}
	return nil
}

func runVDCShow(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	v := vdc.Vdc
	fmt.Printf("Name:        %s\n", v.Name)
	fmt.Printf("ID:          %s\n", v.ID)
	fmt.Printf("Description: %s\n", v.Description)
	fmt.Printf("Status:      %d\n", v.Status)
	fmt.Printf("Allocation:  %s\n", v.AllocationModel)
	for _, cc := range v.ComputeCapacity {
		if cc.CPU != nil {
			fmt.Printf("CPU (MHz):   %d/%d/%d (alloc/limit/used)\n", cc.CPU.Allocated, cc.CPU.Limit, cc.CPU.Used)
		}
		if cc.Memory != nil {
			fmt.Printf("Memory (MB): %d/%d/%d (alloc/limit/used)\n", cc.Memory.Allocated, cc.Memory.Limit, cc.Memory.Used)
		}
	}
	if v.NicQuota != 0 {
		fmt.Printf("Nic Quota:   %d\n", v.NicQuota)
	}
	if v.NetworkQuota != 0 {
		fmt.Printf("Net Quota:   %d\n", v.NetworkQuota)
	}
	return nil
}

func runVDCStorageList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	if vdc.Vdc.VdcStorageProfiles == nil {
		fmt.Println("No storage profiles found")
		return nil
	}
	for _, sp := range vdc.Vdc.VdcStorageProfiles.VdcStorageProfile {
		fmt.Printf("%-30s %s\n", sp.Name, sp.ID)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newVDCCmd())
}
