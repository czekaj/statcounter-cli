package cmd

import (
	"context"
	"encoding/json"

	"github.com/czekaj/statcounter-cli/internal/statcounter"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP server (stdio) for AI agents",
		Long: `Run statcounter as a Model Context Protocol server over stdio.

It exposes tools (list_projects, page_views, came_from, visitors) that an MCP
client (e.g. Claude) can call to read your StatCounter stats. Authenticate first
with ` + "`statcounter login`" + `, or provide STATCOUNTER_USERNAME / STATCOUNTER_PASSWORD
in the server's environment.

Example MCP client config entry:
  {"command": "statcounter", "args": ["mcp"]}`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCP(cmd.Context())
		},
	}
}

type projectArgs struct {
	Project string `json:"project" jsonschema:"project id (pNNN or NNN) or a name substring; see list_projects"`
	Limit   int    `json:"limit,omitempty" jsonschema:"maximum number of rows to return; 0 or omitted returns all available"`
}

func runMCP(ctx context.Context) error {
	svc, err := statcounter.New()
	if err != nil {
		return err
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "statcounter",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_projects",
		Description: "List the StatCounter projects (websites) in the account, with their ids and names.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		ps, err := svc.Projects()
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(ps)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "page_views",
		Description: "Recent Page View Activity for a project: each individual page load with timestamp, device, OS, browser, country, page and referrer.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, a projectArgs) (*mcp.CallToolResult, any, error) {
		items, err := svc.PageViews(a.Project, a.Limit)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(items)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "came_from",
		Description: "Recent Came From for a project: inbound referrals with timestamp, referring URL, domain and entry page.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, a projectArgs) (*mcp.CallToolResult, any, error) {
		items, err := svc.CameFrom(a.Project, a.Limit)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(items)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "visitors",
		Description: "Recent Visitor Activity (sessions) for a project: timestamp, country, OS, browser, referrer and entry page.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, a projectArgs) (*mcp.CallToolResult, any, error) {
		items, err := svc.Visitors(a.Project, a.Limit)
		if err != nil {
			return errResult(err), nil, nil
		}
		return jsonResult(items)
	})

	return server.Run(ctx, &mcp.StdioTransport{})
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}

func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
