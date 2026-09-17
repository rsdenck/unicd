package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type fwRule struct {
	Name                   string           `json:"name"`
	Description            string           `json:"description"`
	Action                 string           `json:"action"`
	ActionValue            string           `json:"actionValue"`
	IPProtocol             string           `json:"ipProtocol"`
	Direction              string           `json:"direction"`
	Logging                bool             `json:"logging"`
	Enabled                bool             `json:"enabled"`
	SourceFirewallGroups   []map[string]any `json:"sourceFirewallGroups,omitempty"`
	DestFirewallGroups     []map[string]any `json:"destinationFirewallGroups,omitempty"`
	AppPortProfiles        []map[string]any `json:"applicationPortProfiles,omitempty"`
	NetworkContextProfiles []map[string]any `json:"networkContextProfiles,omitempty"`
}

type fwRulesPayload struct {
	UserDefinedRules []map[string]any `json:"userDefinedRules"`
}

func newFwCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fw",
		Short: "Firewall rule operations (CloudAPI)",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List firewall rules",
		RunE:  runFwList,
	}
	listCmd.Flags().StringP("edge", "e", "", "Edge gateway name (default: auto-discover)")
	listCmd.Flags().Bool("json", false, "Output raw JSON")
	cmd.AddCommand(listCmd)
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create firewall rule",
		RunE:  runFwCreate,
	}
	createCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	createCmd.Flags().StringP("name", "n", "", "Rule name")
	createCmd.Flags().StringP("action", "a", "ALLOW", "Action (ALLOW|DROP)")
	createCmd.Flags().StringP("source", "s", "any", "Source IP/CIDR")
	createCmd.Flags().StringP("destination", "d", "any", "Destination IP/CIDR")
	createCmd.Flags().StringP("direction", "", "IN_OUT", "Direction (IN|OUT|IN_OUT)")
	createCmd.Flags().StringP("protocol", "p", "any", "Protocol (tcp|udp|icmp|any)")
	createCmd.Flags().IntP("port", "", 0, "Port number")
	cmd.AddCommand(createCmd)
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a firewall rule by name",
		RunE:  runFwDelete,
	}
	deleteCmd.Flags().StringP("edge", "e", "", "Edge gateway name")
	deleteCmd.Flags().StringP("name", "n", "", "Rule name to delete")
	cmd.AddCommand(deleteCmd)
	cmd.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable firewall (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable firewall (not implemented)",
		RunE:  stubCmd,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "policy",
		Short: "Set firewall default policy (not implemented)",
		RunE:  stubCmd,
	})
	return cmd
}

