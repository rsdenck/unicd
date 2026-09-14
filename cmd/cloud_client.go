package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/denck/unicd/pkg/client"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

type cloudClient struct {
	host   string
	apiVer string
	token  string
}

var delHTTP = &http.Client{Timeout: 30 * time.Second}

func newCloudClientForOrg(_ interface{}) (*cloudClient, error) {
	host := os.Getenv("UNICD_HOST")
	user := os.Getenv("UNICD_USER")
	pass := os.Getenv("UNICD_PASS")
	if host == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("UNICD_HOST/USER/PASS env vars required")
	}
	apiVer := os.Getenv("UNICD_API_VERSION")
	if apiVer == "" {
		apiVer = delAPIVer
	}

	c := &cloudClient{host: host, apiVer: apiVer}
	b64 := b64enc(user + ":" + pass)

	req, _ := http.NewRequest("POST", "https://"+host+"/cloudapi/1.0.0/sessions/provider", nil)
	req.Header.Set("Accept", "application/json;version="+apiVer)
	req.Header.Set("Authorization", "Basic "+b64)

	resp, err := delHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("login HTTP %d", resp.StatusCode)
	}
	c.token = resp.Header.Get("x-vmware-vcloud-access-token")
	if c.token == "" {
		return nil, fmt.Errorf("no token")
	}
	return c, nil
}

func (c *cloudClient) DoJSON(method, path string, body interface{}) (int, []byte, error) {
	return c.DoRaw(method, path, "application/json", body)
}

func (c *cloudClient) DoXML(method, path string, body interface{}) (int, []byte, error) {
	return c.DoRaw(method, path, "application/*+xml", body)
}

func (c *cloudClient) DoRaw(method, path, accept string, body interface{}) (int, []byte, error) {
	var r io.Reader
	if body != nil {
		d, _ := json.Marshal(body)
		r = bytes.NewReader(d)
	}
	req, _ := http.NewRequest(method, "https://"+c.host+path, r)
	req.Header.Set("Accept", accept+";version="+c.apiVer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := delHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	d, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, d, nil
}

func b64enc(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// sdkClientFromToken builds a go-vcloud-director client from the cloudapi
// bearer token (legacy /api/sessions is disabled on vCD 39.x).
func sdkClientFromToken(cloud *cloudClient) (*client.VCDClient, error) {
	return client.NewClientWithToken(cloud.host, "System", string(govcd.BearerTokenHeader), cloud.token, cloud.apiVer, true)
}
