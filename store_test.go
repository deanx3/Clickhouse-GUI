package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func testStore(t *testing.T) *store {
	t.Helper()
	s, err := openStore(filepath.Join(t.TempDir(), "chgui.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSavedPasswordIsEncryptedAtRestAndComesBackIntact(t *testing.T) {
	s := testStore(t)
	// The password that started all this - every shell metacharacter in it.
	const pw = `D@n!aL#9Xv$7QpL2^tZr8K!mW4@cYhF6`

	added, err := s.add(saved{Name: "Haramain", URL: "http://ch:8123", User: "danial", Pass: pw, CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if added.Pass != "" {
		t.Error("add() handed the password back out")
	}

	// Nothing readable may sit in the column.
	var blob []byte
	if err := s.db.QueryRow(`SELECT secret FROM connections WHERE id = ?`, added.ID).Scan(&blob); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), pw) || strings.Contains(string(blob), "cYhF6") {
		t.Fatal("the password is readable in the database file")
	}

	got, err := s.get(added.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Pass != pw {
		t.Fatalf("round trip changed the password:\n got %q\nwant %q", got.Pass, pw)
	}
	if got.CreatedBy != "admin" {
		t.Errorf("createdBy = %q", got.CreatedBy)
	}
}

func TestListNeverDecryptsPasswords(t *testing.T) {
	s := testStore(t)
	if _, err := s.add(saved{Name: "A", URL: "http://a:8123", User: "u", Pass: "hunter2"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.list()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d rows", len(list))
	}
	if list[0].Pass != "" {
		t.Fatal("list() decrypted a password it had no reason to touch")
	}
}

func TestTheSameServerAndUserCannotBeSavedTwice(t *testing.T) {
	s := testStore(t)
	row := saved{Name: "A", URL: "http://a:8123", User: "u", Pass: "p"}
	if _, err := s.add(row); err != nil {
		t.Fatal(err)
	}
	if _, err := s.add(row); err == nil {
		t.Fatal("duplicate accepted")
	}
	// A different user on the same server is a different connection.
	if _, err := s.add(saved{Name: "A2", URL: "http://a:8123", User: "other", Pass: "p"}); err != nil {
		t.Fatalf("same server, different user was rejected: %v", err)
	}
}

func TestRemoveAndMissingRows(t *testing.T) {
	s := testStore(t)
	added, err := s.add(saved{Name: "A", URL: "http://a:8123", User: "u", Pass: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.get("nope"); err != errNoSuchConn {
		t.Errorf("get(missing) = %v, want errNoSuchConn", err)
	}
	if err := s.remove("nope"); err != errNoSuchConn {
		t.Errorf("remove(missing) = %v, want errNoSuchConn", err)
	}
	if err := s.remove(added.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.get(added.ID); err != errNoSuchConn {
		t.Errorf("row survived removal: %v", err)
	}
}

// Rotating CHGUI_SECRET must fail loudly rather than hand back rubbish.
func TestRowsSavedUnderAnotherKeyAreRefusedClearly(t *testing.T) {
	s := testStore(t)
	added, err := s.add(saved{Name: "Prod", URL: "http://a:8123", User: "u", Pass: "p"})
	if err != nil {
		t.Fatal(err)
	}

	garbage, err := sealBytes(aeadFor("some-other-key"), []byte("p"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE connections SET secret = ? WHERE id = ?`, garbage, added.ID); err != nil {
		t.Fatal(err)
	}

	_, err = s.get(added.ID)
	if err == nil {
		t.Fatal("decrypted a row sealed with a different key")
	}
	if !strings.Contains(err.Error(), "CHGUI_SECRET") {
		t.Errorf("error does not explain the cause: %v", err)
	}
}

// Each purpose must get its own key, or a stolen cookie could read the store.
func TestCookieAndStoreKeysAreDistinct(t *testing.T) {
	blob, err := sealBytes(storeAEAD, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openBytes(cookieAEAD, blob); err == nil {
		t.Fatal("the cookie key can decrypt the connection store")
	}
}

func TestSaveKeepsOneRowPerServerAndUserAndRefreshesThePassword(t *testing.T) {
	s := testStore(t)
	first, err := s.save(saved{Name: "ch:8123", URL: "http://ch:8123", User: "admin", Pass: "old"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.save(saved{Name: "ch:8123", URL: "http://ch:8123", User: "admin", Pass: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID {
		t.Fatalf("second save made a new row: %s vs %s", again.ID, first.ID)
	}
	if list, _ := s.list(); len(list) != 1 {
		t.Fatalf("%d rows, want 1", len(list))
	}
	if got, _ := s.get(first.ID); got.Pass != "new" {
		t.Fatalf("password = %q, want the new one", got.Pass)
	}
}
