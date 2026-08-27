# Port for the Go server. 8080 is often already taken, so default off it.
PORT ?= 8090
API  := http://localhost:$(PORT)

# Connections: users add their own from the header menu, kept in their session.
# CHGUI_CONNECTIONS   pre-seed shared servers (inline JSON or a path to a file)
# CHGUI_LOCK_CONNECTIONS=1  shared deployments: only the configured servers
#
## install- install as a desktop app for the current user
install: ; ./install.sh --build $(if $(PORT),--port $(PORT))

## up     - start ClickHouse (once per boot)
up:    ; docker compose up -d
down:  ; docker compose down

## run    - build everything, serve UI + API from one binary on $(PORT)
run: build
	CHGUI_ADDR=:$(PORT) ./chgui

## api    - Go server with hot reload off, for dev alongside `make web`
api:   ; CHGUI_ADDR=:$(PORT) go run .

## web    - Vite dev server on :5173, proxying /api to $(API)
web:   ; CHGUI_API=$(API) npm --prefix web run dev

## test   - Go and JS unit tests
test:  ; go test ./... && npm --prefix web test

## build  - one binary with the UI embedded
build: ; npm --prefix web run build && go build -ldflags="-s -w" -o chgui .

## help   - list targets
help:  ; @grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  make /'

.PHONY: up down run api web test build help install
