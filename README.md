# statcounter

An open-source, **agent-friendly** command-line tool (and MCP server) for
[StatCounter](https://statcounter.com) web analytics.

It works with the **free plan**. StatCounter's official API is paid-only, so
this tool signs in to the web dashboard with your credentials and reads the same
data the dashboard shows you — exposed as clean JSON for scripts and AI agents.

```console
$ statcounter pageviews --project mintberry.ai --limit 3 --json
[
  {
    "timestamp": "2026-06-19T17:42:10Z",
    "visitor_id": "…",
    "device": "desktop",
    "os": "OS X",
    "browser": "Chrome",
    "country": "United States",
    "host": "mintberry.ai",
    "page": "/pricing/",
    "referrer": "https://www.google.com/"
  }
]
```

> **Unofficial.** Not affiliated with or endorsed by StatCounter. It reads your
> own data from your own account. Because it parses the dashboard's HTML, a site
> redesign can break it — please open an issue if something stops working.

## Why not the official API?

StatCounter's API (`api.statcounter.com`) requires a **paid upgrade**:

> *"the Statcounter API is only available to Statcounter members with a paid
> upgrade … we are not currently in a position to offer the Statcounter API on a
> free basis."*

So on the free plan the only way to get your data programmatically is the web
dashboard. This tool automates that: log in once, then query from the CLI or via
MCP. (If you do have a paid plan and prefer the signed API, that's a great future
backend — contributions welcome.)

## Install

### Homebrew (macOS)

```sh
brew install czekaj/tap/statcounter
```

### go install

```sh
go install github.com/czekaj/statcounter-cli@latest
```

### From source

```sh
git clone https://github.com/czekaj/statcounter-cli
cd statcounter-cli
go build -o statcounter .
```

## Authenticate

```sh
statcounter login          # prompts for username + password (password is never stored)
```

Only the resulting **session cookies** are saved, to
`~/.config/statcounter/session.json` (mode `0600`). Use `statcounter logout` to
remove them, and `statcounter whoami` to check status.

For non-interactive / automated use (CI, agents), set credentials in the
environment instead — the tool logs in transparently and caches the session:

```sh
export STATCOUNTER_USERNAME="you@example.com"
export STATCOUNTER_PASSWORD="…"
statcounter projects --json
```

## Usage

```
statcounter projects                       List the projects in your account
statcounter pageviews  -p <project>        Page View Activity (each page load)
statcounter camefrom   -p <project>        Recent Came From (inbound referrals)
statcounter visitors   -p <project>        Visitor Activity (sessions)
statcounter mcp                            Run as an MCP server (stdio)
statcounter whoami | login | logout | version
```

Common flags:

| Flag | Description |
|------|-------------|
| `-p, --project` | Project id (`p13239990` / `13239990`) **or** a name substring (`mintberry`). Env: `STATCOUNTER_PROJECT`. |
| `-n, --limit`   | Max rows to return (`0` = all available, ~last 500 on the free plan). |
| `-o, --output`  | `table` (default) or `json`. Env: `STATCOUNTER_OUTPUT`. |
| `--json`        | Shortcut for `--output json`. |

Examples:

```sh
statcounter projects
statcounter pageviews -p mintberry.ai -n 20
statcounter camefrom  -p p13239990 --json | jq '.[].referrer'
statcounter visitors  -p mintberry --json
```

## Use as an MCP server (AI agents)

`statcounter mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io)
server over stdio, exposing these tools:

- `list_projects` — projects in the account
- `page_views` — Page View Activity for a project
- `came_from` — Recent Came From for a project
- `visitors` — Visitor Activity for a project

`page_views`, `came_from`, and `visitors` take `{ "project": "...", "limit": N }`.

Add it to an MCP client. For **Claude Code**:

```sh
claude mcp add statcounter -- statcounter mcp
```

Or a generic client config (authenticate first with `statcounter login`, or put
credentials in `env`):

```json
{
  "mcpServers": {
    "statcounter": {
      "command": "statcounter",
      "args": ["mcp"],
      "env": {
        "STATCOUNTER_USERNAME": "you@example.com",
        "STATCOUNTER_PASSWORD": "…"
      }
    }
  }
}
```

## Agent / scripting notes

- **JSON everywhere:** `--json` (or `STATCOUNTER_OUTPUT=json`) on every command;
  the MCP server always returns JSON. Field names are stable (see
  [`internal/model`](internal/model/model.go)).
- **Exit codes:** `0` success · `1` error · `2` usage error · `3` not
  authenticated. Data goes to stdout; errors and prompts go to stderr.
- **Project references** accept ids or fuzzy names, so an agent can use whatever
  the user typed.

## Limitations

- Free-plan logs hold roughly the **last 500 events**; there is no historical
  date-range querying (the dashboard doesn't expose it without a paid log size).
- Summary/aggregate reports (popular pages, totals, charts) aren't wrapped yet —
  the focus is the activity logs (page views, came-from, visitors).
- Visitor IP addresses are intentionally omitted from output.

## Development

```sh
go test ./...      # unit tests for the HTML parsers
go vet ./...
go build -o statcounter .
```

The dashboard endpoints and DOM structures this tool depends on are documented in
[`docs/RECON.md`](docs/RECON.md).

## License

MIT — see [LICENSE](LICENSE).
