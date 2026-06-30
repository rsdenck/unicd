package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"github.com/spf13/cobra"
)

func newVmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Virtual machine operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List VMs",
		RunE:  runVmList,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show VM details and NICs",
		RunE:  runVmShow,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "poweron",
		Short: "Power on a VM",
		RunE:  runVmPowerOn,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "poweroff",
		Short: "Power off a VM",
		RunE:  runVmPowerOff,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "reboot",
		Short: "Reboot a VM",
		RunE:  runVmReboot,
	})
	resizeCmd := &cobra.Command{
		Use:   "resize",
		Short: "Change CPU and memory of a VM",
		RunE:  runVmResize,
	}
	resizeCmd.Flags().IntP("cpu", "", 0, "Number of CPUs")
	resizeCmd.Flags().IntP("memory", "", 0, "Memory in MB")
	cmd.AddCommand(resizeCmd)
	diskCmd := &cobra.Command{
		Use:   "disk-resize",
		Short: "Resize a VM disk",
		RunE:  runVmDiskResize,
	}
	diskCmd.Flags().IntP("size", "s", 0, "New disk size in GB")
	diskCmd.Flags().IntP("id", "", 0, "Disk ID (0 = first disk)")
	cmd.AddCommand(diskCmd)
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a VM and its vApp",
		RunE:  runVmDelete,
	}
	cmd.AddCommand(deleteCmd)
	tplCmd := &cobra.Command{
		Use:   "template",
		Short: "List available vApp templates",
		RunE:  runVmTemplateList,
	}
	tplCmd.Flags().StringP("catalog", "", "", "Catalog to list templates from")
	cmd.AddCommand(tplCmd)
	deployCmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a VM from a template",
		RunE:  runVmDeploy,
	}
	deployCmd.Flags().StringP("network", "", "", "Network name to attach")
	deployCmd.Flags().StringP("ip", "", "", "Static IP address")
	deployCmd.Flags().StringP("mode", "", "POOL", "IP mode: POOL, DHCP, MANUAL, NONE")
	deployCmd.Flags().StringP("catalog", "", "", "Catalog containing the template")
	cmd.AddCommand(deployCmd)
	netCmd := &cobra.Command{
		Use:   "attach-net",
		Short: "Attach a network NIC to a VM",
		RunE:  runVmAttachNet,
	}
	netCmd.Flags().StringP("network", "", "", "Network name")
	netCmd.Flags().StringP("ip", "", "", "IP address (empty for POOL/DHCP)")
	netCmd.Flags().StringP("mode", "", "POOL", "IP mode: POOL, DHCP, MANUAL, NONE")
	netCmd.Flags().IntP("index", "", -1, "NIC index (omit for auto)")
	cmd.AddCommand(netCmd)

	syncCmd := &cobra.Command{
		Use:   "sync-nics",
		Short: "Copy NIC layout from one VM to another",
		RunE:  runVmSyncNics,
	}
	cmd.AddCommand(syncCmd)
	cmd.AddCommand(newVmMediaCmd())
	return cmd
}

func findVM(cl *client.VCDClient, ctx *config.Context, vmName string) (*govcd.VM, error) {
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return nil, err
	}
	vms, err := vdc.QueryVmList(types.VmQueryFilterAll)
	if err != nil {
		return nil, err
	}
	var containerName string
	for _, vm := range vms {
		if vm.Name == vmName {
			containerName = vm.ContainerName
			break
		}
	}
	if containerName == "" {
		return nil, fmt.Errorf("VM %q not found", vmName)
	}
	vapp, err := vdc.GetVAppByName(containerName, true)
	if err != nil {
		return nil, err
	}
	govm, err := vapp.GetVMByName(vmName, true)
	if err != nil {
		return nil, err
	}
	return govm, nil
}

func runVmList(cmd *cobra.Command, args []string) error {
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
	for _, vm := range vms {
		ip := vm.IpAddress
		if ip == "" {
			ip = "-"
		}
		fmt.Printf("%-30s %-20s cpu:%-2d mem:%-4dMB IP:%-15s %s\n", vm.Name, vm.ContainerName, vm.Cpus, vm.MemoryMB, ip, vm.Status)
	}
	return nil
}

