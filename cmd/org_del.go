package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

const (
	delAPIVer = "38.1"
	delPageSz = 25
	delTick   = "\033[32m✓\033[0m"
	delCross  = "\033[31m✗\033[0m"
	delInfo   = "\033[36m→\033[0m"
	delWarn   = "\033[33m!\033[0m"
	delBold   = "\033[1m"
	delReset  = "\033[0m"
	delCyan   = "\033[36m"
	delGreen  = "\033[32m"
	delRed    = "\033[31m"
	delDim    = "\033[2m"
)

func newOrgDelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del <org-name>",
		Short: "Delete organization completely",
		Args:  cobra.ExactArgs(1),
		RunE:  runOrgDel,
	}
	cmd.Flags().Bool("dry-run", false, "Preview only")
	cmd.Flags().Bool("yes", false, "Skip confirmation")
	return cmd
}

func RegisterOrgDelCmd(orgCmd *cobra.Command) {
	orgCmd.AddCommand(newOrgDelCmd())
}

func runOrgDel(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	skipYes, _ := cmd.Flags().GetBool("yes")

	// auth: cloudapi provider session (legacy /api/sessions is disabled on vCD 39.x)
	cloud, err := newCloudClientForOrg(nil)
	if err != nil {
		return err
	}
	cl, err := sdkClientFromToken(cloud)
	if err != nil {
		return fmt.Errorf("%s sdk: %v", delCross, err)
	}

	targetOrg, err := cl.VCDClient.GetAdminOrgByName(orgName)
	if err != nil {
		return fmt.Errorf("%s org %q not found: %v", delCross, orgName, err)
	}

	orgID := targetOrg.AdminOrg.ID
	orgUrn := "urn:vcloud:org:" + xUUID(orgID)

	// ── Header ──────────────────────────────────────────────────────
	hdr := func() {
		fmt.Fprintln(cmd.ErrOrStderr())
		fmt.Fprintf(cmd.ErrOrStderr(), "%s╔══════════════════════════════════════════════╗%s\n", delCyan, delReset)
		fmt.Fprintf(cmd.ErrOrStderr(), "%s║%s  UNICD ORG DELETE                           %s║%s\n", delCyan, delBold, delCyan, delReset)
		fmt.Fprintf(cmd.ErrOrStderr(), "%s╚══════════════════════════════════════════════╝%s\n", delCyan, delReset)
		fmt.Fprintln(cmd.ErrOrStderr())
	}

	step := func(n int, msg string) {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n%s[%d/8]%s %s%s%s\n", delBold, n, delReset, delBold, msg, delReset)
	}
	ok := func(msg string) {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", delTick, msg)
	}
	fail := func(msg string) {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", delCross, msg)
	}
	info := func(msg string) {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", delInfo, msg)
	}
	none := func() {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %s %s\n", delDim, "(nenhum)")
	}

	hdr()
	fmt.Fprintf(cmd.ErrOrStderr(), "  Org:  %s%s%s\n", delBold, orgName, delReset)
	fmt.Fprintf(cmd.ErrOrStderr(), "  ID:   %s%s%s\n", delDim, orgUrn, delReset)

	if dryRun {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  %s%s modo dry-run %s\n", delWarn, delBold, delReset)
		return dryRunDel(cl.VCDClient, cloud, targetOrg, orgUrn, cmd)
	}

	if !skipYes {
		fmt.Fprintf(cmd.ErrOrStderr(), "\n  %s%sTem certeza que deseja deletar TUDO em %q?%s\n", delWarn, delBold, orgName, delReset)
		fmt.Fprintf(cmd.ErrOrStderr(), "  Digite YES para confirmar: ")
		var ans string
		fmt.Scanln(&ans)
		if ans != "YES" {
			fmt.Fprintf(cmd.ErrOrStderr(), "\n  %s Cancelado.%s\n", delDim, delReset)
			return nil
		}
	}

	start := time.Now()

	// 1. Application Port Profiles
	step(1, "Application Port Profiles")
	delAppPort(cloud, orgUrn, ok, fail, info, none)

	// 2. VMs + vApps
	step(2, "VMs e vApps")
	delVMs(cl.VCDClient, targetOrg, ok, fail, info, none)

	// 3. Org Networks (must precede edge gateways)
	step(3, "Redes")
	delNets(cl.VCDClient, targetOrg, ok, fail, info, none)

	// 4. Edge Gateways
	step(4, "Edge Gateways")
	delEdges(cl.VCDClient, cloud, targetOrg, ok, fail, info, none)

	// 5. Catalogs (must precede VDCs)
	step(5, "Catalogs")
	delCatalogs(cl.VCDClient, targetOrg, ok, fail, info, none)

	// 6. VDCs
	step(6, "VDCs")
	delVDCs(cl.VCDClient, targetOrg, ok, fail, info, none)

	// 7. Disable Org (required before deletion)
	step(7, "Desabilitar Org")
	delOrgDisable(cl.VCDClient, targetOrg, ok, fail)

	// 8. Delete Org
	step(8, "Deletar Org")
	delOrgFinal(cl.VCDClient, cloud, targetOrg, orgID, ok, fail)

	elapsed := time.Since(start).Round(time.Second)
	fmt.Fprintf(cmd.ErrOrStderr(), "\n%s════════════════════════════════════════════════%s\n", delCyan, delReset)
	fmt.Fprintf(cmd.ErrOrStderr(), "%s%s  ORG %s DELETADA EM %s%s\n", delCyan, delBold, orgName, elapsed, delReset)
	fmt.Fprintf(cmd.ErrOrStderr(), "%s════════════════════════════════════════════════%s\n\n", delCyan, delReset)

	return nil
}

