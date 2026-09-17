package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func newVDCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vdc",
		Short: "VDC operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List VDCs",
		RunE:  runVDCList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show VDC details",
		RunE:  runVDCShow,
	})

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create VDC (admin)",
		RunE:  runVDCCreate,
	}
	createCmd.Flags().String("user", "", "User in org")
	createCmd.Flags().String("password", "", "Password")
	createCmd.Flags().String("host", "", "vCD host")
	createCmd.Flags().String("org", "", "Organization")
	createCmd.Flags().String("name", "", "VDC name")
	createCmd.Flags().String("allocation", "AllocationPool", "Allocation model")
	createCmd.Flags().String("cpu-units", "MHz", "CPU units")
	createCmd.Flags().Int64("cpu-limit", 0, "CPU limit")
	createCmd.Flags().Int64("cpu-reserved", 0, "CPU reserved")
	createCmd.Flags().Int64("mem-limit", 0, "Memory limit (MB)")
	createCmd.Flags().Int64("mem-reserved", 0, "Memory reserved (MB)")
	createCmd.Flags().String("provider-vdc", "", "Provider VDC HREF")
	createCmd.Flags().String("storage-profile", "", "Storage profile HREF")
	createCmd.Flags().String("network-pool", "", "Network pool HREF")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("provider-vdc")
	createCmd.MarkFlagRequired("storage-profile")
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete VDC (admin)",
		Args:  cobra.ExactArgs(1),
		RunE:  runVDCDelete,
	}
	deleteCmd.Flags().Bool("force", true, "Force delete")
	deleteCmd.Flags().Bool("recursive", true, "Recursive delete")
	cmd.AddCommand(deleteCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update VDC (admin)",
		RunE:  runVDCUpdate,
	}
	updateCmd.Flags().String("name", "", "VDC name")
	updateCmd.Flags().String("description", "", "Description")
	updateCmd.Flags().String("new-name", "", "New VDC name")
	updateCmd.Flags().Bool("enable", true, "Enable VDC")
	updateCmd.MarkFlagRequired("name")
	cmd.AddCommand(updateCmd)

	vappCmd := &cobra.Command{
		Use:   "vapp",
		Short: "vApp operations within VDC",
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
		RunE:  runVDCResource,
	})
	return cmd
}

func runVDCResource(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	v := vdc.Vdc
	fmt.Printf("Name:            %s\n", v.Name)
	fmt.Printf("ID:              %s\n", v.ID)
	fmt.Printf("Description:     %s\n", v.Description)
	fmt.Printf("Status:          %d\n", v.Status)
	fmt.Printf("Allocation:      %s\n", v.AllocationModel)
	fmt.Printf("Enabled:         %t\n", v.IsEnabled)

	for _, cc := range v.ComputeCapacity {
		if cc.CPU != nil {
			fmt.Printf("CPU (MHz):       alloc=%d limit=%d used=%d reserved=%d\n", cc.CPU.Allocated, cc.CPU.Limit, cc.CPU.Used, cc.CPU.Reserved)
		}
		if cc.Memory != nil {
			fmt.Printf("Memory (MB):     alloc=%d limit=%d used=%d reserved=%d\n", cc.Memory.Allocated, cc.Memory.Limit, cc.Memory.Used, cc.Memory.Reserved)
		}
	}
	if v.NicQuota != 0 {
		fmt.Printf("Nic Quota:       %d\n", v.NicQuota)
	}
	if v.NetworkQuota != 0 {
		fmt.Printf("Net Quota:       %d\n", v.NetworkQuota)
	}
	if v.VMQuota != 0 {
		fmt.Printf("VM Quota:        %d\n", v.VMQuota)
	}
	if v.VdcStorageProfiles != nil {
		fmt.Println("Storage Profiles:")
		for _, sp := range v.VdcStorageProfiles.VdcStorageProfile {
			fmt.Printf("  - %-30s %s\n", sp.Name, sp.ID)
		}
	}
	return nil
}

func runVDCCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	allocation, _ := cmd.Flags().GetString("allocation")
	cpuLimit, _ := cmd.Flags().GetInt64("cpu-limit")
	cpuReserved, _ := cmd.Flags().GetInt64("cpu-reserved")
	cpuUnits, _ := cmd.Flags().GetString("cpu-units")
	memLimit, _ := cmd.Flags().GetInt64("mem-limit")
	memReserved, _ := cmd.Flags().GetInt64("mem-reserved")
	providerVDC, _ := cmd.Flags().GetString("provider-vdc")
	storageProfile, _ := cmd.Flags().GetString("storage-profile")
	networkPool, _ := cmd.Flags().GetString("network-pool")

	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := cl.GetAdminOrg()
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	config := &types.VdcConfiguration{
		Name:            name,
		AllocationModel: allocation,
		IsEnabled:       true,
	}
	if cpuLimit > 0 || cpuReserved > 0 {
		config.ComputeCapacity = append(config.ComputeCapacity, &types.ComputeCapacity{
			CPU: &types.CapacityWithUsage{
				Units:    cpuUnits,
				Limit:    cpuLimit,
				Reserved: cpuReserved,
			},
			Memory: &types.CapacityWithUsage{
				Units:    "MB",
				Limit:    memLimit,
				Reserved: memReserved,
			},
		})
	}
	if providerVDC != "" {
		config.ProviderVdcReference = &types.Reference{HREF: providerVDC}
	}
	if storageProfile != "" {
		config.VdcStorageProfile = append(config.VdcStorageProfile, &types.VdcStorageProfileConfiguration{
			Enabled:                   &[]bool{true}[0],
			Default:                   true,
			Limit:                     0,
			Units:                     "MB",
			ProviderVdcStorageProfile: &types.Reference{HREF: storageProfile},
		})
	}
	if networkPool != "" {
		config.NetworkPoolReference = &types.Reference{HREF: networkPool}
	}
	_, err = adminOrg.CreateOrgVdc(config)
	if err != nil {
		return fmt.Errorf("creating VDC: %w", err)
	}
	fmt.Printf("VDC %q created\n", name)
	return nil
}

func runVDCDelete(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	recursive, _ := cmd.Flags().GetBool("recursive")
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	_, err = vdc.Delete(force, recursive)
	if err != nil {
		return fmt.Errorf("deleting VDC: %w", err)
	}
	fmt.Printf("VDC %s deleted\n", ctx.VDC)
	return nil
}

func runVDCUpdate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	newName, _ := cmd.Flags().GetString("new-name")
	enable, _ := cmd.Flags().GetBool("enable")

	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	adminOrg, err := cl.GetAdminOrg()
	if err != nil {
		return fmt.Errorf("getting admin org: %w", err)
	}
	adminVdc, err := adminOrg.GetAdminVDCByName(name, true)
	if err != nil {
		return fmt.Errorf("finding VDC %q: %w", name, err)
	}
	if description != "" {
		adminVdc.AdminVdc.Description = description
	}
	if newName != "" {
		adminVdc.AdminVdc.Name = newName
	}
	adminVdc.AdminVdc.IsEnabled = enable
	_, err = adminVdc.Update()
	if err != nil {
		return fmt.Errorf("updating VDC: %w", err)
	}
	fmt.Printf("VDC %q updated\n", name)
	return nil
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