func runVmShow(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	vm := govm.VM
	status, _ := govm.GetStatus()
	fmt.Printf("Name:       %s\n", vm.Name)
	fmt.Printf("Status:     %s\n", status)
	if vm.VAppParent != nil {
		fmt.Printf("VApp:       %s\n", vm.VAppParent.Name)
	}
	if vm.VmSpecSection != nil && vm.VmSpecSection.NumCpus != nil {
		fmt.Printf("CPU:        %d\n", *vm.VmSpecSection.NumCpus)
	}
	if vm.VmSpecSection != nil && vm.VmSpecSection.MemoryResourceMb != nil {
		fmt.Printf("Memory:     %d MB\n", vm.VmSpecSection.MemoryResourceMb.Configured)
	}
	netSection, err := govm.GetNetworkConnectionSection()
	if err != nil {
		return err
	}
	fmt.Println("NICs:")
	for _, nic := range netSection.NetworkConnection {
		fmt.Printf("  [%d] %-20s IP:%-15s MAC:%s Connected:%v\n",
			nic.NetworkConnectionIndex, nic.Network, nic.IPAddress, nic.MACAddress, nic.IsConnected)
	}
	return nil
}

func runVmPowerOn(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	task, err := govm.PowerOn()
	if err != nil {
		return err
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("VM %q powered on\n", args[0])
	return nil
}

func runVmPowerOff(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	task, err := govm.PowerOff()
	if err != nil {
		return err
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("VM %q powered off\n", args[0])
	return nil
}

func runVmReboot(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	task, err := govm.PowerOff()
	if err != nil {
		return err
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	task, err = govm.PowerOn()
	if err != nil {
		return err
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("VM %q rebooted\n", args[0])
	return nil
}

func runVmAttachNet(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	network, _ := cmd.Flags().GetString("network")
	if network == "" {
		return fmt.Errorf("--network is required")
	}
	ip := cmd.Flag("ip").Value.String()
	if ip == "" {
		ip, _ = cmd.Flags().GetString("ip")
	}
	mode, _ := cmd.Flags().GetString("mode")
	idx, _ := cmd.Flags().GetInt("index")

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	netSection, err := govm.GetNetworkConnectionSection()
	if err != nil {
		return err
	}
	if idx < 0 {
		idx = 0
		for _, n := range netSection.NetworkConnection {
			if n.NetworkConnectionIndex >= idx {
				idx = n.NetworkConnectionIndex + 1
			}
		}
	} else {
		for i, n := range netSection.NetworkConnection {
			if n.NetworkConnectionIndex == idx {
				netSection.NetworkConnection = append(netSection.NetworkConnection[:i], netSection.NetworkConnection[i+1:]...)
				break
			}
		}
	}
	ipType := "IPV4"
	if mode == "NONE" {
		mode = "NONE"
		ipType = ""
	}
	conn := &types.NetworkConnection{
		NetworkConnectionIndex:  idx,
		Network:                 network,
		IPAddress:               ip,
		IpType:                  ipType,
		IsConnected:             true,
		IPAddressAllocationMode: mode,
	}
	netSection.NetworkConnection = append(netSection.NetworkConnection, conn)
	if err := govm.UpdateNetworkConnectionSection(netSection); err != nil {
		return fmt.Errorf("adding NIC: %w", err)
	}
	fmt.Printf("NIC [%d] %s attached to %s\n", idx, network, args[0])
	return nil
}

func runVmSyncNics(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: unicd vm sync-nics <sourceVM> <targetVM>")
	}
	srcName := args[0]
	dstName := args[1]

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	srcVM, err := findVM(cl, ctx, srcName)
	if err != nil {
		return err
	}
	dstVM, err := findVM(cl, ctx, dstName)
	if err != nil {
		return err
	}
	srcSection, err := srcVM.GetNetworkConnectionSection()
	if err != nil {
		return err
	}
	dstSection, err := dstVM.GetNetworkConnectionSection()
	if err != nil {
		return err
	}
	nicMap := map[int]*types.NetworkConnection{}
	for _, n := range dstSection.NetworkConnection {
		nicMap[n.NetworkConnectionIndex] = n
	}
	var newNics []*types.NetworkConnection
	for _, srcConn := range srcSection.NetworkConnection {
		idx := srcConn.NetworkConnectionIndex
		if existing, ok := nicMap[idx]; ok && existing.Network == srcConn.Network {
			fmt.Printf("  [%d] %s already exists (IP: %s)\n", idx, srcConn.Network, existing.IPAddress)
			newNics = append(newNics, existing)
			continue
		}
		newConn := &types.NetworkConnection{
			NetworkConnectionIndex:  idx,
			Network:                 srcConn.Network,
			IPAddress:               "",
			IpType:                  "IPV4",
			IsConnected:             true,
			IPAddressAllocationMode: "POOL",
		}
		newNics = append(newNics, newConn)
		fmt.Printf("  [%d] %s added to target\n", idx, srcConn.Network)
	}
	dstSection.NetworkConnection = newNics
	if err := dstVM.UpdateNetworkConnectionSection(dstSection); err != nil {
		return fmt.Errorf("syncing NICs: %w", err)
	}
	fmt.Printf("NICs synced from %s to %s\n", srcName, dstName)
	return nil
}

func findVApp(cl *client.VCDClient, ctx *config.Context, vmName string) (*govcd.VApp, error) {
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return nil, err
	}
	vms, err := vdc.QueryVmList(types.VmQueryFilterAll)
	if err != nil {
		return nil, err
	}
	var containerName string
	for _, vm := range vms {
		if vm.Name == vmName {
			containerName = vm.ContainerName
			break
		}
	}
	if containerName == "" {
		return nil, fmt.Errorf("VM %q not found", vmName)
	}
	return vdc.GetVAppByName(containerName, true)
}

func runVmDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
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
		vapp, err = findVApp(cl, ctx, args[0])
		if err != nil {
			return fmt.Errorf("finding vApp for %q: %w", args[0], err)
		}
	}
	powerOffVApp(vapp)
	if err := undeployVApp(vapp); err != nil {
		return err
	}
	task, err := vapp.Delete()
	if err != nil {
		return fmt.Errorf("deleting vApp: %w", err)
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("VM %q deleted\n", args[0])
	return nil
}

func powerOffVApp(vapp *govcd.VApp) error {
	status, err := vapp.GetStatus()
	if err != nil {
		return nil
	}
	if status == "POWERED_ON" || status == "MIXED" {
		task, err := vapp.PowerOff()
		if err != nil {
			return fmt.Errorf("power off vApp: %w", err)
		}
		return task.WaitTaskCompletion()
	}
	return nil
}

func undeployVApp(vapp *govcd.VApp) error {
	status, err := vapp.GetStatus()
	if err != nil {
		return nil
	}
	if status != "UNRESOLVED" && status != "FAILED_CREATION" {
		task, err := vapp.Undeploy()
		if err != nil {
			return fmt.Errorf("undeploy: %w", err)
		}
		return task.WaitTaskCompletion()
	}
	return nil
}

func runVmTemplateList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	catalogName, _ := cmd.Flags().GetString("catalog")
	if catalogName != "" {
		org, err := cl.GetOrg()
		if err != nil {
			return err
		}
		catalog, err := org.GetCatalogByName(catalogName, false)
		if err != nil {
			return fmt.Errorf("finding catalog %q: %w", catalogName, err)
		}
		for _, ci := range catalog.Catalog.CatalogItems {
			for _, item := range ci.CatalogItem {
				fmt.Printf("%-40s\n", item.Name)
			}
		}
	} else {
		vdc, err := cl.GetVDC(ctx.VDC)
		if err != nil {
			return err
		}
		templates, err := vdc.QueryVappTemplateList()
		if err != nil {
			return err
		}
		for _, t := range templates {
			fmt.Printf("%-40s catalog:%-20s status:%-10s\n", t.Name, t.CatalogName, t.Status)
		}
	}
	return nil
}

