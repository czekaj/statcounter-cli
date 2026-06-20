// Package cmd implements the statcounter CLI.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/czekaj/statcounter-cli/internal/client"
	"github.com/czekaj/statcounter-cli/internal/output"
	"github.com/czekaj/statcounter-cli/internal/statcounter"
	"github.com/spf13/cobra"
)

var (
	flagOutput string
	flagJSON   bool
	outFormat  output.Format
)

// version is set by Execute.
var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "statcounter",
		Short: "Command-line access to your StatCounter web stats",
		Long: `statcounter is an agent-friendly CLI for StatCounter (statcounter.com).

It works with the FREE plan by signing in to the web dashboard with your
credentials (the official StatCounter API is paid-only). Authenticate once with
` + "`statcounter login`" + `, or set STATCOUNTER_USERNAME and STATCOUNTER_PASSWORD
for non-interactive/automated use.

Every command supports --json for machine-readable output, and ` + "`statcounter mcp`" + `
runs an MCP server so AI agents can query your stats directly.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			f, err := output.Resolve(flagOutput, flagJSON)
			if err != nil {
				return err
			}
			outFormat = f
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "", "output format: table or json (env STATCOUNTER_OUTPUT)")
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "shortcut for --output json")

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newWhoamiCmd(),
		newProjectsCmd(),
		newPageviewsCmd(),
		newCamefromCmd(),
		newVisitorsCmd(),
		newMCPCmd(),
		newVersionCmd(),
	)
	return root
}

// Execute runs the CLI and returns a process exit code.
func Execute(v string) int {
	version = v
	client.Version = v
	err := newRootCmd().Execute()
	if err == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	switch {
	case errors.Is(err, client.ErrNotAuthenticated), errors.Is(err, client.ErrLoginFailed):
		return 3
	default:
		return 1
	}
}

func newService() (*statcounter.Service, error) {
	return statcounter.New()
}

// emitJSON writes v as JSON; emitTable writes the table form. render picks based
// on the resolved output format.
func render(w io.Writer, data any, headers []string, rows [][]string) error {
	if outFormat == output.FormatJSON {
		return output.JSON(w, data)
	}
	output.Table(w, headers, rows)
	return nil
}
