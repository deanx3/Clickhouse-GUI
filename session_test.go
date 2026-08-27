package main

import (
	"encoding/base64"
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
