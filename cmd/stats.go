package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/czekaj/statcounter-cli/internal/output"
	"github.com/spf13/cobra"
)

// projectFlags adds the shared --project and --limit flags to a command.
func projectFlags(cmd *cobra.Command, project *string, limit *int) {
	cmd.Flags().StringVarP(project, "project", "p", os.Getenv("STATCOUNTER_PROJECT"),
		"project id (pNNN / NNN) or name substring (env STATCOUNTER_PROJECT)")
	cmd.Flags().IntVarP(limit, "limit", "n", 0, "max rows to return (0 = all available)")
}

// when picks the best timestamp representation for a table cell.
func when(ts, date, tm string) string {
	if ts != "" {
		return ts
	}
	return strings.TrimSpace(date + " " + tm)
}

func newProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "projects",
		Short: "List the projects (websites) in your account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			projects, err := svc.Projects()
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(projects))
			for _, p := range projects {
				rows = append(rows, []string{p.ID, p.Name})
			}
			return render(cmd.OutOrStdout(), projects, []string{"ID", "NAME"}, rows)
		},
	}
}

func newPageviewsCmd() *cobra.Command {
	var project string
	var limit int
	cmd := &cobra.Command{
		Use:     "pageviews",
		Aliases: []string{"pageloads", "pages-activity"},
		Short:   "Recent Page View Activity (each page load)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			items, err := svc.PageViews(project, limit)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, v := range items {
				rows = append(rows, []string{
					when(v.Timestamp, v.Date, v.Time),
					v.Device, v.OS, v.Browser, v.Country,
					output.Truncate(v.ISP, 20),
					output.Truncate(v.Page, 32),
					output.Truncate(v.Referrer, 32),
				})
			}
			return render(cmd.OutOrStdout(), items,
				[]string{"WHEN", "DEVICE", "OS", "BROWSER", "COUNTRY", "ISP", "PAGE", "REFERRER"}, rows)
		},
	}
	projectFlags(cmd, &project, &limit)
	return cmd
}

func newCamefromCmd() *cobra.Command {
	var project string
	var limit int
	cmd := &cobra.Command{
		Use:     "camefrom",
		Aliases: []string{"came-from", "referrers"},
		Short:   "Recent Came From (inbound referrals)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			items, err := svc.CameFrom(project, limit)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, v := range items {
				rows = append(rows, []string{
					when(v.Timestamp, v.Date, v.Time),
					output.Truncate(v.Referrer, 50),
					v.Domain,
					output.Truncate(v.EntryPage, 40),
				})
			}
			return render(cmd.OutOrStdout(), items,
				[]string{"WHEN", "REFERRER", "DOMAIN", "ENTRY PAGE"}, rows)
		},
	}
	projectFlags(cmd, &project, &limit)
	return cmd
}

func newVisitorsCmd() *cobra.Command {
	var project string
	var limit int
	cmd := &cobra.Command{
		Use:     "visitors",
		Aliases: []string{"visitor-activity"},
		Short:   "Recent Visitor Activity (sessions)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			items, err := svc.Visitors(project, limit)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, v := range items {
				rows = append(rows, []string{
					when(v.Timestamp, v.Date, v.Time),
					v.Country,
					output.Truncate(v.ISP, 20),
					output.Truncate(v.System, 20),
					strconv.Itoa(v.PageViews),
					output.Truncate(v.Referrer, 30),
					output.Truncate(v.EntryPage, 30),
				})
			}
			return render(cmd.OutOrStdout(), items,
				[]string{"WHEN", "COUNTRY", "ISP", "SYSTEM", "PV", "REFERRER", "ENTRY PAGE"}, rows)
		},
	}
	projectFlags(cmd, &project, &limit)
	return cmd
}
