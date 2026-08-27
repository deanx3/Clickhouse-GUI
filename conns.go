package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// A connection is a ClickHouse server chgui can reach. There are two kinds:
//
//   - configured: fixed by the operator in CHGUI_CONNECTIONS, shared by everyone
//   - own:        added by a user at runtime, stored only in that user's
//     encrypted session cookie, never visible to anyone else
//
// Set CHGUI_LOCK_CONNECTIONS=1 on a shared deployment to allow only the first
// kind. See checkTarget for why that switch exists.
type conn struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

const savedPrefix = "saved:"

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// loadConns reads CHGUI_CONNECTIONS, which is either inline JSON or a path to
// a JSON file. With neither set it falls back to the single CHGUI_CLICKHOUSE_URL
// server, so existing single-cluster deployments keep working untouched.
func loadConns(spec, fallbackURL string) ([]conn, error) {
	if strings.TrimSpace(spec) == "" {
		return []conn{{ID: "default", Name: "ClickHouse", URL: fallbackURL}}, nil
	}

	raw := []byte(spec)
	if !strings.HasPrefix(strings.TrimSpace(spec), "[") {
		b, err := os.ReadFile(spec)
		if err != nil {
			return nil, fmt.Errorf("reading CHGUI_CONNECTIONS: %w", err)
		}
		raw = b
	}

	var conns []conn
	if err := json.Unmarshal(raw, &conns); err != nil {
		return nil, fmt.Errorf("parsing CHGUI_CONNECTIONS: %w", err)
	}
	if len(conns) == 0 {
		return nil, fmt.Errorf("CHGUI_CONNECTIONS lists no connections")
	}

	seen := make(map[string]bool, len(conns))
	for i, c := range conns {
		if !idPattern.MatchString(c.ID) {
			return nil, fmt.Errorf("connection %d: id %q must be lowercase letters, digits, - or _", i, c.ID)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("connection id %q appears twice", c.ID)
		}
		seen[c.ID] = true

		clean, err := normalizeURL(c.URL)
		if err != nil {
			return nil, fmt.Errorf("connection %q: %w", c.ID, err)
		}
		conns[i].URL = clean
		if c.Name == "" {
			conns[i].Name = c.ID
		}
	}
	return conns, nil
}

func normalizeURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("url must be an absolute http(s) URL, got %q", raw)
	}
	// Anything past the origin would be silently dropped when we append "/?..."
	// so reject it rather than connect somewhere the user did not mean.
	if u.RawQuery != "" || u.Fragment != "" || strings.Trim(u.Path, "/") != "" {
		return "", fmt.Errorf("url must be just scheme://host:port, got %q", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

// checkTarget is the guard on user-added connections. The server, not the
// browser, makes the outbound request, so an arbitrary URL turns chgui into a
// request proxy positioned inside the network. Cloud metadata endpoints hand
// out credentials to anything that asks, so they are refused outright.
func checkTarget(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := u.Hostname()

	if strings.EqualFold(host, "metadata.google.internal") {
		return fmt.Errorf("that host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLinkLocalUnicast() {
		// 169.254.0.0/16 and fe80::/10 - the cloud metadata range.
		return fmt.Errorf("link-local addresses are not allowed")
	}
	return nil
}

func findConn(conns []conn, id string) (conn, bool) {
	for _, c := range conns {
		if c.ID == id {
			return c, true
		}
	}
	return conn{}, false
}

// slugify turns a display name into a stable, unique own: id.
func slugify(name string, taken map[string]bool) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "server"
	}

	id := savedPrefix + base
	for i := 2; taken[id]; i++ {
		id = fmt.Sprintf("%s%s-%d", savedPrefix, base, i)
	}
	return id
}
