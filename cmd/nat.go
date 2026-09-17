package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newNatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nat",
		Short: "NAT rule operations (CloudAPI)",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List NAT rules for an edge gateway",
		RunE:  runNatList,
	}
	listCmd.Flags().StringP("edge", "e", "", "Edge gateway name (default: auto-discover)")
	cmd.AddCommand(listCmd)
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a NAT rule",
	}
	dnatCmd := &cobra.Command{
		Use:   "dnat",
		Short: "Create DNAT rule",
		RunE:  runNatCreateDNAT,
	}
	dnatCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	dnatCmd.Flags().StringP("internal-ip", "i", "", "Internal IP")
	dnatCmd.Flags().StringP("external-ip", "x", "", "External IP")
	dnatCmd.Flags().IntP("internal-port", "", 0, "Internal port")
	dnatCmd.Flags().IntP("external-port", "", 0, "External port")
	dnatCmd.Flags().StringP("protocol", "p", "tcp", "Protocol (tcp|udp|icmp|any)")
	dnatCmd.Flags().StringP("name", "n", "", "Rule name (default: auto-generated)")
	createCmd.AddCommand(dnatCmd)

	snatCmd := &cobra.Command{
		Use:   "snat",
		Short: "Create SNAT rule",
		RunE:  runNatCreateSNAT,
	}
	snatCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	snatCmd.Flags().StringP("internal-ip", "i", "", "Internal network/source IP (CIDR)")
	snatCmd.Flags().StringP("external-ip", "x", "", "External/public IP")
	snatCmd.Flags().StringP("name", "n", "", "Rule name (default: auto-generated)")
	createCmd.AddCommand(snatCmd)
	cmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete a NAT rule by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runNatDelete,
	}
	deleteCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	cmd.AddCommand(deleteCmd)

	renameCmd := &cobra.Command{
		Use:   "rename <rule-id> <new-name>",
		Short: "Rename a NAT rule",
		Args:  cobra.ExactArgs(2),
		RunE:  runNatRename,
	}
	renameCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	cmd.AddCommand(renameCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "flush",
		Short: "Remove all NAT rules (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show NAT rule details (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func runNatList(cmd *cobra.Command, args []string) error {
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	edgeName, _ := cmd.Flags().GetString("edge")
	if edgeName == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return err
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways found")
		}
		edgeName = edges[0].Name
		fmt.Fprintf(cmd.ErrOrStderr(), "Using edge: %s\n", edgeName)
	}
	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)
	data, err := cl.RawCloudAPI("GET", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules", nil)
	if err != nil {
		return fmt.Errorf("fetching NAT rules: %w", err)
	}
	var result struct {
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Println(string(data))
		return err
	}
	if len(result.Values) == 0 {
		fmt.Println("No NAT rules")
		return nil
	}
	for _, r := range result.Values {
		id, _ := r["id"].(string)
		name, _ := r["name"].(string)
		rtype, _ := r["ruleType"].(string)
		enabled, _ := r["enabled"].(bool)
		desc, _ := r["description"].(string)
		extAddr, _ := r["externalAddresses"].(string)
		intAddr, _ := r["internalAddresses"].(string)
		extPorts, _ := r["dnatExternalPort"].(string)
		intPorts, _ := r["internalPort"].(string)
		if extAddr == "" {
			extAddr = "-"
		}
		if intAddr == "" {
			intAddr = "-"
		}
		if extPorts == "" {
			extPorts = "-"
		}
		if intPorts == "" {
			intPorts = "-"
		}
		fmt.Printf("%-40s %-8s %-30s %-5t %-18s %-18s %-6s %-6s %s\n", id, rtype, name, enabled, extAddr, intAddr, extPorts, intPorts, desc)
	}
	return nil
}

func runNatCreateDNAT(cmd *cobra.Command, args []string) error {
	edgeName, _ := cmd.Flags().GetString("edge")
	internalIP, _ := cmd.Flags().GetString("internal-ip")
	externalIP, _ := cmd.Flags().GetString("external-ip")
	internalPort, _ := cmd.Flags().GetInt("internal-port")
	externalPort, _ := cmd.Flags().GetInt("external-port")
	protocol, _ := cmd.Flags().GetString("protocol")
	ruleName, _ := cmd.Flags().GetString("name")

	if internalIP == "" || externalIP == "" {
		return fmt.Errorf("--internal-ip and --external-ip are required")
	}

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}

	if edgeName == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return err
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways found")
		}
		edgeName = edges[0].Name
		fmt.Printf("Using edge: %s\n", edgeName)
	}

	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)

	rule := buildDNATRule(externalIP, internalIP, externalPort, internalPort, protocol, ruleName)
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	resp, err := cl.RawCloudAPI("POST", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating DNAT rule: %w", err)
	}
	fmt.Printf("DNAT rule created: %s\n", string(resp))
	return nil
}

