package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

type VCDClient struct {
	*govcd.VCDClient
	OrgName string
	VDCName string
}

func NewClient(host, user, pass, org, apiVer string, insecure bool) (*VCDClient, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s/api", host))
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	client := govcd.NewVCDClient(*u, insecure)
	if apiVer == "" {
		apiVer = "37.0"
	}

	if err := client.Authenticate(user, pass, org); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return &VCDClient{
		VCDClient: client,
		OrgName:   org,
	}, nil
}

func NewClientWithToken(host, org, authHeader, token, apiVer string, insecure bool) (*VCDClient, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s/api", host))
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	client := govcd.NewVCDClient(*u, insecure)
	if apiVer == "" {
		apiVer = "37.0"
	}

	if err := client.SetToken(org, authHeader, token); err != nil {
		return nil, fmt.Errorf("token auth failed: %w", err)
	}

	return &VCDClient{
		VCDClient: client,
		OrgName:   org,
	}, nil
}

func (c *VCDClient) GetOrg() (*govcd.Org, error) {
	return c.VCDClient.GetOrgByNameOrId(c.OrgName)
}

func (c *VCDClient) GetVDC(name string) (*govcd.Vdc, error) {
	if name == "" {
		name = c.VDCName
	}
	org, err := c.GetOrg()
	if err != nil {
		return nil, err
	}
	return org.GetVDCByName(name, true)
}

func (c *VCDClient) ListEdgeGateways(vdcName string) ([]*types.QueryResultEdgeGatewayRecordType, error) {
	vdc, err := c.GetVDC(vdcName)
	if err != nil {
		return nil, err
	}
	return vdc.QueryEdgeGatewayList()
}

func (c *VCDClient) GetEdgeByName(vdcName, name string) (*govcd.EdgeGateway, error) {
	vdc, err := c.GetVDC(vdcName)
	if err != nil {
		return nil, err
	}
	return vdc.GetEdgeGatewayByName(name, true)
}

func (c *VCDClient) RawCloudAPI(method, path string, body io.Reader) ([]byte, error) {
	u := fmt.Sprintf("https://%s%s", c.VCDClient.Client.VCDHREF.Host, path)
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json;version=37.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	cl := c.VCDClient.Client
	if cl.VCDAuthHeader != "" && cl.VCDToken != "" {
		req.Header.Set(cl.VCDAuthHeader, cl.VCDToken)
	}
	if len(cl.VCDToken) > 32 {
		req.Header.Set("Authorization", "bearer "+cl.VCDToken)
	}
	resp, err := cl.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
