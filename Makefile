# SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
# SPDX-License-Identifier: AGPL-3.0-only

BIN     ?= openfastchat
PREFIX  ?= /usr/local
DESTDIR ?=

.PHONY: build frontend backend run dev db-up install uninstall

build: frontend backend        ## build the UI then the server binary

frontend:                      ## build the React UI into frontend/dist
	pnpm -C frontend install
	pnpm -C frontend build

backend:                       ## build the Go server (serves API + frontend/dist)
	CGO_ENABLED=0 go build -o bin/$(BIN) ./backend

run: build db-up               ## build everything + boot on localhost (needs Postgres)
	./bin/$(BIN)

dev: db-up                     ## Vite dev server (proxies /api) + live backend
	pnpm -C frontend dev

db-up:                         ## start the Postgres+pgvector container
	docker compose up -d db

install: build                 ## install the binary to $(DESTDIR)$(PREFIX)/bin (AUR-ready)
	install -Dm755 bin/$(BIN) $(DESTDIR)$(PREFIX)/bin/$(BIN)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BIN)
