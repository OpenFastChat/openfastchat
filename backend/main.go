// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

// Command openfastchat is the self-host server: it wires the shared
// openfastchat-core handlers with a no-op billing seam and bring-your-own-key
// providers, serving the API and the built frontend from one binary.
package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/openfastchat/openfastchat/backend/nobilling"
	"github.com/openfastchat/openfastchat/backend/web"

	"github.com/openfastchat/openfastchat-core/coreconfig"
	"github.com/openfastchat/openfastchat-core/crypto"
	corehandler "github.com/openfastchat/openfastchat-core/handler"
	"github.com/openfastchat/openfastchat-core/middleware"
	"github.com/openfastchat/openfastchat-core/store"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env", "../.env")
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dbURL := env("OFC_DATABASE_URL", os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		slog.Error("OFC_DATABASE_URL (or DATABASE_URL) is required")
		os.Exit(1)
	}
	dataDir := env("OFC_DATA_DIR", defaultDataDir())
	jwtSecret, err := loadSecret(dataDir, "jwt_secret", "OFC_JWT_SECRET", "JWT_SECRET")
	if err != nil {
		slog.Error("jwt secret", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := store.Migrate(pool); err != nil {
		slog.Error("migrations", "err", err)
		os.Exit(1)
	}

	// Plain-HTTP localhost: browsers drop Secure cookies, so disable the flag
	// unless the operator terminates TLS in front.
	middleware.SetCookieSecure(strings.HasPrefix(env("OFC_BASE_URL", ""), "https://"))

	storeQ := store.New(pool)
	hcfg := coreconfig.HandlerConfig{
		BraveAPIKey:        os.Getenv("OFC_BRAVE_API_KEY"),
		CompactModel:       env("OFC_COMPACT_MODEL", "openai/gpt-oss-20b"),
		TrustForwardedIP:   os.Getenv("OFC_TRUST_FORWARDED_IP") == "true",
		MaxUploadSizeBytes: coreconfig.DefaultMaxUploadSizeBytes,
	}
	billing := nobilling.New()

	// Providers live encrypted in the DB and are managed at runtime via
	// /admin — nothing comes from the environment but the master key.
	encryptionKey, err := loadSecret(
		dataDir, "provider_encryption_key", "OFC_PROVIDER_ENCRYPTION_KEY",
	)
	if err != nil {
		slog.Error("provider encryption key", "err", err)
		os.Exit(1)
	}
	box, err := crypto.New(encryptionKey)
	if err != nil {
		slog.Error("provider encryption key invalid", "err", err)
		os.Exit(1)
	}
	providerManager := corehandler.NewProviderManager(storeQ, box, nil)
	if err := providerManager.Reload(context.Background()); err != nil {
		slog.Warn("provider reload at startup failed", "err", err)
	}
	providerManager.StartModelRefresh(context.Background())

	providerAdmin := corehandler.NewProviderAdmin(storeQ, box, providerManager, nil)

	chatHandler := corehandler.NewChatHandler(storeQ, pool, hcfg)
	uploadHandler := corehandler.NewUploadHandler(storeQ)
	docsHandler := corehandler.NewDocumentsHandler(storeQ)
	flowsHandler := corehandler.NewFlowsHandler(storeQ, pool, hcfg, billing)
	accountHandler := corehandler.NewAccountHandler(storeQ, corehandler.AccountConfig{})
	marketplaceHandler := corehandler.NewMarketplaceHandler(storeQ)
	issuesHandler := corehandler.NewIssuesHandler(storeQ)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.ErrorLog(storeQ))

	r.Get("/api/health", corehandler.NewHealthHandler(pool))
	r.Get("/api/models", corehandler.ModelsHandler)
	r.Get("/api/welcome", corehandler.WelcomeMessages)

	// Admin: DB-backed provider config. Self-host is single-user and that
	// user is the admin by definition, so there is no gate — the operator
	// keeps a public instance behind their own network/proxy auth, exactly
	// as they must to protect the provider keys and API credit themselves.
	r.Get("/api/admin/providers", providerAdmin.List)
	r.Post("/api/admin/providers", providerAdmin.Create)
	r.Put("/api/admin/providers/{id}", providerAdmin.Update)
	r.Delete("/api/admin/providers/{id}", providerAdmin.Delete)

	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalUser(jwtSecret))
		r.Get("/api/me", meHandler(storeQ))
		r.Get("/api/flows", flowsHandler.List)
		r.Get("/api/marketplace/flows", marketplaceHandler.List)
		r.Get("/api/marketplace/search", marketplaceHandler.Search)
		r.Get("/api/marketplace/flows/{id}", marketplaceHandler.Get)
	})

	r.Group(func(r chi.Router) {
		// Single-user self-host: every session resolves to the one owner
		// account (created on first write), no captcha/caps.
		r.Use(middleware.EnsureUser(jwtSecret, storeQ, provisionSingleUser, nil, nil))
		r.Post("/api/issues", issuesHandler.Report)
		r.Post("/api/auth/logout", accountHandler.Logout)
		r.Post("/api/auth/delete-account", accountHandler.DeleteAccount)
		r.Get("/api/preferred-model", accountHandler.GetPreferredModel)
		r.Put("/api/preferred-model", accountHandler.UpdatePreferredModel)
		r.Get("/api/node-menu", accountHandler.GetNodeMenu)
		r.Put("/api/node-menu", accountHandler.UpdateNodeMenu)
		r.Patch("/api/me/display-name", accountHandler.UpdateDisplayName)
		r.Get("/api/threads", chatHandler.ListThreads)
		r.Post("/api/threads", chatHandler.CreateThread)
		r.Delete("/api/threads/{id}", chatHandler.DeleteThread)
		r.Patch("/api/threads/{id}", chatHandler.RenameThread)
		r.Put("/api/threads/{id}/scroll", chatHandler.UpdateScroll)
		r.Get("/api/threads/{id}/messages", chatHandler.Messages)
		r.Delete("/api/threads/{id}/messages", chatHandler.ClearMessages)
		r.Post("/api/upload", uploadHandler.Upload)
		r.Get("/api/documents", docsHandler.List)
		r.Delete("/api/documents/{id}", docsHandler.Delete)
		r.Post("/api/flows", flowsHandler.Create)
		r.Get("/api/flows/{id}", flowsHandler.Get)
		r.Put("/api/flows/{id}", flowsHandler.Update)
		r.Delete("/api/flows/{id}", flowsHandler.Delete)
		r.Put("/api/flows/{id}/activate", flowsHandler.Activate)
		r.Post("/api/flows/{id}/reset", flowsHandler.Reset)
		r.Post("/api/flows/{id}/publish", flowsHandler.Publish)
		r.Delete("/api/flows/{id}/publish", flowsHandler.Unpublish)
		r.Post("/api/flows/execute", flowsHandler.Execute)
		r.Put("/api/marketplace/flows/{id}/vote", marketplaceHandler.Vote)
	})

	r.Handle("/*", spaHandler(web.Dist()))

	addr := env("LISTEN_ADDR", "127.0.0.1:8080")
	slog.Info("openfastchat self-host starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

// meHandler returns the self-host identity for the current session, or an
// anonymous stub. Self-host is single-user, so the session is always the
// admin — is_admin is unconditionally true so the frontend reveals /admin.
func meHandler(storeQ *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr, email, ok := middleware.UserFromContext(r.Context())
		if !ok {
			corehandler.WriteJSON(w, http.StatusOK, map[string]any{
				"is_anonymous": true, "is_guest": true, "is_admin": true,
			})
			return
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			corehandler.WriteJSON(w, http.StatusOK, map[string]any{
				"is_anonymous": true, "is_guest": true, "is_admin": true,
			})
			return
		}
		user, err := storeQ.GetUserByID(r.Context(), uid)
		if err != nil {
			corehandler.WriteServerError(w, "get user", err)
			return
		}
		corehandler.WriteJSON(w, http.StatusOK, map[string]any{
			"id":              uid.String(),
			"email":           email,
			"is_anonymous":    false,
			"is_guest":        user.IsGuest,
			"display_name":    user.DisplayName,
			"default_flow_id": corehandler.UUIDPtr(user.DefaultFlowID),
			"is_template":     false,
			"is_admin":        true,
		})
	}
}

// spaHandler serves the built frontend, falling back to index.html so the
// client-side router owns unknown paths.
func spaHandler(dist fs.FS) http.HandlerFunc {
	files := http.FileServerFS(dist)
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFileFS(w, r, dist, "index.html")
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
