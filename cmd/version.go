package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "1.3.0"
	BuildDate = "unknown"
	Commit    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show unicd version info",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("unicd version %s\n", Version)
			fmt.Printf("commit:     %s\n", Commit)
			fmt.Printf("built:      %s\n", BuildDate)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newVersionCmd())
}