// ── helpers ────────────────────────────────────────────────────────

type dmsg func(string)

// ── 1. Application Port Profiles ──────────────────────────────────

func delAppPort(cloud *cloudClient, orgUrn string, ok, fail, info dmsg, none func()) {
	profiles := listOrgPortProfiles(cloud, orgUrn)
	if len(profiles) == 0 {
		none()
		return
	}
	info(fmt.Sprintf("encontrados %d profiles", len(profiles)))
	for _, p := range profiles {
		urn, _ := p["id"].(string)
		name, _ := p["name"].(string)
		s, body, err := cloud.DoJSON("DELETE", "/cloudapi/1.0.0/applicationPortProfiles/"+urlEnc(urn), nil)
		if err != nil || s >= 300 {
			fail(fmt.Sprintf("%s (HTTP %d)", name, s))
			_ = body
		} else {
			ok(name)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func listOrgPortProfiles(cloud *cloudClient, orgUrn string) []map[string]interface{} {
	var all []map[string]interface{}
	page := 1
	for {
		p := fmt.Sprintf("/cloudapi/1.0.0/applicationPortProfiles?page=%d&pageSize=%d&filter=%s",
			page, delPageSz, urlQEnc("scope==TENANT"))
		s, data, err := cloud.DoJSON("GET", p, nil)
		if err != nil || s != 200 {
			break
		}
		var r struct {
			Values      []map[string]interface{} `json:"values"`
			ResultTotal int                      `json:"resultTotal"`
		}
		json.Unmarshal(data, &r)
		for _, v := range r.Values {
			ref, _ := v["orgRef"].(map[string]interface{})
			if ref["id"] == orgUrn {
				all = append(all, v)
			}
		}
		if len(r.Values) < delPageSz || len(all) >= r.ResultTotal {
			break
		}
		page++
	}
	return all
}

// ── 2. Edge Gateways ──────────────────────────────────────────────

func delEdges(cl *govcd.VCDClient, cloud *cloudClient, org *govcd.AdminOrg, ok, fail, info dmsg, none func()) {
	vdcs, err := org.GetAllVDCs(false)
	if err != nil {
		fail(fmt.Sprintf("listar VDCs: %v", err))
		return
	}
	found := 0
	for _, vdc := range vdcs {
		records, err := vdc.QueryEdgeGatewayList()
		if err != nil {
			continue
		}
		for _, rec := range records {
			found++
			name := rec.Name
			eid := lastSeg(rec.HREF)
			info(fmt.Sprintf("%s (VDC %s)", name, vdc.Vdc.Name))

			delNAT(cloud, eid)
			delLB(cloud, eid)

			eg, err := vdc.GetEdgeGatewayByName(name, true)
			if err == nil {
				if err := eg.Delete(true, true); err != nil {
					fail(fmt.Sprintf("  edge: %v", err))
				} else {
					ok("  edge deletado")
				}
				continue
			}
			nsxt, err := vdc.GetNsxtEdgeGatewayByName(name)
			if err == nil {
				if err := nsxt.Delete(); err != nil {
					fail(fmt.Sprintf("  nsxt edge: %v", err))
				} else {
					ok("  nsxt edge deletado")
				}
				continue
			}
			fail(fmt.Sprintf("  obter edge: %v", err))
		}
	}
	if found == 0 {
		none()
	}
}

func delNAT(cloud *cloudClient, eid string) {
	base := "/cloudapi/1.0.0/edgeGateways/" + eid + "/nat/rules"
	page := 1
	for {
		s, data, err := cloud.DoJSON("GET", fmt.Sprintf("%s?page=%d&pageSize=100", base, page), nil)
		if err != nil || s != 200 {
			break
		}
		var r struct {
			Values []struct {
				ID string `json:"id"`
			} `json:"values"`
			ResultTotal int `json:"resultTotal"`
		}
		json.Unmarshal(data, &r)
		for _, rule := range r.Values {
			cloud.DoJSON("DELETE", base+"/"+rule.ID, nil)
			time.Sleep(50 * time.Millisecond)
		}
		if len(r.Values) < 100 || page*100 >= r.ResultTotal {
			break
		}
		page++
	}
}

func delLB(cloud *cloudClient, eid string) {
	lbPath := "/cloudapi/1.0.0/edgeGateways/" + eid + "/loadBalancer"
	s, data, err := cloud.DoJSON("GET", lbPath, nil)
	if err != nil || s != 200 {
		return
	}
	var lb map[string]interface{}
	json.Unmarshal(data, &lb)
	if enabled, _ := lb["enabled"].(bool); !enabled {
		return
	}
	cloud.DoJSON("PUT", lbPath, map[string]interface{}{"enabled": false})
	// remove SE groups
	sePath := lbPath + "/serviceEngineGroups"
	s, sedata, _ := cloud.DoJSON("GET", sePath, nil)
	if s == 200 {
		var se struct {
			Values []struct {
				ID string `json:"id"`
			} `json:"values"`
		}
		json.Unmarshal(sedata, &se)
		for _, g := range se.Values {
			cloud.DoJSON("DELETE", sePath+"/"+g.ID, nil)
		}
	}
}

// ── 3. VMs + vApps ───────────────────────────────────────────────

func delVMs(cl *govcd.VCDClient, org *govcd.AdminOrg, ok, fail, info dmsg, none func()) {
	vdcs, err := org.GetAllVDCs(false)
	if err != nil {
		fail(fmt.Sprintf("listar VDCs: %v", err))
		return
	}
	total := 0
	for _, vdc := range vdcs {
		vms, err := vdc.QueryVmList(types.VmQueryFilterAll)
		if err != nil || len(vms) == 0 {
			continue
		}
		total += len(vms)
		// standalone VMs
		for _, vm := range vms {
			if vm.ContainerName == "" || vm.ContainerName == "vApp" {
				govm, err := vdc.QueryVmByName(vm.Name)
				if err != nil {
					continue
				}
				powerOff(govm)
				if err := govm.Delete(); err != nil {
					fail(fmt.Sprintf("VM %s: %v", vm.Name, err))
				} else {
					ok("VM " + vm.Name)
				}
			}
		}
		// vApps
		vappMap := map[string]bool{}
		for _, vm := range vms {
			if vm.ContainerName != "" && vm.ContainerName != "vApp" {
				vappMap[vm.ContainerName] = true
			}
		}
		for vname := range vappMap {
			vapp, err := vdc.GetVAppByName(vname, true)
			if err != nil {
				continue
			}
			if vapp.VApp != nil && vapp.VApp.Children != nil {
				for _, vmr := range vapp.VApp.Children.VM {
					if g, err := vapp.GetVMByName(vmr.Name, true); err == nil {
						powerOff(g)
					}
				}
			}
			if t, err := vapp.Undeploy(); err == nil {
				t.WaitTaskCompletion()
			}
			t, err := vapp.Delete()
			if err == nil {
				err = t.WaitTaskCompletion()
			}
			if err != nil {
				fail(fmt.Sprintf("vApp %s: %v", vname, err))
			} else {
				ok("vApp " + vname)
			}
		}
	}
	if total == 0 {
		none()
	} else {
		info(fmt.Sprintf("total: %d VMs", total))
	}
}

func powerOff(g *govcd.VM) {
	st, err := g.GetStatus()
	if err != nil {
		return
	}
	if st == "POWERED_ON" || st == "SUSPENDED" {
		if t, err := g.PowerOff(); err == nil {
			t.WaitTaskCompletion()
		}
	}
}

// ── 4. Org Networks ──────────────────────────────────────────────

func delNets(cl *govcd.VCDClient, org *govcd.AdminOrg, ok, fail, info dmsg, none func()) {
	vdcs, err := org.GetAllVDCs(false)
	if err != nil {
		fail(fmt.Sprintf("listar VDCs: %v", err))
		return
	}
	found := 0
	for _, vdc := range vdcs {
		nets, err := vdc.GetNetworkList()
		if err != nil {
			continue
		}
		for _, n := range nets {
			if n.LinkType == 0 { // external/direct
				continue
			}
			found++
			net, err := vdc.GetOrgVdcNetworkByName(n.Name, true)
			if err != nil {
				fail(fmt.Sprintf("rede %s: %v", n.Name, err))
				continue
			}
			t, err := net.Delete()
			if err == nil {
				err = t.WaitTaskCompletion()
			}
			if err != nil {
				fail(fmt.Sprintf("rede %s: %v", n.Name, err))
			} else {
				ok("rede " + n.Name)
			}
		}
	}
	if found == 0 {
		none()
	}
}

// ── 5. VDCs ─────────────────────────────────────────────────────

func delVDCs(cl *govcd.VCDClient, org *govcd.AdminOrg, ok, fail, info dmsg, none func()) {
	vdcs, err := org.GetAllVDCs(false)
	if err != nil {
		fail(fmt.Sprintf("listar VDCs: %v", err))
		return
	}
	if len(vdcs) == 0 {
		none()
		return
	}
	info(fmt.Sprintf("encontrados %d VDCs", len(vdcs)))
	for _, vdc := range vdcs {
		name := vdc.Vdc.Name
		v, err := org.GetVDCByName(name, true)
		if err != nil {
			fail(fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := v.DeleteWait(true, true); err != nil {
			fail(fmt.Sprintf("%s: %v", name, err))
		} else {
			ok(name)
		}
	}
}

// ── 6. Catalogs ─────────────────────────────────────────────────

func delCatalogs(cl *govcd.VCDClient, org *govcd.AdminOrg, ok, fail, info dmsg, none func()) {
	recs, err := org.QueryCatalogList()
	if err != nil {
		if strings.Contains(err.Error(), govcd.ErrorEntityNotFound.Error()) {
			none()
			return
		}
		fail(fmt.Sprintf("listar: %v", err))
		return
	}
	// only catalogs owned by this org (skip shared catalogs from others)
	var names []string
	for _, r := range recs {
		if r.OrgName != "" && r.OrgName != org.AdminOrg.Name {
			continue
		}
		names = append(names, r.Name)
	}
	if len(names) == 0 {
		none()
		return
	}
	info(fmt.Sprintf("encontrados %d catalogs", len(names)))
	for _, name := range names {
		cat, err := org.GetCatalogByName(name, true)
		if err != nil {
			fail(fmt.Sprintf("catalog %s: %v", name, err))
			continue
		}
		if err := cat.Delete(true, true); err != nil {
			fail(fmt.Sprintf("catalog %s: %v", name, err))
		} else {
			ok("catalog " + name)
		}
	}
}

// ── 7. Disable Org ──────────────────────────────────────────────

func delOrgDisable(cl *govcd.VCDClient, org *govcd.AdminOrg, ok, fail dmsg) {
	if !org.AdminOrg.IsEnabled {
		fmt.Printf("  %s org ja desabilitada%s\n", delDim, delReset)
		ok(org.AdminOrg.Name)
		return
	}
	if err := org.Disable(); err != nil {
		fail(fmt.Sprintf("disable: %v", err))
	} else {
		ok(org.AdminOrg.Name)
	}
}

// ── 8. Delete Org ───────────────────────────────────────────────

func delOrgFinal(cl *govcd.VCDClient, cloud *cloudClient, org *govcd.AdminOrg, orgID string, ok, fail dmsg) {
	uuid := xUUID(orgID)
	s, body, err := cloud.DoXML("DELETE", "/api/admin/org/"+uuid, nil)
	if err != nil {
		fail(fmt.Sprintf("delete: %v", err))
		return
	}
	if s != 202 && s != 200 {
		fail(fmt.Sprintf("HTTP %d: %s", s, trunc(string(body), 200)))
		return
	}
	tid := extractTaskID(string(body))
	if tid == "" {
		ok(org.AdminOrg.Name)
		return
	}
	// poll
	fmt.Print("  " + delInfo + " aguardando task " + sid8(tid) + " ")
	for i := 0; i < 120; i++ {
		time.Sleep(5 * time.Second)
		st, td, _ := cloud.DoXML("GET", "/api/task/"+tid, nil)
		if st != 200 {
			continue
		}
		s := string(td)
		if strings.Contains(s, `status="success"`) {
			fmt.Printf("%s\n", delTick)
			ok(org.AdminOrg.Name)
			return
		}
		if strings.Contains(s, `status="error"`) || strings.Contains(s, `status="aborted"`) || strings.Contains(s, `status="canceled"`) {
			fmt.Printf("%s\n", delCross)
			fail(fmt.Sprintf("task failed"))
			return
		}
		if i%4 == 0 && i > 0 {
			fmt.Print(".")
		}
	}
	fmt.Printf("%s\n", delWarn)
	fail("task timeout")
}

func sid8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// ── Dry run ──────────────────────────────────────────────────────

func dryRunDel(cl *govcd.VCDClient, cloud *cloudClient, org *govcd.AdminOrg, orgUrn string, cmd *cobra.Command) error {
	w := cmd.ErrOrStderr()

	profiles := listOrgPortProfiles(cloud, orgUrn)
	fmt.Fprintf(w, "\n  %sApp Port Profiles:  %d%s\n", delCyan, len(profiles), delReset)
	for _, p := range profiles {
		n, _ := p["name"].(string)
		fmt.Fprintf(w, "    %s %s\n", delDim, n)
	}

	vdcs, _ := org.GetAllVDCs(false)
	fmt.Fprintf(w, "  %sVDCs:               %d%s\n", delCyan, len(vdcs), delReset)
	for _, vdc := range vdcs {
		vms, _ := vdc.QueryVmList(types.VmQueryFilterAll)
		edges, _ := vdc.QueryEdgeGatewayList()
		nets, _ := vdc.GetNetworkList()
		fmt.Fprintf(w, "    %s %s  VMs:%d  Edges:%d  Nets:%d\n", delDim, vdc.Vdc.Name, len(vms), len(edges), len(nets))
	}

	recs, err := org.QueryCatalogList()
	var catNames []string
	if err == nil {
		for _, r := range recs {
			if r.OrgName != "" && r.OrgName != org.AdminOrg.Name {
				continue
			}
			catNames = append(catNames, r.Name)
		}
	}
	fmt.Fprintf(w, "  %sCatalogs:           %d%s\n", delCyan, len(catNames), delReset)
	for _, n := range catNames {
		fmt.Fprintf(w, "    %s %s\n", delDim, n)
	}

	fmt.Fprintf(w, "\n  %sOrg: %s%s\n\n", delBold, org.AdminOrg.Name, delReset)
	return nil
}

// ── Utility ──────────────────────────────────────────────────────

func xUUID(id string) string {
	if strings.HasPrefix(id, "urn:vcloud:") {
		parts := strings.Split(id, ":")
		return parts[len(parts)-1]
	}
	return id
}

func lastSeg(href string) string {
	parts := strings.Split(strings.TrimSuffix(href, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return href
}

func extractTaskID(body string) string {
	if i := strings.Index(body, "/task/"); i >= 0 {
		j := i + 6
		for j < len(body) && body[j] != '/' && body[j] != '"' {
			j++
		}
		return body[i+6 : j]
	}
	if i := strings.Index(body, `"id":`); i >= 0 {
		s := i + 5
		for s < len(body) && (body[s] == ' ' || body[s] == '"') {
			s++
		}
		e := s
		for e < len(body) && body[e] != '"' && body[e] != ',' {
			e++
		}
		return body[s:e]
	}
	return ""
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func urlEnc(s string) string {
	return strings.NewReplacer(":", "%3A", "/", "%2F").Replace(s)
}

func urlQEnc(s string) string {
	return strings.NewReplacer(" ", "%20", "=", "%3D", "&", "%26").Replace(s)
}
