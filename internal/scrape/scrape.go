// Package scrape turns StatCounter dashboard HTML into normalized model types.
// Selectors here mirror the DOM documented in docs/RECON.md.
package scrape

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/czekaj/statcounter-cli/internal/model"
)

var projectHref = regexp.MustCompile(`^/p(\d+)/summary/$`)
var wsRun = regexp.MustCompile(`\s+`)

// text returns trimmed, whitespace-collapsed text of a selection.
func text(s *goquery.Selection) string {
	return strings.TrimSpace(wsRun.ReplaceAllString(s.Text(), " "))
}

// classToken returns the first class on the selection that is in want.
func classToken(s *goquery.Selection, want ...string) string {
	classes := strings.Fields(s.AttrOr("class", ""))
	set := make(map[string]bool, len(classes))
	for _, c := range classes {
		set[strings.ToLower(c)] = true
	}
	for _, w := range want {
		if set[strings.ToLower(w)] {
			return w
		}
	}
	return ""
}

// tsToISO converts a data-ts attribute (unix seconds or milliseconds) to an
// RFC3339 UTC timestamp. It returns "" if the value is not numeric.
func tsToISO(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return ""
	}
	switch {
	case n > 1e14: // microseconds
		n /= 1e6
	case n > 1e11: // milliseconds
		n /= 1e3
	}
	if n < 1e8 { // implausibly small to be a unix time
		return ""
	}
	return time.Unix(n, 0).UTC().Format(time.RFC3339)
}

