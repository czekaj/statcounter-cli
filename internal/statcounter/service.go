// Package statcounter is the high-level service used by both the CLI commands
// and the MCP server. It composes the authenticated client with the HTML
// scrapers and exposes one method per data type.
package statcounter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/czekaj/statcounter-cli/internal/client"
	"github.com/czekaj/statcounter-cli/internal/model"
	"github.com/czekaj/statcounter-cli/internal/scrape"
)

// Service is the application's data access layer.
type Service struct {
	c *client.Client
}

// New constructs a Service backed by a client that loads any saved session.
func New() (*Service, error) {
	c, err := client.New()
	if err != nil {
		return nil, err
	}
	return &Service{c: c}, nil
}

// Login authenticates and persists the session.
func (s *Service) Login(user, pass string, remember bool) error {
	return s.c.Login(user, pass, remember)
}

// Logout clears the saved session.
func (s *Service) Logout() error { return s.c.Logout() }

// Username returns the logged-in username, if known.
func (s *Service) Username() string { return s.c.Username() }

// CheckAuth verifies the session is currently valid.
func (s *Service) CheckAuth() (bool, error) { return s.c.CheckAuth() }

// Projects lists all projects in the account.
func (s *Service) Projects() ([]model.Project, error) {
	doc, err := s.c.AuthedDoc("/")
	if err != nil {
		return nil, err
	}
	return scrape.Projects(doc), nil
}

var idRe = regexp.MustCompile(`^p?(\d+)$`)

// ResolveProject turns a user-supplied project reference (numeric id, "pNNN", or
// a name/domain substring) into a canonical "pNNN" id.
func (s *Service) ResolveProject(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("no project specified (use --project; see `statcounter projects`)")
	}
	if m := idRe.FindStringSubmatch(ref); m != nil {
		return "p" + m[1], nil
	}
	projects, err := s.Projects()
	if err != nil {
		return "", err
	}
	// Exact name match first, then case-insensitive substring.
	for _, p := range projects {
		if strings.EqualFold(p.Name, ref) {
			return p.ID, nil
		}
	}
	var matches []model.Project
	low := strings.ToLower(ref)
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.Name), low) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0].ID, nil
	case 0:
		return "", fmt.Errorf("no project matches %q (see `statcounter projects`)", ref)
	default:
		var names []string
		for _, m := range matches {
			names = append(names, fmt.Sprintf("%s (%s)", m.Name, m.ID))
		}
		return "", fmt.Errorf("%q is ambiguous, matches: %s", ref, strings.Join(names, ", "))
	}
}

func clamp[T any](items []T, limit int) []T {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

// PageViews returns the Page View Activity log for a project.
func (s *Service) PageViews(project string, limit int) ([]model.PageView, error) {
	id, err := s.ResolveProject(project)
	if err != nil {
		return nil, err
	}
	doc, err := s.c.AuthedDoc("/" + id + "/pageload/")
	if err != nil {
		return nil, err
	}
	return clamp(scrape.PageViews(doc), limit), nil
}

// CameFrom returns the Recent Came From log for a project.
func (s *Service) CameFrom(project string, limit int) ([]model.CameFrom, error) {
	id, err := s.ResolveProject(project)
	if err != nil {
		return nil, err
	}
	doc, err := s.c.AuthedDoc("/" + id + "/camefrom_activity/")
	if err != nil {
		return nil, err
	}
	return clamp(scrape.CameFrom(doc), limit), nil
}

// Visitors returns the Visitor Activity log for a project.
func (s *Service) Visitors(project string, limit int) ([]model.Visitor, error) {
	id, err := s.ResolveProject(project)
	if err != nil {
		return nil, err
	}
	doc, err := s.c.AuthedDoc("/" + id + "/visitor/")
	if err != nil {
		return nil, err
	}
	return clamp(scrape.Visitors(doc), limit), nil
}
