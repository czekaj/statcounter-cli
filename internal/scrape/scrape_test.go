package scrape

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func doc(t *testing.T, html string) *goquery.Document {
	t.Helper()
	d, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	return d
}

func TestProjects(t *testing.T) {
	html := `<html><body>
	<a href="/p100/summary/">Acme Site</a>
	<a href="/p100/summary/">View Stats</a>
	<a href="/p200/summary/">View Stats</a>
	<a href="/p200/summary/">Other Co</a>
	<a href="/add-project/">Add</a>
	<a href="/p300/settings/">settings not summary</a>
	</body></html>`
	got := Projects(doc(t, html))
	if len(got) != 2 {
		t.Fatalf("want 2 projects, got %d: %+v", len(got), got)
	}
	if got[0].ID != "p100" || got[0].Name != "Acme Site" {
		t.Errorf("project[0] = %+v", got[0])
	}
	// Name should prefer the meaningful label over "View Stats" regardless of order.
	if got[1].ID != "p200" || got[1].Name != "Other Co" {
		t.Errorf("project[1] = %+v", got[1])
	}
}

func TestPageViews(t *testing.T) {
	// data-ts 1700000000 -> 2023-11-14T22:13:20Z
	html := `<html><body><table class="results standard">
	<tr data-visitor="abc" data-ts="1700000000">
		<td class="mag"><a class="magnifier" href="#"></a></td>
		<td class="date">14th Nov</td>
		<td class="time">22:13<span class="ampm">pm</span></td>
		<td class="browser-device"><div>
			<div class="device-icon desktop screenindicator" title="Desktop"></div>
			<div class="os-overlay os_x desktop" title="OS X, Desktop"></div>
			<span class="browser-icon chrome browser-desktop" title="Chrome"></span>
		</div></td>
		<td><span class="nobreak">1440x900</span></td>
		<td class="flag"><img class="flag" title="United States"></td>
		<td><span class="rpa-language">/ en</span></td>
		<td class="mix-l">
			<span class="hostname">Comcast Cable</span>
			<a class="visitor-label" href="/null">label</a>
			<a class="ind pj-l to-wrap" href="/pricing/">/pricing/</a>
			<a class="ind ref-l to-wrap" href="https://google.com/search?q=x">google</a>
		</td>
	</tr>
	</table></body></html>`
	got := PageViews(doc(t, html))
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	pv := got[0]
	checks := map[string]string{
		"VisitorID": pv.VisitorID, "Timestamp": pv.Timestamp, "Device": pv.Device,
		"OS": pv.OS, "Browser": pv.Browser, "Country": pv.Country,
		"ISP": pv.ISP, "Page": pv.Page, "Referrer": pv.Referrer, "Language": pv.Language,
	}
	want := map[string]string{
		"VisitorID": "abc", "Timestamp": "2023-11-14T22:13:20Z", "Device": "desktop",
		"OS": "OS X", "Browser": "Chrome", "Country": "United States",
		"ISP": "Comcast Cable", "Page": "/pricing/", "Referrer": "https://google.com/search?q=x",
		"Language": "en",
	}
	for k, w := range want {
		if checks[k] != w {
			t.Errorf("%s = %q, want %q", k, checks[k], w)
		}
	}
}

func TestCameFrom(t *testing.T) {
	html := `<html><body><table class="results standard">
	<tr data-visitor="v1">
		<td class="mag"><a class="magnifier" href="#"></a></td>
		<td class="date">14th Nov</td>
		<td class="time">09:00<span class="ampm">am</span></td>
		<td class="m-col">
			<a class="ind ref-l to-wrap" href="https://news.example/article">news.example</a>
			<span class="dm">news.example</span>
		</td>
		<td class="m-col">
			<span class="ind pj-l to-wrap"><a href="/landing/">/landing/</a></span>
		</td>
	</tr>
	</table></body></html>`
	got := CameFrom(doc(t, html))
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	cf := got[0]
	if cf.VisitorID != "v1" || cf.Referrer != "https://news.example/article" ||
		cf.Domain != "news.example" || cf.EntryPage != "/landing/" {
		t.Errorf("camefrom = %+v", cf)
	}
}