func runVmDeploy(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: unicd vm deploy <vmname> <template> [--network NET] [--ip IP] [--mode MODE] [--catalog CAT]")
	}
	vmName := args[0]
	templateName := args[1]
	network, _ := cmd.Flags().GetString("network")
	ip, _ := cmd.Flags().GetString("ip")
	mode, _ := cmd.Flags().GetString("mode")
	catalogName, _ := cmd.Flags().GetString("catalog")

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}

	var vappTemplate govcd.VAppTemplate
	if catalogName != "" {
		org, err := cl.GetOrg()
		if err != nil {
			return err
		}
		catalog, err := org.GetCatalogByName(catalogName, false)
		if err != nil {
			return fmt.Errorf("finding catalog %q: %w", catalogName, err)
		}
		catalogItem, err := catalog.GetCatalogItemByName(templateName, false)
		if err != nil {
			return fmt.Errorf("finding catalog item %q: %w", templateName, err)
		}
		vappTemplate, err = catalogItem.GetVAppTemplate()
		if err != nil {
			return fmt.Errorf("getting vApp template: %w", err)
		}
	} else {
		t, err := vdc.GetVAppTemplateByName(templateName)
		if err != nil {
			return fmt.Errorf("finding template %q: %w", templateName, err)
		}
		vappTemplate = *t
	}

	vapp, err := vdc.CreateRawVApp(vmName, "")
	if err != nil {
		return fmt.Errorf("creating raw vApp: %w", err)
	}

	if network != "" {
		net, err := vdc.GetOrgVdcNetworkByName(network, false)
		if err != nil {
			return fmt.Errorf("finding network %q: %w", network, err)
		}
		task, err := vapp.AddRAWNetworkConfig([]*types.OrgVDCNetwork{net.OrgVDCNetwork})
		if err != nil {
			return fmt.Errorf("attaching network: %w", err)
		}
		if err := task.WaitTaskCompletion(); err != nil {
			return err
		}
	}

	netConfig := &types.NetworkConnectionSection{
		PrimaryNetworkConnectionIndex: 0,
	}
	if network != "" {
		netConfig.NetworkConnection = append(netConfig.NetworkConnection, &types.NetworkConnection{
			NetworkConnectionIndex:  0,
			Network:                 network,
			IPAddress:               ip,
			IpType:                  "IPV4",
			IsConnected:             true,
			IPAddressAllocationMode: mode,
		})
	}

	task, err := vapp.AddNewVM(vmName, vappTemplate, netConfig, true)
	if err != nil {
		return fmt.Errorf("adding VM from template: %w", err)
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}

	fmt.Printf("VM %q deployed\n", vmName)
	return nil
}

