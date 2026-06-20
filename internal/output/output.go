// Package output renders results either as indented JSON (for agents/scripts) or
// as a human-readable table (for interactive terminals).
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// Format is the chosen rendering mode.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

// Resolve decides the output format from an explicit flag value, the
// STATCOUNTER_OUTPUT env var, and a --json shortcut. It defaults to table.
func Resolve(flagValue string, jsonShortcut bool) (Format, error) {
	if jsonShortcut {
		return FormatJSON, nil
	}
	v := flagValue
	if v == "" {
		v = os.Getenv("STATCOUNTER_OUTPUT")
	}
	switch strings.ToLower(v) {
	case "", "table", "text":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid output format %q (want table or json)", flagValue)
	}
}

// JSON writes v as indented JSON followed by a newline.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Table writes a simple aligned table. An empty rows slice prints just a
// "(no results)" note to stderr-like effect on the given writer.
func Table(w io.Writer, headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "(no results)")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
		seps := make([]string, len(headers))
		for i, h := range headers {
			seps[i] = strings.Repeat("-", max(3, len(h)))
		}
		fmt.Fprintln(tw, strings.Join(seps, "\t"))
	}
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}

// Truncate shortens s to n runes, adding an ellipsis when cut. Useful for
// keeping table columns readable.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
