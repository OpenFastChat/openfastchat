<!--
SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
SPDX-License-Identifier: AGPL-3.0-only
-->

# OpenFastChat

Self-hostable, **bring-your-own-key** AI chat where the conversation itself is an
editable, shareable flow — not a separate app you have to build first. AGPL-3.0-only.

> **Don't want to self-host?** A managed cloud version is available at
> **https://openfast.chat**

Bring your own provider key (Groq, DeepSeek, or any OpenAI-compatible / self-hosted
endpoint), run the binary, and chat. The flow behind each answer is fully editable
and shareable.

## Architecture

This is the **self-host wrapper**: a thin app over the open core. It shares the
engine and UI with the managed service; the two are siblings over core, not
layered on each other.

- **Engine + UI** — [`openfastchat-core`](https://github.com/1-Bart-1/openfastchat-core):
  the Go flow engine (`github.com/1-Bart-1/openfastchat-core`) and the React
  building blocks (`@openfastchat/ui`).
- **This repo** — basic auth, thread/flow/provider storage, a BYOK provider
  config, an **unbilled** LLM client, and a thin frontend shell that composes
  `@openfastchat/ui`. No Stripe, no multi-tenant abuse controls.
- **Managed service** — a separate private wrapper adds billing, rate/abuse
  limits, and the marketplace business on top of the same core.

## Quick start

Needs Docker (for Postgres+pgvector), Go, and pnpm. Then:

```bash
cp .env.example .env      # set OFC_GROQ_API_KEY (your provider key)
make run                  # start Postgres, build the UI + binary, serve :8080
```

Open http://127.0.0.1:8080. `make run` builds the frontend into `frontend/dist`,
which the binary serves alongside the API — no separate frontend server.

To install the binary onto your PATH (AUR-ready):

```bash
make install              # → /usr/local/bin/openfastchat (override with PREFIX=)
```

Other targets: `make dev` (Vite dev server + live backend), `make db-up`
(Postgres only), `make build` (compile without running), `make uninstall`.

## Local development

The frontend consumes the shared UI from the core repo via pnpm. During local
staging (both repos checked out side by side) it resolves to the working copy:

```jsonc
// frontend/package.json
"@openfastchat/ui": "link:../../openfastchat-core/frontend"
```

It swaps to a published version once the repos are open-sourced — no code churn
at release time.

## License

AGPL-3.0-only. See [`LICENSES/`](./LICENSES). REUSE-compliant; run `reuse lint`.