func runFwList(cmd *cobra.Command, args []string) error {
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
	data, err := cl.RawCloudAPI("GET", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/firewall/rules", nil)
	if err != nil {
		return fmt.Errorf("fetching FW rules: %w", err)
	}
	var result struct {
		UserDefinedRules []map[string]interface{} `json:"userDefinedRules"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Println(string(data))
		return err
	}
	if len(result.UserDefinedRules) == 0 {
		fmt.Println("No firewall rules")
		return nil
	}
	showJSON, _ := cmd.Flags().GetBool("json")
	if showJSON {
		fmt.Println(string(data))
		return nil
	}
	for _, r := range result.UserDefinedRules {
		name, _ := r["name"].(string)
		action, _ := r["action"].(string)
		enabled, _ := r["enabled"].(bool)
		direction, _ := r["direction"].(string)
		ipProto, _ := r["ipProtocol"].(string)
		if ipProto == "" {
			ipProto = "any"
		}
		fmt.Printf("%-30s %-6s %-5t %-8s %s\n", name, action, enabled, direction, ipProto)
	}
	return nil
}

func runFwCreate(cmd *cobra.Command, args []string) error {
	edgeName, _ := cmd.Flags().GetString("edge")
	name, _ := cmd.Flags().GetString("name")
	action, _ := cmd.Flags().GetString("action")
	direction, _ := cmd.Flags().GetString("direction")

	if name == "" {
		return fmt.Errorf("--name is required")
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
			return fmt.Errorf("no edge gateways")
		}
		edgeName = edges[0].Name
	}
	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)

	// GET existing user-defined rules
	existingData, err := cl.RawCloudAPI("GET", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/firewall/rules", nil)
	if err != nil {
		return fmt.Errorf("fetching existing FW rules: %w", err)
	}
	var existing struct {
		UserDefinedRules []map[string]any `json:"userDefinedRules"`
	}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("parsing existing rules: %w", err)
	}

	source, _ := cmd.Flags().GetString("source")
	dest, _ := cmd.Flags().GetString("destination")
	protocol, _ := cmd.Flags().GetString("protocol")
	port, _ := cmd.Flags().GetInt("port")

	ipProto := "IPV4_IPV6"
	if protocol == "tcp" || protocol == "udp" || protocol == "icmp" {
		ipProto = protocol
	}

	newRule := map[string]any{
		"name":        name,
		"description": "Created by unicd",
		"action":      action,
		"actionValue": action,
		"ipProtocol":  ipProto,
		"direction":   direction,
		"logging":     false,
		"enabled":     true,
	}
	if source != "" && source != "any" {
		newRule["sourceFirewallGroups"] = []map[string]any{
			{"type": "IPV4_ADDRESS", "name": source, "ipAddresses": []string{source}},
		}
	}
	if dest != "" && dest != "any" {
		newRule["destinationFirewallGroups"] = []map[string]any{
			{"type": "IPV4_ADDRESS", "name": dest, "ipAddresses": []string{dest}},
		}
	}
	if port > 0 {
		proto := "TCP"
		if protocol == "udp" {
			proto = "UDP"
		}
		newRule["applicationPortProfiles"] = []map[string]any{
			{"name": fmt.Sprintf("%s-port-%d", proto, port), "applicationProtocol": proto, "ports": []string{fmt.Sprintf("%d", port)}},
		}
	}

	allRules := existing.UserDefinedRules
	newJSON, _ := json.Marshal(newRule)
	var newAsMap map[string]any
	json.Unmarshal(newJSON, &newAsMap)
	allRules = append(allRules, newAsMap)

	payload := fwRulesPayload{
		UserDefinedRules: allRules,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := cl.RawCloudAPI("PUT", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/firewall/rules", strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating FW rule: %w", err)
	}
	fmt.Printf("Firewall rule created: %s\n", string(resp))
	return nil
}

func runFwDelete(cmd *cobra.Command, args []string) error {
	edgeName, _ := cmd.Flags().GetString("edge")
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("--name is required")
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
			return fmt.Errorf("no edge gateways")
		}
		edgeName = edges[0].Name
	}
	eg, err := cl.GetEdgeByName(ctx.VDC, edgeName)
	if err != nil {
		return err
	}
	edgeID := extractEdgeID(eg.EdgeGateway.ID)
	data, err := cl.RawCloudAPI("GET", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/firewall/rules", nil)
	if err != nil {
		return fmt.Errorf("fetching FW rules: %w", err)
	}
	var result struct {
		UserDefinedRules []map[string]any `json:"userDefinedRules"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	var filtered []map[string]any
	found := false
	for _, r := range result.UserDefinedRules {
		rname, _ := r["name"].(string)
		if rname == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if !found {
		return fmt.Errorf("rule %q not found", name)
	}
	payload := fwRulesPayload{UserDefinedRules: filtered}
	payloadJSON, _ := json.Marshal(payload)
	_, err = cl.RawCloudAPI("PUT", "/cloudapi/1.0.0/edgeGateways/"+edgeID+"/firewall/rules", strings.NewReader(string(payloadJSON)))
	if err != nil {
		return fmt.Errorf("deleting FW rule: %w", err)
	}
	fmt.Printf("Firewall rule %q deleted\n", name)
	return nil
}

func init() {
	rootCmd.AddCommand(newFwCmd())
}
