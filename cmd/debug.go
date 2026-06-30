package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug tools",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Show debug info",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("OS:      %s\n", runtime.GOOS)
			fmt.Printf("Arch:    %s\n", runtime.GOARCH)
			fmt.Printf("Version: %s\n", Version)
			return nil
		},
	})
	apiCmd := &cobra.Command{
		Use:   "api [method] [path]",
		Short: "Make raw CloudAPI call",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, _ := cmd.Flags().GetString("data")
			dataFile, _ := cmd.Flags().GetString("file")
			cl, ctx, err := getClientFromContext()
			if err != nil {
				return err
			}
			_ = ctx
			method := strings.ToUpper(args[0])
			path := args[1]
			var body io.Reader
			if data != "" {
				body = strings.NewReader(data)
			} else if dataFile != "" {
				b, err := os.ReadFile(dataFile)
				if err != nil {
					return err
				}
				body = strings.NewReader(string(b))
			}
			resp, err := cl.RawCloudAPI(method, path, body)
			if err != nil {
				return err
			}
			fmt.Println(string(resp))
			return nil
		},
	}
	apiCmd.Flags().StringP("data", "d", "", "JSON body data")
	apiCmd.Flags().StringP("file", "f", "", "Read body from file")
	cmd.AddCommand(apiCmd)
	return cmd
}

func init() {
	rootCmd.AddCommand(newDebugCmd())
}
