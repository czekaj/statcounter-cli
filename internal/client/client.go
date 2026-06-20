// Package client handles authentication against the StatCounter web dashboard
// and fetching of (server-rendered) pages. The free plan has no API, so we log
// in with the user's credentials, persist the session cookies, and request the
// same HTML pages the dashboard serves.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// BaseURL is the StatCounter dashboard origin.
const BaseURL = "https://statcounter.com"

// ErrNotAuthenticated is returned when no valid session exists and no
// credentials are available to establish one.
var ErrNotAuthenticated = errors.New("not authenticated: run `statcounter login`, or set STATCOUNTER_USERNAME and STATCOUNTER_PASSWORD")

// ErrLoginFailed is returned when credentials are rejected.
var ErrLoginFailed = errors.New("login failed: check your username and password")

// Version is stamped by the CLI at startup so the User-Agent identifies the tool.
var Version = "dev"

func userAgent() string {
	return fmt.Sprintf("Mozilla/5.0 (compatible; statcounter-cli/%s; +https://github.com/czekaj/statcounter-cli)", Version)
}

// Client is an authenticated StatCounter dashboard client.
type Client struct {
	http    *http.Client
	jar     *cookiejar.Jar
	base    *url.URL
	cfgDir  string
	user    string
	hadSess bool
}

type sessionFile struct {
	User    string         `json:"user,omitempty"`
	Cookies []sessionCooky `json:"cookies"`
	Saved   time.Time      `json:"saved"`
}

type sessionCooky struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// New constructs a Client, loading any persisted session from disk.
func New() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	base, _ := url.Parse(BaseURL)
	cfgDir, err := configDir()
	if err != nil {
		return nil, err
	}
	c := &Client{
		jar:    jar,
		base:   base,
		cfgDir: cfgDir,
		http: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
	c.loadSession()
	return c, nil
}

// configDir returns the directory used to store the session, honoring
// STATCOUNTER_CONFIG_DIR and XDG_CONFIG_HOME.
func configDir() (string, error) {
	if d := os.Getenv("STATCOUNTER_CONFIG_DIR"); d != "" {
		return d, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "statcounter"), nil
}

func (c *Client) sessionPath() string { return filepath.Join(c.cfgDir, "session.json") }

// Username returns the username associated with the loaded/established session.
func (c *Client) Username() string { return c.user }

func (c *Client) loadSession() {
	data, err := os.ReadFile(c.sessionPath())
	if err != nil {
		return
	}
	var sf sessionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return
	}
	cookies := make([]*http.Cookie, 0, len(sf.Cookies))
	for _, ck := range sf.Cookies {
		cookies = append(cookies, &http.Cookie{Name: ck.Name, Value: ck.Value})
	}
	if len(cookies) > 0 {
		c.jar.SetCookies(c.base, cookies)
		c.hadSess = true
	}
	c.user = sf.User
}

func (c *Client) saveSession() error {
	if err := os.MkdirAll(c.cfgDir, 0o700); err != nil {
		return err
	}
	var sf sessionFile
	sf.User = c.user
	sf.Saved = time.Now().UTC()
	for _, ck := range c.jar.Cookies(c.base) {
		sf.Cookies = append(sf.Cookies, sessionCooky{Name: ck.Name, Value: ck.Value})
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.sessionPath(), data, 0o600)
}

// Logout clears the persisted session.
func (c *Client) Logout() error {
	c.user = ""
	c.hadSess = false
	jar, _ := cookiejar.New(nil)
	c.jar = jar
	c.http.Jar = jar
	err := os.Remove(c.sessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (c *Client) newRequest(method, rawurl string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, rawurl, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	return req, nil
}

// Login authenticates with the given credentials and persists the session.
func (c *Client) Login(username, password string, remember bool) error {
	form := url.Values{}
	form.Set("form_user", username)
	form.Set("form_pass", password)
	if remember {
		form.Set("remember", "on")
	}
	req, err := c.newRequest(http.MethodPost, c.base.String()+"/", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.base.String()+"/login/")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("parsing login response: %w", err)
	}
	if !isAuthed(doc) {
		return ErrLoginFailed
	}
	c.user = username
	if err := c.saveSession(); err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	c.hadSess = true
	return nil
}

// isAuthed reports whether a fetched page is an authenticated dashboard page.
// Every dashboard page renders a logout link in the global nav; the marketing
// site and the "Login Required" interstitial do not. This is more reliable than
// looking for the login form, because the unauthenticated projects page ("/")
// returns the marketing homepage rather than the login form.
func isAuthed(doc *goquery.Document) bool {
	return doc.Find(`a[href*="logout"]`).Length() > 0
}

// getDoc fetches a dashboard path and returns the parsed document.
func (c *Client) getDoc(path string) (*goquery.Document, error) {
	req, err := c.newRequest(http.MethodGet, c.base.String()+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("statcounter returned HTTP %d for %s", resp.StatusCode, path)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}

// AuthedDoc fetches a path that requires authentication. If the session is
// missing or expired and STATCOUNTER_USERNAME/STATCOUNTER_PASSWORD are set, it
// transparently logs in once and retries.
func (c *Client) AuthedDoc(path string) (*goquery.Document, error) {
	doc, err := c.getDoc(path)
	if err != nil {
		return nil, err
	}
	if isAuthed(doc) {
		return doc, nil
	}
	// Session missing/expired. Try environment credentials.
	user, pass := os.Getenv("STATCOUNTER_USERNAME"), os.Getenv("STATCOUNTER_PASSWORD")
	if user == "" || pass == "" {
		return nil, ErrNotAuthenticated
	}
	if err := c.Login(user, pass, true); err != nil {
		return nil, err
	}
	doc, err = c.getDoc(path)
	if err != nil {
		return nil, err
	}
	if !isAuthed(doc) {
		return nil, ErrNotAuthenticated
	}
	return doc, nil
}

// CheckAuth verifies the current session is valid by loading the projects page.
func (c *Client) CheckAuth() (bool, error) {
	doc, err := c.getDoc("/")
	if err != nil {
		return false, err
	}
	return isAuthed(doc), nil
}
