# Clickhouse-GUI

A ClickHouse console: browse schemas, run SQL, manage users, roles and grants.
You sign in with your ClickHouse account, and ClickHouse decides what you can do.

## Linux app

Needs Go and Node.js.

```sh
sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev
./install.sh --build
```

Open **ClickHouse Console** from your app menu, or run `chgui-app`.

To update, run `git pull && ./install.sh --build`, then reopen the app.

Settings are in `~/.config/chgui/chgui.env`. Set `CHGUI_USER` and
`CHGUI_PASSWORD` there to skip the login page. Logs are in
`~/.local/state/chgui.log`.

## Browser

```sh
make up    # ClickHouse in Docker, with demo users
make run   # serve on http://localhost:8090
```

Sign in as `admin` / `admin`. There are also the read-only `analyst` / `analyst`
and the read/insert `etl` / `etl`.

## Settings

| Variable               | Meaning                                          |
| ---------------------- | ------------------------------------------------ |
| `CHGUI_ADDR`           | Listen address (`:8080`)                         |
| `CHGUI_CLICKHOUSE_URL` | Default server (`http://localhost:8123`)         |
| `CHGUI_CONNECTIONS`    | JSON or a file listing servers                   |
| `CHGUI_SECRET`         | 64 hex chars. Without it, restarts sign you out  |
| `CHGUI_SECURE_COOKIE`  | `1` when served over HTTPS                       |

Passwords are encrypted, never stored in plain text. Use HTTPS over a network.

## Development

```sh
make api   # Go server
make web   # Vite dev server on :5173
make test  # Go + JS tests
```

MIT licence.