func runVmResize(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	cpu, _ := cmd.Flags().GetInt("cpu")
	memory, _ := cmd.Flags().GetInt("memory")
	if cpu == 0 && memory == 0 {
		return fmt.Errorf("specify --cpu, --memory, or both")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	if cpu > 0 {
		if err := govm.ChangeCPU(cpu, 1); err != nil {
			return fmt.Errorf("changing CPU: %w", err)
		}
		fmt.Printf("CPU set to %d\n", cpu)
	}
	if memory > 0 {
		if err := govm.ChangeMemory(int64(memory)); err != nil {
			return fmt.Errorf("changing memory: %w", err)
		}
		fmt.Printf("Memory set to %d MB\n", memory)
	}
	return nil
}

func runVmDiskResize(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("VM name is required")
	}
	sizeGB, _ := cmd.Flags().GetInt("size")
	if sizeGB == 0 {
		return fmt.Errorf("--size in GB is required")
	}
	diskIdx, _ := cmd.Flags().GetInt("id")
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, args[0])
	if err != nil {
		return err
	}
	spec := govm.VM.VmSpecSection
	if spec == nil || spec.DiskSection == nil {
		return fmt.Errorf("no disk section found")
	}
	for i, disk := range spec.DiskSection.DiskSettings {
		if i == diskIdx || (diskIdx == 0 && i == 0) {
			oldMB := disk.SizeMb
			newMB := sizeGB * 1024
			disk.SizeMb = int64(newMB)
			if _, err := govm.UpdateInternalDisks(spec); err != nil {
				return fmt.Errorf("resizing disk: %w", err)
			}
			fmt.Printf("Disk %d resized from %dMB to %dMB\n", i, oldMB, newMB)
			return nil
		}
	}
	return fmt.Errorf("disk index %d not found", diskIdx)
}

func init() {
	rootCmd.AddCommand(newVmCmd())
}
