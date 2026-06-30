package cmd

import (
	"fmt"
	"os"

	"github.com/denck/unicd/pkg/client"
	"github.com/denck/unicd/pkg/config"
	"github.com/spf13/cobra"
)

func getClientFromContext() (*client.VCDClient, *config.Context, error) {
	user := os.Getenv("UNICD_USER")
	pass := os.Getenv("UNICD_PASS")
	host := os.Getenv("UNICD_HOST")
	org := os.Getenv("UNICD_ORG")
	apiVer := os.Getenv("UNICD_API_VERSION")
	if apiVer == "" {
		apiVer = "37.0"
	}

	if user != "" && pass != "" && host != "" && org != "" {
		cl, err := client.NewClient(host, user, pass, org, apiVer, true)
		if err != nil {
			return nil, nil, fmt.Errorf("env auth failed: %w", err)
		}
		cl.OrgName = org
		ctx := &config.Context{
			Name:       org,
			Host:       host,
			Org:        org,
			Username:   user,
			APIVersion: apiVer,
		}
		return cl, ctx, nil
	}

	ctx, err := config.GetCurrentContext()
	if err != nil {
		return nil, nil, fmt.Errorf("login first: %w", err)
	}
	session, err := config.LoadSession()
	if err != nil {
		return nil, nil, fmt.Errorf("no session, login first")
	}
	token := session["token"]
	authHeader := session["auth"]
	if authHeader == "" {
		authHeader = "x-vcloud-authorization"
	}
	cl, err := client.NewClientWithToken(ctx.Host, ctx.Org, authHeader, token, ctx.APIVersion, true)
	if err != nil {
		return nil, nil, fmt.Errorf("connecting: %w", err)
	}
	cl.VDCName = ctx.VDC
	return cl, ctx, nil
}

func stubCmd(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not implemented yet")
}
