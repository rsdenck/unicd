package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newEdgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "Edge gateway operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List edge gateways",
		RunE:  runEdgeList,
	})
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show edge gateway details",
		RunE:  runEdgeShow,
	}
	showCmd.Flags().StringP("name", "n", "", "Edge gateway name")
	cmd.AddCommand(showCmd)

	interfacesCmd := &cobra.Command{
		Use:   "interfaces",
		Short: "List edge gateway interfaces",
		RunE:  runEdgeInterfaces,
	}
	interfacesCmd.Flags().StringP("name", "n", "", "Edge gateway name")
	cmd.AddCommand(interfacesCmd)

	return cmd
}

func runEdgeList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	edges, err := cl.ListEdgeGateways(ctx.VDC)
	if err != nil {
		return err
	}
	for _, e := range edges {
		fmt.Printf("%-30s | VDC: %-20s | Status: %s\n", e.Name, e.Vdc, e.GatewayStatus)
	}
	return nil
}

func runEdgeShow(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	eg, err := cl.GetEdgeByName(ctx.VDC, name)
	if err != nil {
		return err
	}
	e := eg.EdgeGateway
	fmt.Printf("Name:            %s\n", e.Name)
	fmt.Printf("ID:              %s\n", e.ID)
	fmt.Printf("Description:     %s\n", e.Description)
	fmt.Printf("Status:          %d\n", e.Status)
	if e.Configuration != nil {
		fmt.Printf("Backing Config:  %s\n", e.Configuration.GatewayBackingConfig)
		fmt.Printf("Backward Compat: %t\n", e.Configuration.BackwardCompatibilityMode)
		if e.Configuration.HaEnabled != nil {
			fmt.Printf("HA Enabled:      %t\n", *e.Configuration.HaEnabled)
		}
	}
	return nil
}

func runEdgeInterfaces(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
	}
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	eg, err := cl.GetEdgeByName(ctx.VDC, name)
	if err != nil {
		return err
	}
	e := eg.EdgeGateway
	fmt.Printf("Edge Gateway: %s\n", e.Name)
	fmt.Printf("ID:           %s\n\n", e.ID)
	if e.Configuration == nil {
		fmt.Println("No configuration available")
		return nil
	}
	if e.Configuration.GatewayInterfaces != nil {
		for _, iface := range e.Configuration.GatewayInterfaces.GatewayInterface {
			if iface.InterfaceType == "uplink" {
				fmt.Printf("Interface:    %s (uplink)\n", iface.Name)
				if iface.SubnetParticipation != nil {
					for _, sp := range iface.SubnetParticipation {
						fmt.Printf("  Subnet:     %s/%s\n", sp.Gateway, sp.Netmask)
						if sp.IPAddress != "" {
							fmt.Printf("  IP:         %s\n", sp.IPAddress)
						}
					}
				}
				fmt.Println()
			}
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newEdgeCmd())
}