func runNatCreateSNAT(cmd *cobra.Command, args []string) error {
	edgeName, _ := cmd.Flags().GetString("edge")
	internalIP, _ := cmd.Flags().GetString("internal-ip")
	externalIP, _ := cmd.Flags().GetString("external-ip")
	ruleName, _ := cmd.Flags().GetString("name")

	if internalIP == "" || externalIP == "" {
		return fmt.Errorf("--internal-ip and --external-ip are required")
	}

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}

	if edgeName == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return err
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways found")
		}
		edgeName = edges[0].Name
		fmt.Printf("Using edge: %s\n", edgeName)
	}

	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)

	name := ruleName
	if name == "" {
		name = fmt.Sprintf("SNAT-%s->%s", internalIP, externalIP)
	}
	rule := natRule{
		Name:                     name,
		Description:              "Created by unicd CLI",
		Enabled:                  true,
		RuleType:                 "SNAT",
		ExternalAddresses:        externalIP,
		InternalAddresses:        internalIP,
		FirewallMatch:            "MATCH_INTERNAL_ADDRESS",
		Priority:                 0,
		Logging:                  false,
		SnatDestinationAddresses: "",
	}
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	resp, err := cl.RawCloudAPI("POST", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating SNAT rule: %w", err)
	}
	fmt.Printf("SNAT rule created: %s\n", string(resp))
	return nil
}

func runNatDelete(cmd *cobra.Command, args []string) error {
	ruleID := args[0]

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}

	edgeName, _ := cmd.Flags().GetString("edge")
	if edgeName == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return err
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways found")
		}
		edgeName = edges[0].Name
		fmt.Fprintf(cmd.ErrOrStderr(), "Using edge: %s\n", edgeName)
	}

	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)

	_, err = cl.RawCloudAPI("DELETE", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules/"+ruleID, nil)
	if err != nil {
		return fmt.Errorf("deleting NAT rule: %w", err)
	}

	fmt.Printf("NAT rule %s deleted\n", ruleID)
	return nil
}

func runNatRename(cmd *cobra.Command, args []string) error {
	ruleID := args[0]
	newName := args[1]

	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}

	edgeName, _ := cmd.Flags().GetString("edge")
	if edgeName == "" {
		edges, err := cl.ListEdgeGateways(ctx.VDC)
		if err != nil {
			return err
		}
		if len(edges) == 0 {
			return fmt.Errorf("no edge gateways found")
		}
		edgeName = edges[0].Name
		fmt.Fprintf(cmd.ErrOrStderr(), "Using edge: %s\n", edgeName)
	}

	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)

	data, err := cl.RawCloudAPI("GET", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules", nil)
	if err != nil {
		return fmt.Errorf("fetching NAT rules: %w", err)
	}
	var result struct {
		Values []map[string]any `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	var found bool
	for _, r := range result.Values {
		id, _ := r["id"].(string)
		if id == ruleID {
			r["name"] = newName
			found = true
			payload, _ := json.Marshal(r)
			_, err := cl.RawCloudAPI("PUT", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/nat/rules/"+ruleID, strings.NewReader(string(payload)))
			if err != nil {
				return fmt.Errorf("renaming rule: %w", err)
			}
			break
		}
	}
	if !found {
		return fmt.Errorf("rule %s not found", ruleID)
	}
	fmt.Printf("NAT rule %s renamed to %q\n", ruleID, newName)
	return nil
}

func extractEdgeID(id string) string {
	return id
}

type natRule struct {
	ID                       string         `json:"id,omitempty"`
	Name                     string         `json:"name"`
	Description              string         `json:"description"`
	Enabled                  bool           `json:"enabled"`
	RuleType                 string         `json:"ruleType"`
	ExternalAddresses        string         `json:"externalAddresses"`
	InternalAddresses        string         `json:"internalAddresses"`
	DnatExternalPort         string         `json:"dnatExternalPort,omitempty"`
	ApplicationPortProfile   map[string]any `json:"applicationPortProfile,omitempty"`
	AppliedTo                map[string]any `json:"appliedTo,omitempty"`
	FirewallMatch            string         `json:"firewallMatch"`
	Priority                 int            `json:"priority"`
	Logging                  bool           `json:"logging"`
	SnatDestinationAddresses string         `json:"snatDestinationAddresses,omitempty"`
}

func buildDNATRule(extIP, intIP string, extPort, intPort int, protocol, name string) natRule {
	if name == "" {
		name = fmt.Sprintf("DNAT-%s->%s", extIP, intIP)
	}
	r := natRule{
		Name:              name,
		Description:       "Created by unicd CLI",
		Enabled:           true,
		RuleType:          "DNAT",
		ExternalAddresses: extIP,
		InternalAddresses: intIP,
		FirewallMatch:     "MATCH_EXTERNAL_ADDRESS",
		Priority:          0,
		Logging:           false,
	}
	if extPort > 0 {
		r.DnatExternalPort = fmt.Sprintf("%d", extPort)
	}

	return r
}

func init() {
	rootCmd.AddCommand(newNatCmd())
}
