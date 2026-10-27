<!--
SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
SPDX-License-Identifier: AGPL-3.0-only
-->

# AGENTS.md

Self-host, bring-your-own-key wrapper (AGPL). Part of the OpenFastChat
workspace — see `../AGENTS.md` for the three-repo layout and cross-repo wiring.

- Thin app over `openfastchat-core`: basic auth, storage, BYOK providers, an
  unbilled LLM client (`backend/nobilling/`), a shell over `@openfastchat/ui`.
- `backend/main.go` serves the API + the UI embedded from `backend/web/dist`
  (the Vite build output) as one binary; `make run` or `docker compose up -d`.
- Depends on published core (`go.mod`) and `@openfastchat/ui` (npm). The UI
  ships as source, so `vite.config.ts` dedupes singletons and `src/index.css`
  has `@source` for it. Don't drop them.
