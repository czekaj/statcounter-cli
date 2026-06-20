// Command statcounter is an agent-friendly CLI and MCP server for StatCounter.
package main

import (
	"os"

	"github.com/czekaj/statcounter-cli/cmd"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cmd.Execute(version))
}
