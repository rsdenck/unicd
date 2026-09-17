package cmd

import (
	"fmt"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func newIpsecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ipsec",
		Short: "IPSec VPN tunnel operations",
	}

	cmd.PersistentFlags().StringP("edge", "e", "", "Edge gateway name (default: auto-discover)")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List IPSec VPN tunnels",
		RunE:  runIpsecList,
	})

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show IPSec VPN tunnel details",
		Args:  cobra.ExactArgs(1),
		RunE:  runIpsecShow,
	}
	cmd.AddCommand(showCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create IPSec VPN tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runIpsecCreate,
	}
	createCmd.Flags().String("local-ip", "", "Local endpoint IP (sub-allocated on edge)")
	createCmd.Flags().String("local-net", "", "Local networks CIDR (comma-separated)")
	createCmd.Flags().String("peer-ip", "", "Remote endpoint IP")
	createCmd.Flags().String("peer-net", "", "Remote networks CIDR (comma-separated)")
	createCmd.Flags().String("psk", "", "Pre-shared key")
	createCmd.Flags().String("description", "", "Tunnel description")
	createCmd.Flags().Bool("enabled", true, "Enable tunnel")
	createCmd.MarkFlagRequired("local-ip")
	createCmd.MarkFlagRequired("local-net")
	createCmd.MarkFlagRequired("peer-ip")
	createCmd.MarkFlagRequired("peer-net")
	createCmd.MarkFlagRequired("psk")
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete IPSec VPN tunnel",
		Args:  cobra.ExactArgs(1),
		RunE:  runIpsecDelete,
	}
	cmd.AddCommand(deleteCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show IPSec VPN tunnel status",
		Args:  cobra.ExactArgs(1),
		RunE:  runIpsecStatus,
	}
	cmd.AddCommand(statusCmd)

	return cmd
}

func getEdgeGateway(cl *client.VCDClient, ctx *config.Context, cmd *cobra.Command) (*govcd.NsxtEdgeGateway, error) {
	name, _ := cmd.Flags().GetString("edge")
	if name == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return nil, fmt.Errorf("listing edge gateways: %w", err)
		}
		if len(edges) == 0 {
			return nil, fmt.Errorf("no edge gateways found in VDC")
		}
		name = edges[0].Name
	}
	return cl.GetNsxtEdgeGatewayByName(ctx.VDC, name)
}

func parseIpsecTunnel(t *types.NsxtIpSecVpnTunnel) {
	fmt.Printf("Name:            %s\n", t.Name)
	fmt.Printf("ID:              %s\n", t.ID)
	if t.Description != "" {
		fmt.Printf("Description:     %s\n", t.Description)
	}
	fmt.Printf("Enabled:         %t\n", t.Enabled)
	fmt.Printf("Local IP:        %s\n", t.LocalEndpoint.LocalAddress)
	for i, n := range t.LocalEndpoint.LocalNetworks {
		fmt.Printf("Local Net[%d]:     %s\n", i, n)
	}
	fmt.Printf("Remote IP:       %s\n", t.RemoteEndpoint.RemoteAddress)
	for i, n := range t.RemoteEndpoint.RemoteNetworks {
		fmt.Printf("Remote Net[%d]:    %s\n", i, n)
	}
	fmt.Printf("PSK:             %s\n", t.PreSharedKey)
	if t.SecurityType != "" {
		fmt.Printf("Security Type:   %s\n", t.SecurityType)
	}
	if t.AuthenticationMode != "" {
		fmt.Printf("Auth Mode:       %s\n", t.AuthenticationMode)
	}
}

func runIpsecList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	egw, err := getEdgeGateway(cl, ctx, cmd)
	if err != nil {
		return err
	}
	tunnels, err := egw.GetAllIpSecVpnTunnels(nil)
	if err != nil {
		return fmt.Errorf("listing IPSec tunnels: %w", err)
	}
	if len(tunnels) == 0 {
		fmt.Println("No IPSec VPN tunnels found")
		return nil
	}
	for _, t := range tunnels {
		fmt.Printf("%-24s local=%s -> remote=%s\n", t.NsxtIpSecVpn.Name, t.NsxtIpSecVpn.LocalEndpoint.LocalAddress, t.NsxtIpSecVpn.RemoteEndpoint.RemoteAddress)
	}
	return nil
}

func runIpsecShow(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	egw, err := getEdgeGateway(cl, ctx, cmd)
	if err != nil {
		return err
	}
	tunnel, err := egw.GetIpSecVpnTunnelByName(args[0])
	if err != nil {
		return fmt.Errorf("finding IPSec tunnel %q: %w", args[0], err)
	}
	parseIpsecTunnel(tunnel.NsxtIpSecVpn)
	return nil
}

func runIpsecCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	localIP, _ := cmd.Flags().GetString("local-ip")
	localNet, _ := cmd.Flags().GetString("local-net")
	peerIP, _ := cmd.Flags().GetString("peer-ip")
	peerNet, _ := cmd.Flags().GetString("peer-net")
	psk, _ := cmd.Flags().GetString("psk")
	description, _ := cmd.Flags().GetString("description")
	enabled, _ := cmd.Flags().GetBool("enabled")

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	egw, err := getEdgeGateway(cl, ctx, cmd)
	if err != nil {
		return err
	}

	config := &types.NsxtIpSecVpnTunnel{
		Name:         name,
		Description:  description,
		Enabled:      enabled,
		PreSharedKey: psk,
		LocalEndpoint: types.NsxtIpSecVpnTunnelLocalEndpoint{
			LocalAddress:  localIP,
			LocalNetworks: splitCSV(localNet),
		},
		RemoteEndpoint: types.NsxtIpSecVpnTunnelRemoteEndpoint{
			RemoteAddress:  peerIP,
			RemoteNetworks: splitCSV(peerNet),
		},
	}
	_, err = egw.CreateIpSecVpnTunnel(config)
	if err != nil {
		return fmt.Errorf("creating IPSec tunnel: %w", err)
	}
	fmt.Printf("IPSec tunnel %q created\n", name)
	return nil
}

func runIpsecDelete(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	egw, err := getEdgeGateway(cl, ctx, cmd)
	if err != nil {
		return err
	}
	tunnel, err := egw.GetIpSecVpnTunnelByName(args[0])
	if err != nil {
		return fmt.Errorf("finding IPSec tunnel %q: %w", args[0], err)
	}
	if err := tunnel.Delete(); err != nil {
		return fmt.Errorf("deleting IPSec tunnel: %w", err)
	}
	fmt.Printf("IPSec tunnel %q deleted\n", args[0])
	return nil
}

func runIpsecStatus(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	egw, err := getEdgeGateway(cl, ctx, cmd)
	if err != nil {
		return err
	}
	tunnel, err := egw.GetIpSecVpnTunnelByName(args[0])
	if err != nil {
		return fmt.Errorf("finding IPSec tunnel %q: %w", args[0], err)
	}
	status, err := tunnel.GetStatus()
	if err != nil {
		return fmt.Errorf("getting tunnel status: %w", err)
	}
	fmt.Printf("Tunnel:        %s\n", args[0])
	fmt.Printf("Status:        %s\n", status.TunnelStatus)
	fmt.Printf("IKE Status:    %s\n", status.IkeStatus.IkeServiceStatus)
	if status.IkeStatus.FailReason != "" {
		fmt.Printf("Fail Reason:   %s\n", status.IkeStatus.FailReason)
	}
	return nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}

func init() {
	rootCmd.AddCommand(newIpsecCmd())
}