// Projects parses the account projects list (the authenticated "/" page).
func Projects(doc *goquery.Document) []model.Project {
	order := []string{}
	names := map[string]string{}
	doc.Find(`a[href]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := projectHref.FindStringSubmatch(href)
		if m == nil {
			return
		}
		id := "p" + m[1]
		label := text(a)
		if _, seen := names[id]; !seen {
			order = append(order, id)
		}
		// Prefer a meaningful label over the generic "View Stats" link.
		if names[id] == "" || (strings.EqualFold(names[id], "View Stats") && !strings.EqualFold(label, "View Stats")) {
			names[id] = label
		}
	})
	out := make([]model.Project, 0, len(order))
	for _, id := range order {
		out = append(out, model.Project{ID: id, Name: names[id]})
	}
	return out
}

// logRows returns the data rows of a results table (one <tr> with a data-visitor
// attribute per entry).
func logRows(doc *goquery.Document) *goquery.Selection {
	return doc.Find("table.results tr[data-visitor]")
}

// PageViews parses the "Page View Activity" (pageload) log.
func PageViews(doc *goquery.Document) []model.PageView {
	var out []model.PageView
	logRows(doc).Each(func(_ int, r *goquery.Selection) {
		pv := model.PageView{
			Timestamp: tsToISO(r.AttrOr("data-ts", "")),
			VisitorID: r.AttrOr("data-visitor", ""),
			Date:      text(r.Find("td.date").First()),
			Time:      text(r.Find("td.time").First()),
		}
		bd := r.Find("td.browser-device").First()
		pv.Device = firstNonEmpty(
			classToken(bd.Find("[class*=device-icon]").First(), "desktop", "mobile", "tablet"),
			bd.Find("[class*=device-icon]").First().AttrOr("title", ""),
		)
		pv.OS = trimDeviceSuffix(bd.Find(".os-overlay").First().AttrOr("title", ""))
		pv.Browser = bd.Find(".browser-icon").First().AttrOr("title", "")

		pv.Country = firstNonEmpty(
			r.Find("td.flag img.flag").First().AttrOr("title", ""),
			r.Find("td.flag img").First().AttrOr("alt", ""),
		)
		// The language cell is rendered as "<region> / <language>" (region may be
		// empty, e.g. "/ en").
		region, language := parseRegionLang(text(r.Find(".rpa-language").First()))
		pv.Region = firstNonEmpty(text(r.Find(".rpa-region").First()), region)
		pv.Language = language

		mix := r.Find("td.mix-l").First()
		// span.hostname is the visitor's resolved network host, i.e. their ISP
		// (e.g. "Synlinq", "Airtel Broadband").
		pv.ISP = text(mix.Find(".hostname").First())
		// The page loaded is the project-link (.pj-l), same convention as the
		// "entry page" column on the came-from log.
		page := mix.Find("a.pj-l").First()
		pv.Page = firstNonEmpty(text(page), page.AttrOr("href", ""))
		ref := mix.Find("a.ref-l").First()
		pv.Referrer = cleanRef(firstNonEmpty(ref.AttrOr("href", ""), text(ref)))

		out = append(out, pv)
	})
	return out
}

// CameFrom parses the "Recent Came From" (camefrom_activity) log.
func CameFrom(doc *goquery.Document) []model.CameFrom {
	var out []model.CameFrom
	logRows(doc).Each(func(_ int, r *goquery.Selection) {
		cf := model.CameFrom{
			Timestamp: tsToISO(r.AttrOr("data-ts", "")),
			VisitorID: r.AttrOr("data-visitor", ""),
			Date:      text(r.Find("td.date").First()),
			Time:      text(r.Find("td.time").First()),
		}
		ref := r.Find("a.ref-l").First()
		cf.Referrer = firstNonEmpty(ref.AttrOr("href", ""), text(ref))
		cf.Domain = text(r.Find("span.dm").First())

		// Entry page lives in the project-link (.pj-l) column.
		entry := r.Find(".pj-l").First()
		cf.EntryPage = firstNonEmpty(entry.Find("a").First().AttrOr("href", ""), text(entry))
		out = append(out, cf)
	})
	return out
}

// Visitors parses the "Visitor Activity" (visitor) log. Each <tr data-visitor>
// is one session, with details rendered as <dt>label</dt><dd>value</dd> pairs.
func Visitors(doc *goquery.Document) []model.Visitor {
	var out []model.Visitor
	logRows(doc).Each(func(_ int, r *goquery.Selection) {
		kv := dlMap(r)
		v := model.Visitor{
			Timestamp:     tsToISO(r.AttrOr("data-ts", "")),
			VisitorID:     r.AttrOr("data-visitor", ""),
			Date:          text(r.Find("span.date").First()),
			Time:          text(r.Find("span.time").First()),
			Country:       r.Find("img.flag").First().AttrOr("title", ""),
			ISP:           text(r.Find("span.hostname").First()),
			Location:      pick(kv, "location"),
			System:        pick(kv, "system"),
			Resolution:    pick(kv, "resolution"),
			ExitTime:      pick(kv, "exit time"),
			TotalSessions: pick(kv, "total sessions"),
			PageViews:     atoi(r.AttrOr("data-session-total-pageviews", "")),
		}
		ref := r.Find("a.ref-l").First()
		v.Referrer = cleanRef(firstNonEmpty(ref.AttrOr("href", ""), pick(kv, "referring url", "referrer")))
		entry := r.Find("a.pj-l").First()
		v.EntryPage = firstNonEmpty(text(entry), entry.AttrOr("href", ""), pick(kv, "visit page", "entry page"))
		out = append(out, v)
	})
	return out
}

// dlMap builds a normalized dt->dd map from the definition lists in a row.
// Labels are rendered with a trailing colon (e.g. "System:"), so keys are
// normalized to bare lowercase words ("system").
func dlMap(r *goquery.Selection) map[string]string {
	m := map[string]string{}
	r.Find("dt").Each(func(_ int, dt *goquery.Selection) {
		label := normKey(text(dt))
		if label == "" {
			return
		}
		if _, ok := m[label]; ok {
			return
		}
		m[label] = text(dt.NextAllFiltered("dd").First())
	})
	return m
}

// normKey lowercases a label and reduces it to letter/digit words separated by
// single spaces (dropping punctuation such as the trailing colon).
func normKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.TrimSpace(wsRun.ReplaceAllString(b.String(), " "))
}

// trimDeviceSuffix removes the redundant ", Desktop"/", Mobile"/", Tablet" suffix
// StatCounter appends to OS titles (e.g. "OS X, Desktop" -> "OS X").
func trimDeviceSuffix(s string) string {
	s = strings.TrimSpace(s)
	for _, d := range []string{", Desktop", ", Mobile", ", Tablet"} {
		if strings.HasSuffix(s, d) {
			return strings.TrimSpace(strings.TrimSuffix(s, d))
		}
	}
	return s
}

// parseRegionLang splits a "<region> / <language>" cell into its parts. With no
// slash the whole string is treated as the language.
func parseRegionLang(s string) (region, language string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return "", s
}

// cleanRef normalizes "direct visit" placeholders to an empty referrer.
func cleanRef(s string) string {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if s == "" || strings.Contains(low, "no referring link") || low == "direct" {
		return ""
	}
	return s
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func pick(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
