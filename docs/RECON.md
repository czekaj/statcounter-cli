# StatCounter dashboard — reverse-engineering notes

These notes document how the free-plan StatCounter web dashboard exposes data, as
discovered by inspecting the logged-in dashboard's network traffic and DOM. The
official API (`api.statcounter.com`) is **paid-only**, so this CLI drives the web
dashboard instead.

> The site is server-rendered HTML (jQuery + MochiKit + d3). There is **no JSON
> data API** behind the dashboard — data is baked into the page HTML. We log in,
> fetch the relevant pages, and parse the HTML.

## Base

- Host: `https://statcounter.com`
- All authenticated requests rely on session cookies set at login.

## Authentication

- **Login**: `POST https://statcounter.com/` (the form on `/login/` posts to `/`).
  - Fields: `form_user` (username/email), `form_pass` (password),
    `remember` (checkbox; send `on` for a longer-lived session).
  - No CSRF token field.
  - Success: session cookie(s) set; the response body is the Projects page.
  - Failure: the response re-renders the login form (`input[name=form_pass]`).
- The login form HTML is served by `GET /login/` **only when unauthenticated**;
  when already logged in, `/login/` redirects to `/`.

## Project discovery

- `GET /` (authenticated) returns the Projects list.
  - Each project: `a[href^="/p"][href$="/summary/"]`; href is `/p<ID>/summary/`,
    link text is the project name. Project id format: `p<digits>` (e.g. `p13239990`).
  - A CSV of the project list is available at `/?csv`.

## Project sections

`GET /p<ID>/<section>/`. Known sections (free plan):

| Section            | Path                       | Notes                                  |
|--------------------|----------------------------|----------------------------------------|
| Summary            | `summary/`                 | overview + d3 charts (server-rendered) |
| Pages              | `pages/`                   | popular/entry/exit; supports `?csv`    |
| Traffic Sources    | `traffic-sources/`         |                                        |
| Visitor Paths      | `path/`                    | supports `?csv`                        |
| Visitor Activity   | `visitor/`                 | log; "vertical-comparison" layout      |
| Page View Activity | `pageload/`                | log; `table.results.standard`          |
| Recent Came From   | `camefrom_activity/`       | log; `table.results.standard`          |
| Keyword Activity   | `keyword_activity/`        |                                        |
| Bots               | `bots/`                    |                                        |

- `?csv` returns CSV on aggregate pages (`pages/`, `path/`) but is **ignored** on
  the activity/log pages (returns the HTML page).
- Raw visitor log download: `GET /p<ID>/download_log/?range=all` (range values
  include `all`; the UI also offers narrower ranges). Returns an attachment.

## Log table row structures

### `pageload/` — Page View Activity
`table.results.standard.wrap-table.tags-table`, one `<tr>` per page view (~last 500).

Row: `<tr data-identity data-visitor data-ts>` — **`data-ts` is a unix-ish
timestamp** (preferred over parsing the visible date/time). 8 `<td>`:

1. `td.mag`            — `a.magnifier[href]` detail link
2. `td.date`           — date text (`<sup>` holds the ordinal suffix)
3. `td.time`           — time text + `span.ampm`
4. `td.browser-device` — `div.device-icon.{desktop|mobile}[title]`,
                          `div.os-overlay.<os>[title]`,
                          `span.browser-icon.<browser>[title]` (device/OS/browser
                          encoded in class names and the `title` attribute)
5. `td` (no class)     — `span.nobreak` × N (resolution / system details)
6. `td.flag`           — `img.flag[title]` (title = country name)
7. `td` (no class)     — `span.rpa-language` (region/language)
8. `td.mix-l`          — `span.hostname`, `a.visitor-label[href]` (host name /
                          web page / referrer)

### `camefrom_activity/` — Recent Came From
`table.results.standard.wrap-table`, one `<tr data-visitor>` per referral. 5 `<td>`:

1. `td.mag`   — detail link
2. `td.date`  — date
3. `td.time`  — time + `span.ampm`
4. `td.m-col` — **referrer**: `a.ind.ref-l.to-wrap[href]` (came-from URL),
                `span.dm` (domain)
5. `td.m-col` — **entry page**: `span.ind.pj-l.to-wrap`, `a[href]` (landing page)

### `visitor/` — Visitor Activity
`table.results.standard.vertical-comparison` — each visitor is a vertical block
rather than a single row; parsed best-effort.

## Output-scrubbing caveat (for future maintainers)

When inspecting via the browser tool, returning raw log content (IPs, 32-char
visitor hashes, referrer URLs with query strings) trips the assistant's output
filter. Inspect *structure* (tag/class/attribute names, counts), not values.
