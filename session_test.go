package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func sample() sess {
	return sess{
		Active: "prod",
		Creds: map[string]cred{
			"prod":  {User: "admin", Pass: "s3cret", At: 42},
			"stage": {User: "ro", Pass: "other-s3cret", At: 41},
		},
		Exp: time.Now().Add(time.Hour).Unix(),
	}
}

func TestSealRoundTripHidesEveryPassword(t *testing.T) {
	in := sample()

	tok, err := seal(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range in.Creds {
		if strings.Contains(tok, c.Pass) {
			t.Fatalf("password %q is readable in the cookie value", c.Pass)
		}
	}

	var out sess
	if err := unseal(tok, &out); err != nil {
		t.Fatal(err)
	}
	if out.Active != in.Active || len(out.Creds) != len(in.Creds) || out.Creds["stage"] != in.Creds["stage"] {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestUnsealRejectsTamperedToken(t *testing.T) {
	tok, err := seal(sample())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 0x01

	var out sess
	if err := unseal(base64.RawURLEncoding.EncodeToString(raw), &out); err == nil {
		t.Fatal("a tampered token was accepted")
	}
}

func TestExpiredSessionIsNotValid(t *testing.T) {
	s := sample()
	s.Exp = time.Now().Add(-time.Minute).Unix()
	if s.valid() {
		t.Fatal("an expired session was accepted")
	}
	if (sess{Exp: time.Now().Add(time.Hour).Unix()}).valid() {
		t.Fatal("a session holding no credentials was accepted")
	}
}

// The cookie has a hard size limit, so signing into many servers must evict
// rather than grow without bound - and must never evict the active one.
func TestEvictionKeepsTheActiveConnectionAndDropsTheOldest(t *testing.T) {
	s := sess{Active: "keep", Creds: map[string]cred{"keep": {At: 1}}}
	for i := range maxSignedIn + 3 {
		s.Creds[string(rune('a'+i))] = cred{At: int64(100 + i)}
	}
	evictOldest(&s)

	if len(s.Creds) != maxSignedIn {
		t.Fatalf("kept %d credentials, want %d", len(s.Creds), maxSignedIn)
	}
	if _, ok := s.Creds["keep"]; !ok {
		t.Fatal("the active connection was evicted despite being the oldest")
	}
	if _, ok := s.Creds["a"]; ok {
		t.Fatal("the oldest non-active connection survived")
	}
}

func TestAutoCredSignsInCookielessRequestsOnly(t *testing.T) {
	conns = []conn{{ID: "default", URL: "http://localhost:8123"}}
	r := httptest.NewRequest("GET", "/api/me", nil)

	if _, ok := session(r); ok {
		t.Fatal("signed in without a cookie or autoCred")
	}

	autoCred = &cred{User: "admin", Pass: "admin"}
	defer func() { autoCred = nil }()
	s, ok := session(r)
	if !ok || s.Active != "default" || s.Creds["default"].User != "admin" {
		t.Fatalf("autoCred did not sign in: %+v", s)
	}

	// A real cookie still wins, so switching servers keeps working.
	tok, _ := seal(sample())
	r.AddCookie(&http.Cookie{Name: cookieName, Value: tok})
	if s, _ := session(r); s.Active != "prod" {
		t.Fatalf("cookie ignored in favour of autoCred: %+v", s)
	}
}

func TestTypedServerIsDesktopOnlyAndBecomesTheActiveConnection(t *testing.T) {
	ch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[[1]]}`))
	}))
	defer ch.Close()

	oldConns, oldDesktop := conns, desktop
	defer func() { conns, desktop = oldConns, oldDesktop }()
	conns = []conn{{ID: "default", Name: "ClickHouse", URL: "http://127.0.0.1:1"}}

	login := func() *httptest.ResponseRecorder {
		body := `{"user":"admin","password":"pw","url":"` + ch.URL + `"}`
		w := httptest.NewRecorder()
		handleLogin(w, httptest.NewRequest("POST", "/api/login", strings.NewReader(body)))
		return w
	}

	desktop = nil
	if w := login(); w.Code != http.StatusForbidden {
		t.Fatalf("network build accepted a typed server: %d %s", w.Code, w.Body)
	}

	desktop = func(http.Handler) {}
	w := login()
	if w.Code != http.StatusOK {
		t.Fatalf("desktop login: %d %s", w.Code, w.Body)
	}
	r := httptest.NewRequest("GET", "/api/me", nil)
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	s, ok := session(r)
	if !ok {
		t.Fatal("no session after login")
	}
	target, c, ok := active(s)
	if !ok || target.URL != ch.URL || c.User != "admin" {
		t.Fatalf("active = %+v %+v %v, want %s as admin", target, c, ok, ch.URL)
	}
}
