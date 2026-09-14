// chgui is a ClickHouse admin console. It holds no state and no user store:
// you sign in with your ClickHouse credentials, and every statement runs as
// you, so ClickHouse itself enforces what you are allowed to see and change.
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed all:web/dist
var distFS embed.FS

const (
	cookieName = "chgui"
	sessionTTL = 12 * time.Hour
	maxBody    = 1 << 20

	// The whole session rides in one cookie, so the number of servers you can
	// stay signed into at once is bounded to keep it well under the 4KB limit.
	maxSignedIn = 8
)

var (
	addr     = env("CHGUI_ADDR", ":8080")
	secure   = os.Getenv("CHGUI_SECURE_COOKIE") == "1"
	lockConn = os.Getenv("CHGUI_LOCK_CONNECTIONS") == "1"
	conns    []conn
	saves    *store

	// desktop replaces the network server with a native window; set by the
	// `desktop` build tag (desktop.go).
	desktop func(http.Handler)

	// One pooled client for every user and every server; credentials ride
	// per-request, so connections are never tied to an identity.
	chClient = &http.Client{Timeout: 60 * time.Second}
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// cred is one set of ClickHouse credentials, for one connection.
type cred struct {
	User string `json:"u"`
	Pass string `json:"p"`
	At   int64  `json:"t"`
}

// sess is what we sealed into the user's cookie: which server is selected and
// the credentials for the configured servers this person has signed into.
// Saved connections are not here - they live in the shared store and carry
// their own credentials.
type sess struct {
	Active string          `json:"a"`
	Creds  map[string]cred `json:"c"`
	Exp    int64           `json:"e"`
}

func (s sess) valid() bool { return s.Exp > time.Now().Unix() && len(s.Creds) > 0 }

// query runs sql over the ClickHouse HTTP interface as c. JSONCompact returns
// rows as arrays rather than repeating every column name on every row, which
// roughly halves the payload on the wide result sets this app lives on.
func query(target conn, c cred, sql string, params map[string]string) ([]byte, error) {
	q := url.Values{"default_format": {"JSONCompact"}}
	for k, v := range params {
		// ClickHouse binds {name:Type} placeholders from param_<name>. Only
		// values, never identifiers - those are quoted client-side.
		q.Set("param_"+k, v)
	}

	req, err := http.NewRequest(http.MethodPost, target.URL+"/?"+q.Encode(), strings.NewReader(sql))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)

	resp, err := chClient.Do(req)
	if err != nil {
		return nil, reachErr(target, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(chError(body))
	}
	return body, nil
}

// reachErr turns Go's raw dial error into something actionable. Leaving the
// port off is the most common mistake by far: a bare host dials 80, while
// ClickHouse serves HTTP on 8123, and "connection refused" alone does not
// point at that.
func reachErr(target conn, err error) error {
	msg := err.Error()
	u, _ := url.Parse(target.URL)
	port := ""
	if u != nil {
		port = u.Port()
	}

	switch {
	case strings.Contains(msg, "connection refused") && (port == "" || port == "80" || port == "443"):
		return fmt.Errorf("%s refused the connection on port %s. ClickHouse serves HTTP on 8123 by default - check the port",
			target.Name, cmp(port, "80"))
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("%s refused the connection on port %s. Check that ClickHouse is listening there",
			target.Name, port)
	case strings.Contains(msg, "no such host"):
		return fmt.Errorf("%s: no such host. Check the hostname", target.Name)
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "i/o timeout"):
		return fmt.Errorf("%s did not respond in time. Check network access and any firewall", target.Name)
	case strings.Contains(msg, "tls:"), strings.Contains(msg, "certificate"):
		return fmt.Errorf("%s: TLS failed. If the server speaks plain HTTP, use http instead of https", target.Name)
	}
	return fmt.Errorf("%s is unreachable: %w", target.Name, err)
}