func TestVisitors(t *testing.T) {
	html := `<html><body><table class="results standard">
	<tr data-visitor="v9" data-ts="1700000000" data-session-total-pageviews="3">
		<td class="mag"></td>
		<td class="spi"><dl>
			<dt>Page Views:</dt><dd>3</dd>
			<dt>Exit Time:</dt><dd>09:05</dd>
			<dt>Resolution:</dt><dd>1920x1080</dd>
			<dt>System:</dt><dd>Windows / Chrome</dd>
			<dt>Total Sessions:</dt><dd>2</dd>
			<span class="date">14th Nov</span><span class="time">09:00</span>
		</dl></td>
		<td class="mix-l"><dl>
			<dt>Location:</dt><dd>New York, United States</dd>
			<dt>ISP / IP Address:</dt><dd>Acme ISP</dd>
			<dt>Referring URL:</dt><dd><a class="ref-l" href="https://t.co/abc">t.co</a></dd>
			<dt>Visit Page:</dt><dd><a class="pj-l" href="/home/">/home/</a></dd>
			<img class="flag" title="United States">
			<span class="hostname">Acme ISP</span>
		</dl></td>
	</tr>
	</table></body></html>`
	got := Visitors(doc(t, html))
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	v := got[0]
	if v.VisitorID != "v9" || v.Timestamp != "2023-11-14T22:13:20Z" {
		t.Errorf("id/ts = %q/%q", v.VisitorID, v.Timestamp)
	}
	if v.PageViews != 3 || v.System != "Windows / Chrome" || v.Resolution != "1920x1080" {
		t.Errorf("pv/system/res = %d/%q/%q", v.PageViews, v.System, v.Resolution)
	}
	if v.Location != "New York, United States" || v.Country != "United States" || v.ISP != "Acme ISP" {
		t.Errorf("location/country/isp = %q/%q/%q", v.Location, v.Country, v.ISP)
	}
	if v.Referrer != "https://t.co/abc" || v.EntryPage != "/home/" {
		t.Errorf("ref/entry = %q/%q", v.Referrer, v.EntryPage)
	}
	if v.TotalSessions != "2" || v.ExitTime != "09:05" {
		t.Errorf("sessions/exit = %q/%q", v.TotalSessions, v.ExitTime)
	}
}

func TestCleanupHelpers(t *testing.T) {
	if got := trimDeviceSuffix("Win11, Desktop"); got != "Win11" {
		t.Errorf("trimDeviceSuffix = %q", got)
	}
	if got := trimDeviceSuffix("Android, Mobile"); got != "Android" {
		t.Errorf("trimDeviceSuffix mobile = %q", got)
	}
	if got := trimDeviceSuffix("OS X"); got != "OS X" {
		t.Errorf("trimDeviceSuffix passthrough = %q", got)
	}
	if r, l := parseRegionLang("/ en"); r != "" || l != "en" {
		t.Errorf("parseRegionLang(/ en) = %q/%q", r, l)
	}
	if r, l := parseRegionLang("California / en-US"); r != "California" || l != "en-US" {
		t.Errorf("parseRegionLang(region) = %q/%q", r, l)
	}
	for _, s := range []string{"", "(No referring link)", "no referring link", "Direct"} {
		if got := cleanRef(s); got != "" {
			t.Errorf("cleanRef(%q) = %q, want empty", s, got)
		}
	}
	if got := cleanRef("https://x.com/"); got != "https://x.com/" {
		t.Errorf("cleanRef(url) = %q", got)
	}
}

func TestTsToISO(t *testing.T) {
	cases := map[string]string{
		"1700000000":    "2023-11-14T22:13:20Z", // seconds
		"1700000000000": "2023-11-14T22:13:20Z", // milliseconds
		"":              "",
		"not-a-number":  "",
		"123":           "", // too small
	}
	for in, want := range cases {
		if got := tsToISO(in); got != want {
			t.Errorf("tsToISO(%q) = %q, want %q", in, got, want)
		}
	}
}
