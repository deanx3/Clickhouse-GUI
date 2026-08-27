package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// store holds the connections people save. It is shared: everyone using this
// deployment sees the same list, and a saved connection carries the credentials
// it was added with, so anyone who can reach chgui can query as that user.
//
// ClickHouse passwords are encrypted rather than hashed, because chgui has to
// replay them on every query - a hash could never be sent back to ClickHouse.
type store struct{ db *sql.DB }

// saved is one row. Pass is only populated by get(), never by list().
type saved struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	User      string `json:"user"`
	CreatedBy string `json:"createdBy"`
	CreatedAt int64  `json:"createdAt"`

	// Broken means the stored password will not decrypt, which happens when
	// CHGUI_SECRET is rotated. The row is listed but cannot be used.
	Broken bool   `json:"broken"`
	Pass   string `json:"-"`
}

const schema = `
CREATE TABLE IF NOT EXISTS connections (
  id         TEXT    PRIMARY KEY,
  name       TEXT    NOT NULL,
  url        TEXT    NOT NULL,
  ch_user    TEXT    NOT NULL,
  secret     BLOB    NOT NULL,
  created_by TEXT    NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS connections_target ON connections(url, ch_user);
`

// defaultDBPath follows the XDG data spec so the installed app and a
// hand-started binary agree on where the file lives.
func defaultDBPath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "chgui.db"
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "chgui", "chgui.db")
}

func openStore(path string) (*store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	// WAL survives an unclean shutdown; the busy timeout keeps concurrent
	// requests from failing outright when one is mid-write.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("preparing %s: %w", path, err)
	}
	// The file holds encrypted credentials; it is nobody else's business.
	_ = os.Chmod(path, 0o600)
	return &store{db: db}, nil
}

func (s *store) Close() error { return s.db.Close() }

// list returns every saved connection. Passwords are checked for decryptability
// so a stale row can be shown as broken rather than failing later with a
// confusing message, but the plaintext is discarded immediately and never
// travels with the result.
func (s *store) list() ([]saved, error) {
	rows, err := s.db.Query(
		`SELECT id, name, url, ch_user, secret, created_by, created_at FROM connections ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []saved{}
	for rows.Next() {
		var c saved
		var blob []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.URL, &c.User, &blob, &c.CreatedBy, &c.CreatedAt); err != nil {
			return nil, err
		}
		_, err := openBytes(storeAEAD, blob)
		c.Broken = err != nil
		out = append(out, c)
	}
	return out, rows.Err()
}

var errNoSuchConn = errors.New("no such connection")

// get returns one connection with its password decrypted, for actually running
// a query. A row that will not decrypt means the master key changed.
func (s *store) get(id string) (saved, error) {
	var c saved
	var blob []byte
	err := s.db.QueryRow(
		`SELECT id, name, url, ch_user, secret, created_by, created_at FROM connections WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.URL, &c.User, &blob, &c.CreatedBy, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return saved{}, errNoSuchConn
	}
	if err != nil {
		return saved{}, err
	}

	plain, err := openBytes(storeAEAD, blob)
	if err != nil {
		return saved{}, fmt.Errorf("%q cannot be decrypted - CHGUI_SECRET has changed since it was saved", c.Name)
	}
	c.Pass = string(plain)
	return c, nil
}

func (s *store) add(c saved) (saved, error) {
	blob, err := sealBytes(storeAEAD, []byte(c.Pass))
	if err != nil {
		return saved{}, err
	}

	existing, err := s.list()
	if err != nil {
		return saved{}, err
	}
	taken := make(map[string]bool, len(existing))
	for _, e := range existing {
		taken[e.ID] = true
	}

	c.ID = slugify(c.Name, taken)
	c.CreatedAt = time.Now().Unix()

	_, err = s.db.Exec(
		`INSERT INTO connections (id, name, url, ch_user, secret, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.URL, c.User, blob, c.CreatedBy, c.CreatedAt)
	if err != nil {
		// The unique index on (url, ch_user) is the only constraint here.
		return saved{}, fmt.Errorf("%s as %s is already saved", c.URL, c.User)
	}
	c.Pass = ""
	return c, nil
}

func (s *store) remove(id string) error {
	res, err := s.db.Exec(`DELETE FROM connections WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNoSuchConn
	}
	return nil
}