func cmp(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// chError pulls the readable message out of a ClickHouse error response.
// Failures raised before any output arrive as plain text, but a query that
// fails once rows have started streaming returns a truncated JSON body with
// the message tucked into an "exception" field instead.
func chError(body []byte) string {
	var partial struct {
		Exception string `json:"exception"`
	}
	if json.Unmarshal(body, &partial) == nil && partial.Exception != "" {
		return partial.Exception
	}
	return strings.TrimSpace(string(body))
}

// autoCred, when set, signs every cookieless request in as that user. Only the
// desktop build sets it, and only for a server bound to loopback.
var autoCred *cred

func session(r *http.Request) (sess, bool) {
	var s sess
	if ck, err := r.Cookie(cookieName); err == nil && unseal(ck.Value, &s) == nil && s.valid() {
		return s, true
	}
	if autoCred != nil {
		return sess{
			Active: conns[0].ID,
			Creds:  map[string]cred{conns[0].ID: *autoCred},
			Exp:    time.Now().Add(sessionTTL).Unix(),
		}, true
	}
	return sess{}, false
}

// active resolves the connection and credentials the session is currently
// pointed at, rejecting anything that no longer exists.
func active(s sess) (conn, cred, bool) {
	if saves != nil {
		if c, err := saves.get(s.Active); err == nil {
			return conn{ID: c.ID, Name: c.Name, URL: c.URL},
				cred{User: c.User, Pass: c.Pass}, true
		}
	}
	c, ok := s.Creds[s.Active]
	if !ok {
		return conn{}, cred{}, false
	}
	target, ok := findConn(conns, s.Active)
	return target, c, ok
}

func setSession(w http.ResponseWriter, s sess) error {
	tok, err := seal(s)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		Expires:  time.Unix(s.Exp, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}

/* --- handlers ------------------------------------------------------------ */

// handleConnections is readable without a session because the sign-in screen
// needs the list. It exposes the URL only for servers you are already signed
// into, so an unauthenticated caller learns names, never internal hostnames.
func handleConnections(w http.ResponseWriter, r *http.Request) {
	s, signedIn := session(r)

	out := []map[string]any{}
	for _, c := range conns {
		_, has := s.Creds[c.ID]
		row := map[string]any{"id": c.ID, "name": c.Name, "saved": false, "signedIn": signedIn && has}
		if signedIn && has {
			row["url"] = c.URL
			row["user"] = s.Creds[c.ID].User
		}
		out = append(out, row)
	}
	// Saved connections carry their own credentials, so they are usable by
	// anyone who can reach this server - which is exactly what "shared" means
	// and what the UI has to say out loud.
	if saves != nil {
		list, err := saves.list()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "could not read saved connections: "+err.Error())
			return
		}
		for _, c := range list {
			out = append(out, map[string]any{
				"id": c.ID, "name": c.Name, "url": c.URL, "user": c.User,
				"saved": true, "signedIn": !c.Broken, "createdBy": c.CreatedBy,
				"broken": c.Broken,
			})
		}
	}

	writeJSON(w, map[string]any{
		"connections": out,
		"active":      s.Active,
		"canAdd":      !lockConn,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Connection string `json:"connection"`
		User       string `json:"user"`
		Password   string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed request")
		return
	}
	if in.Connection == "" {
		in.Connection = conns[0].ID
	}

	target, ok := findConn(conns, in.Connection)
	if !ok {
		writeErr(w, http.StatusBadRequest, "unknown connection")
		return
	}

	c := cred{User: in.User, Pass: in.Password, At: time.Now().Unix()}
	if _, err := query(target, c, "SELECT 1", nil); err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Add to whatever session already exists, so signing into a second server
	// does not sign you out of the first.
	s, _ := session(r)
	if s.Creds == nil {
		s.Creds = map[string]cred{}
	}
	s.Creds[target.ID] = c
	s.Active = target.ID
	s.Exp = time.Now().Add(sessionTTL).Unix()
	evictOldest(&s)

	if err := setSession(w, s); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not start session")
		return
	}
	writeJSON(w, map[string]any{"user": c.User, "connection": target.ID})
}

