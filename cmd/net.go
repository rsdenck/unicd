package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func newNetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "net",
		Short: "Network operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List networks",
		RunE:  runNetList,
	})
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a network",
		RunE:  runNetCreate,
	}
	createCmd.Flags().StringP("name", "n", "", "Network name")
	createCmd.Flags().StringP("gateway", "g", "", "Gateway IP")
	createCmd.Flags().StringP("type", "t", "isolated", "Network type (isolated|routed)")
	createCmd.Flags().StringP("dns1", "", "1.1.1.1", "Primary DNS")
	createCmd.Flags().IntP("prefix", "p", 24, "Subnet prefix length")
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a network",
		RunE:  runNetDelete,
	}
	deleteCmd.Flags().StringP("name", "n", "", "Network name")
	cmd.AddCommand(deleteCmd)

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show network details",
		RunE:  runNetShow,
	}
	showCmd.Flags().StringP("name", "n", "", "Network name")
	cmd.AddCommand(showCmd)

	ipamCmd := &cobra.Command{
		Use:   "ipam",
		Short: "Show IPAM details for a network",
		RunE:  runNetIPAM,
	}
	ipamCmd.Flags().StringP("name", "n", "", "Network name")
	cmd.AddCommand(ipamCmd)

	attachCmd := &cobra.Command{
		Use:   "attach",
		Short: "Attach network to VM",
		RunE:  runNetAttach,
	}
	attachCmd.Flags().StringP("name", "n", "", "Network name")
	attachCmd.Flags().StringP("vm", "", "", "VM name")
	attachCmd.Flags().String("ip", "", "IP address (MANUAL mode)")
	attachCmd.Flags().String("mode", "POOL", "IP allocation mode (MANUAL|DHCP|POOL)")
	cmd.AddCommand(attachCmd)

	return cmd
}

func runNetList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	nets, err := vdc.GetNetworkList()
	if err != nil {
		return err
	}
	for _, n := range nets {
		t := "isolated"
		if n.LinkType == 1 {
			t = "routed"
		} else if n.LinkType == 0 {
			t = "direct"
		}
		fmt.Printf("%-20s %-15s %-15s %-8s VDC: %s\n", n.Name, n.DefaultGateway, n.Netmask, t, n.VdcName)
	}
	return nil
}

func runNetCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	gateway, _ := cmd.Flags().GetString("gateway")
	netType, _ := cmd.Flags().GetString("type")
	dns1, _ := cmd.Flags().GetString("dns1")
	prefix, _ := cmd.Flags().GetInt("prefix")

	if name == "" || gateway == "" {
		return fmt.Errorf("--name and --gateway are required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	netmask := netmaskFromPrefix(prefix)

	netConfig := &types.OrgVDCNetwork{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		Name:  name,
		Configuration: &types.NetworkConfiguration{
			FenceMode: netType,
			IPScopes: &types.IPScopes{
				IPScope: []*types.IPScope{{
					Gateway: gateway,
					Netmask: netmask,
					DNS1:    dns1,
					DNS2:    "8.8.8.8",
					IPRanges: &types.IPRanges{
						IPRange: []*types.IPRange{{
							StartAddress: ipStartFromGateway(gateway, prefix),
							EndAddress:   ipEndFromGateway(gateway, prefix),
						}},
					},
				}},
			},
		},
	}
	if netType == "routed" || netType == "natRouted" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return fmt.Errorf("listing edge gateways: %w", err)
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways available in VDC")
		}
		edgeName := edges[0].Name
		eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
		if err != nil {
			return fmt.Errorf("finding edge gateway: %w", err)
		}
		netConfig.EdgeGateway = &types.Reference{
			HREF: eg.EdgeGateway.HREF,
			Name: eg.EdgeGateway.Name,
			ID:   eg.EdgeGateway.ID,
			Type: "application/vnd.vmware.admin.edgeGateway+xml",
		}
	}
	task, err := vdc.CreateOrgVDCNetwork(netConfig)
	if err != nil {
		return fmt.Errorf("creating network: %w", err)
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("Network %q created (%s)\n", name, netType)
	return nil
}

func netmaskFromPrefix(prefix int) string {
	mask := uint32(0xFFFFFFFF << (32 - prefix))
	return fmt.Sprintf("%d.%d.%d.%d", mask>>24, (mask>>16)&0xFF, (mask>>8)&0xFF, mask&0xFF)
}

