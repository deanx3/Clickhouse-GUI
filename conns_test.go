package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFallsBackToTheSingleConfiguredServer(t *testing.T) {
	got, err := loadConns("", "http://localhost:8123")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "default" || got[0].URL != "http://localhost:8123" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadsInlineJSONAndAFile(t *testing.T) {
	inline := `[{"id":"prod","name":"Production","url":"http://ch-prod:8123/"}]`

	fromInline, err := loadConns(inline, "")
	if err != nil {
		t.Fatal(err)
	}
	// A trailing slash would double up when paths are appended.
	if fromInline[0].URL != "http://ch-prod:8123" {
		t.Fatalf("trailing slash not trimmed: %q", fromInline[0].URL)
	}

	path := filepath.Join(t.TempDir(), "conns.json")
	if err := os.WriteFile(path, []byte(inline), 0o600); err != nil {
		t.Fatal(err)
	}
	fromFile, err := loadConns(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if fromFile[0] != fromInline[0] {
		t.Fatalf("file and inline disagree: %+v vs %+v", fromFile[0], fromInline[0])
	}
}

func TestRejectsConfigurationThatCouldBeAbused(t *testing.T) {
	for name, spec := range map[string]string{
		"non-http scheme": `[{"id":"a","url":"file:///etc/passwd"}]`,
		"relative url":    `[{"id":"a","url":"/etc/passwd"}]`,
		"no host":         `[{"id":"a","url":"http://"}]`,
		"duplicate id":    `[{"id":"a","url":"http://x:8123"},{"id":"a","url":"http://y:8123"}]`,
		"id with a slash": `[{"id":"a/b","url":"http://x:8123"}]`,
		"empty id":        `[{"id":"","url":"http://x:8123"}]`,
		"empty list":      `[]`,
		"not json":        `[oops`,
	} {
		if _, err := loadConns(spec, ""); err == nil {
			t.Errorf("%s: accepted %s", name, spec)
		}
	}
}

func TestNameDefaultsToID(t *testing.T) {
	got, err := loadConns(`[{"id":"staging","url":"http://x:8123"}]`, "")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Name != "staging" {
		t.Fatalf("got name %q", got[0].Name)
	}
}

func TestNormalizeURLRejectsAnythingPastTheOrigin(t *testing.T) {
	ok := map[string]string{
		"http://ch:8123":     "http://ch:8123",
		" https://ch:8443/ ": "https://ch:8443",
		"http://ch.internal": "http://ch.internal",
	}
	for in, want := range ok {
		got, err := normalizeURL(in)
		if err != nil || got != want {
			t.Errorf("normalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	// A path or query would be silently dropped when "/?..." is appended, so
	// the user would be querying a server they did not think they named.
	for _, bad := range []string{
		"http://ch:8123/some/path", "http://ch:8123/?query=1", "http://ch:8123/#x",
		"ftp://ch:8123", "file:///etc/passwd", "ch:8123", "", "http://",
	} {
		if got, err := normalizeURL(bad); err == nil {
			t.Errorf("normalizeURL(%q) accepted, gave %q", bad, got)
		}
	}
}

// The server makes the outbound request, so cloud metadata endpoints - which
// hand credentials to any caller - must be refused however they are spelled.
func TestCheckTargetRefusesMetadataEndpoints(t *testing.T) {
	for _, bad := range []string{
		"http://169.254.169.254", "http://169.254.169.254:80",
		"http://metadata.google.internal", "http://METADATA.GOOGLE.INTERNAL",
		"http://[fe80::1]:8123",
	} {
		if err := checkTarget(bad); err == nil {
			t.Errorf("checkTarget(%q) allowed it", bad)
		}
	}

	// Ordinary internal ClickHouse hosts must still work.
	for _, good := range []string{
		"http://localhost:8123", "http://10.0.0.5:8123",
		"http://ch-prod.internal:8123", "https://clickhouse.example.com",
	} {
		if err := checkTarget(good); err != nil {
			t.Errorf("checkTarget(%q) = %v, want nil", good, err)
		}
	}
}

func TestSlugifyMakesStableUniqueIDs(t *testing.T) {
	taken := map[string]bool{}
	first := slugify("Prod EU-West #1", taken)
	if first != "saved:prod-eu-west-1" {
		t.Fatalf("got %q", first)
	}
	taken[first] = true
	if second := slugify("Prod EU-West #1", taken); second != "saved:prod-eu-west-1-2" {
		t.Fatalf("collision not resolved, got %q", second)
	}
	if odd := slugify("!!!", map[string]bool{}); odd != "saved:server" {
		t.Fatalf("unnameable connection got %q", odd)
	}
}

func TestReachErrPointsAtThePortWhenThatIsTheLikelyCause(t *testing.T) {
	refused := errors.New(`Post "http://101.46.52.153/?x": dial tcp 101.46.52.153:80: connect: connection refused`)

	// The real case that prompted this: no port given, so Go dialled 80.
	got := reachErr(conn{Name: "Haramain", URL: "http://101.46.52.153"}, refused).Error()
	if !strings.Contains(got, "8123") {
		t.Errorf("no port hint for a bare host: %q", got)
	}
	if strings.Contains(got, "dial tcp") {
		t.Errorf("raw dial error leaked to the user: %q", got)
	}

	// With an explicit non-default port, guessing 8123 would be wrong advice.
	got = reachErr(conn{Name: "Haramain", URL: "http://101.46.52.153:9000"}, refused).Error()
	if strings.Contains(got, "8123") {
		t.Errorf("suggested 8123 despite an explicit port: %q", got)
	}
	if !strings.Contains(got, "9000") {
		t.Errorf("did not name the port actually used: %q", got)
	}

	for msg, want := range map[string]string{
		"dial tcp: lookup nope.invalid: no such host":          "no such host",
		"context deadline exceeded (i/o timeout)":              "did not respond in time",
		`tls: first record does not look like a TLS handshake`: "use http instead of https",
	} {
		got := reachErr(conn{Name: "X", URL: "http://x:8123"}, errors.New(msg)).Error()
		if !strings.Contains(got, want) {
			t.Errorf("%q -> %q, want it to mention %q", msg, got, want)
		}
	}
}
