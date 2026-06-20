package cmd

import (
	"fmt"
	"runtime"

	"github.com/czekaj/statcounter-cli/internal/output"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data := map[string]string{
				"version": version,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			}
			if outFormat == output.FormatJSON {
				return output.JSON(cmd.OutOrStdout(), data)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "statcounter %s (%s/%s, %s)\n",
				version, runtime.GOOS, runtime.GOARCH, runtime.Version())
			return nil
		},
	}
}
