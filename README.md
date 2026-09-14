# Clickhouse-GUI

A ClickHouse console: browse schemas, run SQL, and manage users, roles and grants.

Ships as **one binary** with the web UI embedded. No runtime dependencies, no
database of its own, and no user store.

## The idea

You sign in with your **ClickHouse** credentials, and every statement runs as
you. ClickHouse enforces what you may see and change, so the app has no
permission model of its own to get wrong. The account you administer is the
account you authenticate with.

Two consequences worth knowing up front:

- There is no "chgui admin". If ClickHouse refuses you, so does this.
- The grants screens can only manage SQL-defined users. Accounts declared in
  `users.xml` show up read-only, because ClickHouse itself cannot alter them.

## Quick start

```sh
make up      # start ClickHouse in Docker
make run     # build the UI, embed it, serve on :8090
```

Open <http://localhost:8090> and sign in as `admin` / `admin`.

Seed accounts from `clickhouse/init/01-seed.sql` — all local dev fixtures:

| user      | password  | shows off                                            |
| --------- | --------- | ---------------------------------------------------- |
| `admin`   | `admin`   | full access; the RBAC screens are editable            |
| `analyst` | `analyst` | read-only via role `reader`, plus a row policy        |
| `etl`     | `etl`     | read + insert via role `writer`                       |

Sign in as `analyst` to watch ClickHouse reject things the UI happily offers.

## Install as a desktop app

A native Linux app: a GTK/WebKit window with the server running inside it, on
`127.0.0.1` only. No browser needed, and closing the window quits the app.

```sh
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev   # build libraries, once
./install.sh --build --port 8090
```

Installs to `~/.local` and adds an app-menu entry ("ClickHouse Console"). No
root beyond the libraries, nothing outside `$HOME`.

To skip the login page, set your ClickHouse account in
`~/.config/chgui/chgui.env` and reopen the app:

```sh
CHGUI_USER=admin
CHGUI_PASSWORD=admin
```

The app checks them at startup and falls back to the login page if they fail.
Logs go to `~/.local/state/chgui.log`. `make desktop` builds the app
(`chgui-desktop`) without installing it.

Known gap: CSV/JSON export may not save a file from inside the app window.

## Features

- **Explorer** — schema tree, CodeMirror SQL editor with schema-aware
  autocomplete, results grid with sorting, resizing, CSV/JSON export
- **Users / Roles** — a grants matrix that shows *why* a privilege applies:
  granted directly, inherited through a role, or inherited from a broader scope
- **SQL ledger** — every change is staged as the exact statement that will run,
  and nothing executes until you apply it
- **Policies** — row policies, quotas and settings profiles (read-only)
- **Query log** — filtered to access changes by default
- **Command palette** — <kbd>Ctrl/⌘</kbd>+<kbd>K</kbd>

## Configuration

| Variable                  | Default                        | Meaning                                        |
| ------------------------- | ------------------------------ | ---------------------------------------------- |
| `CHGUI_ADDR`              | `:8080`                        | Listen address                                 |
| `CHGUI_CLICKHOUSE_URL`    | `http://localhost:8123`        | Single-server fallback                         |
| `CHGUI_CONNECTIONS`       | —                              | Inline JSON or a file listing servers          |
| `CHGUI_LOCK_CONNECTIONS`  | —                              | `1` = only configured servers, no saving       |
| `CHGUI_DB`                | `$XDG_DATA_HOME/chgui/chgui.db`| Saved connections; `off` disables              |
| `CHGUI_SECRET`            | random                         | 64 hex chars. Unset means restarts sign everyone out |
| `CHGUI_SECURE_COOKIE`     | —                              | `1` when served over HTTPS                     |
| `CHGUI_USER` / `CHGUI_PASSWORD` | —                        | Desktop app only: sign in automatically        |

```jsonc
// CHGUI_CONNECTIONS
[
  { "id": "prod",  "name": "Production", "url": "http://ch-prod:8123" },
  { "id": "stage", "name": "Staging",    "url": "http://ch-stage:8123" }
]
```

## Where credentials live

There is no local user database and nothing is hashed, because a ClickHouse
password has to be **replayed** on every query — a hash could never be sent
back. So:

- **Your session** lives in one `httpOnly` cookie, AES-256-GCM encrypted.
- **Saved connections** live in SQLite with the password encrypted at rest.
  They are **shared**: anyone who can reach the deployment can use them.
- The cookie key and the store key are derived separately from `CHGUI_SECRET`,
  so a stolen cookie cannot decrypt the store.

Set `CHGUI_SECRET` in production, or every restart invalidates both. Over a
network, terminate TLS and set `CHGUI_SECURE_COOKIE=1` — ClickHouse HTTP auth
is Basic auth, so the password is only as protected as the transport.

## Development

```sh
make api     # Go server
make web     # Vite dev server on :5173, proxying /api
make test    # Go + JS unit tests
make desktop # native app build (-tags desktop, needs the GTK/WebKit libraries)
```

`pkgconfig/` aliases `webkit2gtk-4.0` to `4.1`: the webview library asks for
4.0, and newer distros only ship 4.1.

Go server is stdlib only apart from the SQLite driver, plus webview_go in the
desktop build; the UI is Vue 3 talking to ClickHouse's HTTP interface via
`FORMAT JSONCompact`.

## Licence

MIT