// evictOldest keeps the cookie bounded by dropping the least recently used
// credentials once too many servers are signed into at once.
func evictOldest(s *sess) {
	for len(s.Creds) > maxSignedIn {
		oldest, at := "", int64(0)
		for id, c := range s.Creds {
			if id == s.Active {
				continue
			}
			if oldest == "" || c.At < at {
				oldest, at = id, c.At
			}
		}
		if oldest == "" {
			return
		}
		delete(s.Creds, oldest)
	}
}

// handleAddConnection registers a ClickHouse server the user names themselves.
// The credentials are proved before anything is stored, so a connection in the
// list is always one that actually worked.
func handleAddConnection(w http.ResponseWriter, r *http.Request) {
	if lockConn || saves == nil {
		writeErr(w, http.StatusForbidden,
			"This deployment only allows connections configured on the server.")
		return
	}
	s, ok := session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "not signed in")
		return
	}

	var in struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed request")
		return
	}

	clean, err := normalizeURL(in.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := checkTarget(clean); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.TrimPrefix(strings.TrimPrefix(clean, "https://"), "http://")
	}

	// Prove the credentials before storing anything, so a connection in the
	// list is always one that actually worked.
	probe := cred{User: in.User, Pass: in.Password, At: time.Now().Unix()}
	if _, err := query(conn{Name: name, URL: clean}, probe, "SELECT 1", nil); err != nil {
		writeErr(w, http.StatusUnauthorized, err.Error())
		return
	}

	addedBy := "unknown"
	if _, c, ok := active(s); ok {
		addedBy = c.User
	}

	added, err := saves.add(saved{
		Name: name, URL: clean, User: probe.User, Pass: probe.Pass, CreatedBy: addedBy,
	})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}

	s.Active = added.ID
	s.Exp = time.Now().Add(sessionTTL).Unix()
	if err := setSession(w, s); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save the connection")
		return
	}
	writeJSON(w, map[string]any{"id": added.ID, "name": added.Name})
}

func handleRemoveConnection(w http.ResponseWriter, r *http.Request) {
	s, ok := session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "not signed in")
		return
	}
	if lockConn || saves == nil {
		writeErr(w, http.StatusForbidden, "saved connections are disabled on this deployment")
		return
	}

	var in struct {
		Connection string `json:"connection"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed request")
		return
	}

	if err := saves.remove(in.Connection); err != nil {
		if errors.Is(err, errNoSuchConn) {
			writeErr(w, http.StatusNotFound, "no such connection")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fall back to any configured server this session is still signed into.
	if s.Active == in.Connection {
		s.Active = ""
		for id := range s.Creds {
			s.Active = id
			break
		}
	}
	if !s.valid() {
		clearCookie(w)
		writeJSON(w, map[string]any{"ok": true, "connection": ""})
		return
	}
	if err := setSession(w, s); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not remove the connection")
		return
	}
	writeJSON(w, map[string]any{"ok": true, "connection": s.Active})
}

func handleSwitch(w http.ResponseWriter, r *http.Request) {
	s, ok := session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "not signed in")
		return
	}
	var in struct {
		Connection string `json:"connection"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "malformed request")
		return
	}
	isSaved := false
	if saves != nil {
		switch _, err := saves.get(in.Connection); {
		case err == nil:
			isSaved = true
		case errors.Is(err, errNoSuchConn):
			// Fall through to the credential check below.
		default:
			// A row that will not decrypt is a real, explainable failure and
			// must not be reported as "you are not signed in".
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
	}
	if _, held := s.Creds[in.Connection]; !held && !isSaved {
		writeErr(w, http.StatusForbidden, "not signed in to that connection")
		return
	}

	s.Active = in.Connection
	if err := setSession(w, s); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not switch")
		return
	}
	writeJSON(w, map[string]any{"connection": in.Connection})
}

