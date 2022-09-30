package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GreptimeTeam/gtctl/pkg/version"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of gtctl and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", version.Get())
		},
	}
}