func ipStartFromGateway(gw string, prefix int) string {
	ip := parseIP(gw)
	if prefix >= 24 {
		ip[3] = 10
	} else {
		ip[3] = 1
	}
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

func ipEndFromGateway(gw string, prefix int) string {
	ip := parseIP(gw)
	ip[3] = 250
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

func parseIP(s string) [4]byte {
	var ip [4]byte
	fmt.Sscanf(s, "%d.%d.%d.%d", &ip[0], &ip[1], &ip[2], &ip[3])
	return ip
}

func runNetDelete(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	net, err := vdc.GetOrgVdcNetworkByName(name, true)
	if err != nil {
		return err
	}
	task, err := net.Delete()
	if err != nil {
		return err
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("Network %q deleted\n", name)
	return nil
}

func runNetShow(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	net, err := vdc.GetOrgVdcNetworkByName(name, true)
	if err != nil {
		return err
	}
	n := net.OrgVDCNetwork
	fmt.Printf("Name:        %s\n", n.Name)
	fmt.Printf("ID:          %s\n", n.ID)
	fmt.Printf("Fence Mode:  %s\n", n.Configuration.FenceMode)
	if n.Configuration.IPScopes != nil && len(n.Configuration.IPScopes.IPScope) > 0 {
		s := n.Configuration.IPScopes.IPScope[0]
		fmt.Printf("Gateway:     %s\n", s.Gateway)
		fmt.Printf("Netmask:     %s\n", s.Netmask)
		fmt.Printf("DNS1:        %s\n", s.DNS1)
		fmt.Printf("DNS2:        %s\n", s.DNS2)
	}
	if n.EdgeGateway != nil {
		fmt.Printf("Edge GW:     %s\n", n.EdgeGateway.Name)
	}
	return nil
}

func runNetIPAM(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	vdc, err := cl.GetVDC(ctx.VDC)
	if err != nil {
		return err
	}
	net, err := vdc.GetOrgVdcNetworkByName(name, true)
	if err != nil {
		return err
	}
	n := net.OrgVDCNetwork
	fmt.Printf("Network:      %s (%s)\n", n.Name, n.ID)
	if n.Configuration == nil || n.Configuration.IPScopes == nil {
		fmt.Println("No IP scopes configured")
		return nil
	}
	for i, scope := range n.Configuration.IPScopes.IPScope {
		fmt.Printf("Scope %d:\n", i+1)
		fmt.Printf("  Gateway:       %s\n", scope.Gateway)
		fmt.Printf("  Netmask:       %s\n", scope.Netmask)
		fmt.Printf("  Prefix:        %s\n", scope.SubnetPrefixLength)
		fmt.Printf("  DNS1:          %s\n", scope.DNS1)
		fmt.Printf("  DNS2:          %s\n", scope.DNS2)
		fmt.Printf("  DNSSuffix:     %s\n", scope.DNSSuffix)
		fmt.Printf("  Enabled:       %t\n", scope.IsEnabled)
		if scope.IPRanges != nil && len(scope.IPRanges.IPRange) > 0 {
			fmt.Println("  IP Ranges:")
			for _, r := range scope.IPRanges.IPRange {
				fmt.Printf("    %s - %s\n", r.StartAddress, r.EndAddress)
			}
		}
		if scope.AllocatedIPAddresses != nil && len(scope.AllocatedIPAddresses.IPAddress) > 0 {
			fmt.Println("  Allocated IPs:")
			for _, ip := range scope.AllocatedIPAddresses.IPAddress {
				fmt.Printf("    %s\n", ip)
			}
		}
	}
	return nil
}

func runNetAttach(cmd *cobra.Command, args []string) error {
	vmName, _ := cmd.Flags().GetString("vm")
	netName, _ := cmd.Flags().GetString("name")
	ip, _ := cmd.Flags().GetString("ip")
	mode, _ := cmd.Flags().GetString("mode")

	if vmName == "" || netName == "" {
		return fmt.Errorf("--vm and --name are required")
	}

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, vmName)
	if err != nil {
		return err
	}
	netSection, err := govm.GetNetworkConnectionSection()
	if err != nil {
		return err
	}
	idx := 0
	for _, n := range netSection.NetworkConnection {
		if n.NetworkConnectionIndex >= idx {
			idx = n.NetworkConnectionIndex + 1
		}
	}
	modeUpper := "POOL"
	switch mode {
	case "MANUAL":
		modeUpper = "MANUAL"
	case "DHCP":
		modeUpper = "DHCP"
	}
	conn := &types.NetworkConnection{
		NetworkConnectionIndex:  idx,
		Network:                 netName,
		IPAddress:               ip,
		IpType:                  "IPV4",
		IsConnected:             true,
		IPAddressAllocationMode: modeUpper,
	}
	netSection.NetworkConnection = append(netSection.NetworkConnection, conn)
	if err := govm.UpdateNetworkConnectionSection(netSection); err != nil {
		return fmt.Errorf("attaching network: %w", err)
	}
	fmt.Printf("Network %q attached to VM %q (NIC %d)\n", netName, vmName, idx)
	return nil
}

func init() {
	rootCmd.AddCommand(newNetCmd())
}