// handleLogout drops one connection, or the whole session when none is named.
func handleLogout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Connection string `json:"connection"`
	}
	_ = decode(r, &in)

	s, ok := session(r)
	if !ok || in.Connection == "" {
		clearCookie(w)
		writeJSON(w, map[string]any{"ok": true})
		return
	}

	delete(s.Creds, in.Connection)
	if len(s.Creds) == 0 {
		clearCookie(w)
		writeJSON(w, map[string]any{"ok": true})
		return
	}
	if s.Active == in.Connection {
		for id := range s.Creds {
			s.Active = id
			break
		}
	}
	if err := setSession(w, s); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not sign out")
		return
	}
	writeJSON(w, map[string]any{"ok": true, "connection": s.Active})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	s, ok := session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "not signed in")
		return
	}
	target, c, ok := active(s)
	if !ok {
		// The active connection was removed from config since sign-in.
		clearCookie(w)
		writeErr(w, http.StatusUnauthorized, "connection no longer configured")
		return
	}
	writeJSON(w, map[string]any{
		"user":       c.User,
		"clickhouse": target.URL,
		"connection": map[string]string{"id": target.ID, "name": target.Name, "url": target.URL},
	})
}

// handleQuery is deliberately an open SQL passthrough. It is safe to expose
// because the statement runs as the signed-in ClickHouse user: this endpoint
// can do nothing the same person could not do from clickhouse-client.
func handleQuery(w http.ResponseWriter, r *http.Request) {
	s, ok := session(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "not signed in")
		return
	}
	target, c, ok := active(s)
	if !ok {
		writeErr(w, http.StatusUnauthorized, "connection no longer configured")
		return
	}

	var in struct {
		SQL    string            `json:"sql"`
		Params map[string]string `json:"params"`
	}
	if err := decode(r, &in); err != nil || strings.TrimSpace(in.SQL) == "" {
		writeErr(w, http.StatusBadRequest, "expected a sql statement")
		return
	}

	body, err := query(target, c, in.SQL, in.Params)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

/* --- plumbing ------------------------------------------------------------ */

func decode(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, maxBody)).Decode(v)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// staticHandler serves the embedded Vite build, falling back to index.html so
// client-side routes survive a page reload.
func staticHandler() http.Handler {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatal(err)
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(sub, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	})
}

func main() {
	var err error
	conns, err = loadConns(os.Getenv("CHGUI_CONNECTIONS"), env("CHGUI_CLICKHOUSE_URL", "http://localhost:8123"))
	if err != nil {
		log.Fatal(err)
	}

	if path := env("CHGUI_DB", defaultDBPath()); path != "off" && !lockConn {
		saves, err = openStore(path)
		if err != nil {
			log.Fatal(err)
		}
		defer saves.Close()
		log.Printf("saved connections in %s", path)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/connections", handleConnections)
	mux.HandleFunc("POST /api/connections/add", handleAddConnection)
	mux.HandleFunc("POST /api/connections/remove", handleRemoveConnection)
	mux.HandleFunc("POST /api/login", handleLogin)
	mux.HandleFunc("POST /api/switch", handleSwitch)
	mux.HandleFunc("POST /api/logout", handleLogout)
	mux.HandleFunc("GET /api/me", handleMe)
	mux.HandleFunc("POST /api/query", handleQuery)
	mux.Handle("/", staticHandler())

	if desktop != nil {
		desktop(mux)
		return
	}

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	for _, c := range conns {
		log.Printf("connection %-12s %s", c.ID, c.URL)
	}
	if lockConn {
		log.Print("CHGUI_LOCK_CONNECTIONS=1: users cannot add their own servers")
	}
	log.Printf("chgui listening on %s", addr)

	// A busy port is the most common startup failure and the least obvious in
	// a log, so say what to do about it.
	if err := srv.ListenAndServe(); err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			log.Fatalf("%v\n  port %s is already taken. Try: make run PORT=9000", err, strings.TrimPrefix(addr, ":"))
		}
		log.Fatal(err)
	}
}
