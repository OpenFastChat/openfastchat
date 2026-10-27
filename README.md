<!--
SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
SPDX-License-Identifier: AGPL-3.0-only
-->

# OpenFastChat

Self-hostable, **bring-your-own-key** AI chat where the conversation itself is
an editable, shareable flow — not a separate app you have to build first.
AGPL-3.0-only.

> **Don't want to self-host?** A managed version runs at
> **https://openfast.chat**.

Point it at any OpenAI-compatible endpoint — Groq, DeepSeek, OpenRouter, or your
own Ollama / vLLM server — and chat. The flow behind each answer is fully
editable and shareable.

## Quick start (Docker)

Needs Docker with Compose.

```bash
git clone https://github.com/OpenFastChat/openfastchat.git
cd openfastchat
docker compose up -d
```

Open http://127.0.0.1:8080, go to `/admin`, add a provider (base URL + key) and
start chatting:

| Provider   | Base URL                               |
| ---------- | -------------------------------------- |
| Groq       | `https://api.groq.com/openai/v1`       |
| OpenRouter | `https://openrouter.ai/api/v1`         |
| DeepSeek   | `https://api.deepseek.com/v1`          |
| Ollama     | `http://host.docker.internal:11434/v1` |

Provider keys are stored encrypted in Postgres. The encryption key and session
secret are generated on first start and kept in the `ofc_data` volume — back up
both `ofc_data` and `ofc_pgdata`. Optional settings (web search, compaction
model, …) go in a `.env` file next to `docker-compose.yml`; see
[`.env.example`](.env.example).

Update with `git pull && docker compose up -d --build`.

## Exposing it beyond localhost

Self-host is **single-user**: whoever reaches the app is its admin and spends
your provider credit. Compose therefore binds to `127.0.0.1` only. To reach it
from elsewhere, put it behind a reverse proxy that does TLS **and**
authentication, e.g. Caddy:

```caddy
chat.example.com {
    basic_auth {
        you <hash from `caddy hash-password`>
    }
    reverse_proxy 127.0.0.1:8080
}
```

and set `OFC_BASE_URL=https://chat.example.com` and
`OFC_TRUST_FORWARDED_IP=true` in `.env`. A VPN such as Tailscale works just as
well.

## Without Docker

Needs Go, pnpm, and Postgres with pgvector (`docker compose up -d db` provides
one).

```bash
cp .env.example .env
make run                  # build the UI + binary, then serve :8080
make install              # → /usr/local/bin/openfastchat (override with PREFIX=)
```

The UI is embedded in the binary, so `openfastchat` runs from anywhere with only
`OFC_DATABASE_URL` set. Generated secrets go to `~/.config/openfastchat`
(override with `OFC_DATA_DIR`). `make dev` runs the Vite dev server against a
live backend.

## Architecture

A thin wrapper over the open core:

- **Engine + UI** —
  [`openfastchat-core`](https://github.com/OpenFastChat/openfastchat-core): the
  Go flow engine (`github.com/openfastchat/openfastchat-core`) and the React
  building blocks
  ([`@openfastchat/ui`](https://www.npmjs.com/package/@openfastchat/ui)).
- **This repo** — single-user auth, an **unbilled** LLM client, the server
  binary, and a frontend shell over `@openfastchat/ui`. No billing, no
  multi-tenant abuse controls.
- **openfast.chat** — a separate managed wrapper over the same core that adds
  billing and abuse limits.

### Developing against a local core checkout

With `openfastchat-core` cloned next to this repo:

```bash
go work init . ../openfastchat-core                                 # go.work is gitignored
pnpm -C frontend add @openfastchat/ui@link:../../openfastchat-core/frontend  # don't commit
```

## License

AGPL-3.0-only — see [`LICENSE`](LICENSE). REUSE-compliant; run `reuse lint`.
