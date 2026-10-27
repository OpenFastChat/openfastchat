# SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
# SPDX-License-Identifier: AGPL-3.0-only

FROM node:22-alpine AS frontend
WORKDIR /src/frontend
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY backend/ backend/
COPY --from=frontend /src/backend/web/dist backend/web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /openfastchat ./backend

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /openfastchat /openfastchat
ENV LISTEN_ADDR=0.0.0.0:8080 OFC_DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/openfastchat"]
