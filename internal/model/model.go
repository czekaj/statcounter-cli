// Package model holds the normalized data types returned by the StatCounter
// dashboard scraper. These are the structures emitted as JSON by the CLI and
// the MCP server, so their field names are part of the tool's public contract.
package model

// Project is a single StatCounter project (website) in the account.
type Project struct {
	// ID is the StatCounter project id including the leading "p", e.g. "p13239990".
	ID string `json:"id"`
	// Name is the human label shown in the dashboard (often the site title/domain).
	Name string `json:"name"`
}

// PageView is one entry from the "Page View Activity" (pageload) log: a single
// page load by a visitor.
type PageView struct {
	// Timestamp is RFC3339 (UTC) derived from the row's data-ts when parseable.
	Timestamp string `json:"timestamp,omitempty"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
	VisitorID string `json:"visitor_id,omitempty"`

	Device  string `json:"device,omitempty"` // desktop, mobile, tablet
	OS      string `json:"os,omitempty"`
	Browser string `json:"browser,omitempty"`

	Country  string `json:"country,omitempty"`
	Region   string `json:"region,omitempty"`
	Language string `json:"language,omitempty"`

	ISP      string `json:"isp,omitempty"`      // visitor's ISP / network host name
	Page     string `json:"page,omitempty"`     // URL or path of the page loaded
	Referrer string `json:"referrer,omitempty"` // where the visitor came from
}

// CameFrom is one entry from the "Recent Came From" (camefrom_activity) log: an
// inbound referral.
type CameFrom struct {
	Timestamp string `json:"timestamp,omitempty"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
	VisitorID string `json:"visitor_id,omitempty"`

	Referrer  string `json:"referrer"`             // full referring URL
	Domain    string `json:"domain,omitempty"`     // referring domain
	EntryPage string `json:"entry_page,omitempty"` // landing page on your site
}

// Visitor is one entry from the "Visitor Activity" (visitor) log: a visitor
// session. Fields mirror the dashboard's per-visitor detail. The visitor's IP
// address is intentionally omitted.
type Visitor struct {
	Timestamp string `json:"timestamp,omitempty"`
	Date      string `json:"date,omitempty"`
	Time      string `json:"time,omitempty"`
	VisitorID string `json:"visitor_id,omitempty"`

	Country  string `json:"country,omitempty"`
	Location string `json:"location,omitempty"` // city / region / country as shown
	ISP      string `json:"isp,omitempty"`      // visitor's ISP / network host name

	System     string `json:"system,omitempty"`     // OS / browser as displayed
	Resolution string `json:"resolution,omitempty"` // screen resolution

	Referrer  string `json:"referrer,omitempty"`   // referring URL
	EntryPage string `json:"entry_page,omitempty"` // page visited / landing page

	PageViews     int    `json:"page_views,omitempty"`     // page views in this session
	TotalSessions string `json:"total_sessions,omitempty"` // returning-visit count
	ExitTime      string `json:"exit_time,omitempty"`
}
