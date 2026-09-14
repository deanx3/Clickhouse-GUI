#!/bin/sh
# Installs chgui for the current user - no root, nothing outside $HOME.
#   ~/.local/bin/chgui-desktop    the app: native window, server on loopback
#   ~/.local/bin/chgui-app        launcher: loads the config, starts the app
#   ~/.config/chgui/chgui.env     port, ClickHouse URL, cookie secret
#   ~/.local/share/applications   desktop entry, so it shows in the app menu
# Re-running is safe: the config file and its secret are never overwritten.
#
# Usage: ./install.sh [--port N] [--build]
set -e

PORT=""
BUILD=0

usage() {
  echo "usage: ./install.sh [--port N] [--build]"
  echo "  --port N   listen on port N (default 8090, or keep the configured one)"
  echo "  --build    rebuild the UI and binary first"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --port)   PORT="$2"; shift 2 ;;
    --port=*) PORT="${1#*=}"; shift ;;
    --build)  BUILD=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [ -n "$PORT" ]; then
  # A bad port here would only surface as a confusing bind error much later.
  case "$PORT" in
    ''|*[!0-9]*) echo "--port must be a number, got '$PORT'" >&2; exit 2 ;;
  esac
  [ "$PORT" -ge 1 ] && [ "$PORT" -le 65535 ] || { echo "--port must be 1-65535" >&2; exit 2; }
fi

BIN="${XDG_BIN_HOME:-$HOME/.local/bin}"
CONF_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/chgui"
DATA="${XDG_DATA_HOME:-$HOME/.local/share}"
APPS="$DATA/applications"
ICONS="$DATA/icons/hicolor/scalable/apps"
SRC="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$BIN" "$CONF_DIR" "$APPS" "$ICONS"

# --- build ------------------------------------------------------------------
if [ ! -x "$SRC/chgui-desktop" ] || [ "$BUILD" = 1 ]; then
  if ! pkg-config --exists webkit2gtk-4.1 gtk+-3.0 2>/dev/null; then
    echo "missing build libraries. Install them with:" >&2
    echo "  sudo apt install libgtk-3-dev libwebkit2gtk-4.1-dev" >&2
    exit 1
  fi
  echo "building..."
  make -C "$SRC" desktop >/dev/null
fi
install -m 755 "$SRC/chgui-desktop" "$BIN/chgui-desktop"

# --- config: written once, so the secret survives upgrades ------------------
CONF="$CONF_DIR/chgui.env"

if [ ! -f "$CONF" ]; then
  # A stable secret means restarts do not sign you out.
  SECRET=$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')
  cat > "$CONF" <<EOF
# chgui settings. Restart the app after editing.
# 8080 is commonly taken by other services, so default off it.
CHGUI_ADDR=:${PORT:-8090}
CHGUI_CLICKHOUSE_URL=http://localhost:8123

# Stable cookie key. Changing it signs everyone out. Keep it private.
CHGUI_SECRET=$SECRET

# ClickHouse account the app signs in as, so there is no login page.
# Leave empty to be asked on the login page instead.
CHGUI_USER=
CHGUI_PASSWORD=

# Pre-seed shared servers (inline JSON or a file path); users can add their own
# from the header menu unless CHGUI_LOCK_CONNECTIONS=1.
#CHGUI_CONNECTIONS=$CONF_DIR/connections.json
EOF
  chmod 600 "$CONF"
  echo "wrote $CONF"
elif [ -n "$PORT" ]; then
  # Rewrite only the port line, so the existing secret is untouched.
  tmp="$CONF.tmp.$$"
  sed "s|^CHGUI_ADDR=.*|CHGUI_ADDR=:$PORT|" "$CONF" > "$tmp"
  grep -q '^CHGUI_ADDR=' "$CONF" || echo "CHGUI_ADDR=:$PORT" >> "$tmp"
  mv "$tmp" "$CONF"
  chmod 600 "$CONF"
  echo "set   port $PORT in $CONF"
else
  echo "kept  $CONF (existing port and secret preserved)"
fi

# Configs written before auto sign-in existed: add the keys so they are findable.
if ! grep -q '^CHGUI_USER=' "$CONF"; then
  printf '\n# ClickHouse account the app signs in as, so there is no login page.\nCHGUI_USER=\nCHGUI_PASSWORD=\n' >> "$CONF"
  echo "added CHGUI_USER / CHGUI_PASSWORD to $CONF"
fi

LISTEN_PORT=$(sed -n 's|^CHGUI_ADDR=:\{0,1\}\(.*\)|\1|p' "$CONF" | tail -1)
LISTEN_PORT="${LISTEN_PORT:-8090}"

# Warn now rather than at first launch, when the error is buried in a log.
if command -v ss >/dev/null 2>&1 && ss -ltn 2>/dev/null | grep -q ":$LISTEN_PORT "; then
  echo
  echo "warning: something is already listening on port $LISTEN_PORT."
  echo "         pick another with: ./install.sh --port 9000"
fi

# --- icon -------------------------------------------------------------------
cat > "$ICONS/chgui.svg" <<'EOF'
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" width="64" height="64">
  <rect width="64" height="64" rx="14" fill="#12161f"/>
  <g fill="#f0883e">
    <rect x="12" y="18" width="7" height="28" rx="2" opacity=".45"/>
    <rect x="23" y="13" width="7" height="38" rx="2"/>
    <rect x="34" y="22" width="7" height="20" rx="2" opacity=".7"/>
    <rect x="45" y="17" width="7" height="30" rx="2" opacity=".3"/>
  </g>
</svg>
EOF

# --- launcher ---------------------------------------------------------------
cat > "$BIN/chgui-app" <<'EOF'
#!/bin/sh
# Loads the chgui config and opens the app window. Closing it stops the app.
CONF="${XDG_CONFIG_HOME:-$HOME/.config}/chgui/chgui.env"
LOG="${XDG_STATE_HOME:-$HOME/.local/state}/chgui.log"
mkdir -p "$(dirname "$LOG")"

set -a
[ -f "$CONF" ] && . "$CONF"
set +a

exec "$(dirname "$0")/chgui-desktop" >>"$LOG" 2>&1
EOF
chmod 755 "$BIN/chgui-app"

# --- desktop entry ----------------------------------------------------------
cat > "$APPS/chgui.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=ClickHouse Console
GenericName=Database Client
Comment=Explore schemas, run SQL, and manage ClickHouse access
Exec=$BIN/chgui-app
Icon=chgui
Terminal=false
Categories=Development;Database;
Keywords=clickhouse;sql;database;
StartupWMClass=chgui-desktop
EOF

command -v update-desktop-database >/dev/null && update-desktop-database "$APPS" 2>/dev/null || true
command -v gtk-update-icon-cache  >/dev/null && gtk-update-icon-cache -qtf "$DATA/icons/hicolor" 2>/dev/null || true

echo
echo "installed:"
echo "  $BIN/chgui-desktop"
echo "  $BIN/chgui-app"
echo "  $APPS/chgui.desktop"
echo
echo 'Launch from your app menu ("ClickHouse Console"), or run: chgui-app'
