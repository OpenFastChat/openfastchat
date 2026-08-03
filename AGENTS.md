<!--
SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
SPDX-License-Identifier: AGPL-3.0-only
-->

# AGENTS.md

Self-host, bring-your-own-key wrapper (AGPL). Part of the OpenFastChat
workspace — see `../AGENTS.md` for the three-repo layout and cross-repo wiring.

- Thin app over `openfastchat-core`: basic auth, storage, BYOK providers, an
  unbilled LLM client (`backend/nobilling/`), a shell over `@openfastchat/ui`.
- `backend/main.go` serves the API + `frontend/dist` from one binary; `make run`.
- Build gotcha: `@openfastchat/ui` is linked source, so `vite.config.ts` dedupes
  singletons and `src/index.css` has `@source` for the core UI. Don't drop them.
