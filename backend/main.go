// SPDX-FileCopyrightText: 2026 Bart van de Lint <bart@vandelint.net>
// SPDX-License-Identifier: AGPL-3.0-only

// Command openfastchat is the self-host server: it wires the shared
// openfastchat-core handlers with a no-op billing seam and bring-your-own-key
// providers, serving the API and the built frontend from one binary.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/1-Bart-1/openfastchat/backend/nobilling"

	"github.com/1-Bart-1/openfastchat-core/coreconfig"
	corehandler "github.com/1-Bart-1/openfastchat-core/handler"
	"github.com/1-Bart-1/openfastchat-core/middleware"
	"github.com/1-Bart-1/openfastchat-core/store"

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
	jwtSecret := env("OFC_JWT_SECRET", env("JWT_SECRET", "openfastchat-self-host-dev-secret"))

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

	coreconfig.SetActiveProviders(envProviders())

	storeQ := store.New(pool)
	hcfg := coreconfig.HandlerConfig{
		BraveAPIKey:        os.Getenv("OFC_BRAVE_API_KEY"),
		CompactModel:       env("OFC_COMPACT_MODEL", "llama-3.1-8b-instant"),
		TrustForwardedIP:   os.Getenv("OFC_TRUST_FORWARDED_IP") == "true",
		MaxUploadSizeBytes: coreconfig.DefaultMaxUploadSizeBytes,
	}
	billing := nobilling.New()

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

	r.Group(func(r chi.Router) {
		r.Use(middleware.OptionalUser(jwtSecret))
		r.Get("/api/me", meHandler(storeQ))
		r.Get("/api/flows", flowsHandler.List)
		r.Get("/api/marketplace/flows", marketplaceHandler.List)
		r.Get("/api/marketplace/search", marketplaceHandler.Search)
		r.Get("/api/marketplace/flows/{id}", marketplaceHandler.Get)
	})

	r.Group(func(r chi.Router) {
		// Open self-host: anyone gets a user on first write, no captcha/caps.
		r.Use(middleware.EnsureUser(jwtSecret, storeQ, corehandler.ProvisionGuestUser, nil, nil))
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

	r.Handle("/*", spaHandler(env("OFC_FRONTEND_DIR", "frontend/dist")))

	addr := env("LISTEN_ADDR", "127.0.0.1:8080")
	slog.Info("openfastchat self-host starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

// meHandler returns the self-host identity for the current session, or an
// anonymous stub. No billing, subscription, or admin fields.
func meHandler(storeQ *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr, email, ok := middleware.UserFromContext(r.Context())
		if !ok {
			corehandler.WriteJSON(w, http.StatusOK, map[string]any{
				"is_anonymous": true, "is_guest": true,
			})
			return
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			corehandler.WriteJSON(w, http.StatusOK, map[string]any{
				"is_anonymous": true, "is_guest": true,
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
		})
	}
}

// spaHandler serves static files from dir, falling back to index.html so the
// client-side router owns unknown paths.
func spaHandler(dir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	return func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envProviders builds the active provider list from bring-your-own-key env
// vars: Groq by default, plus DeepSeek when its key is present.
func envProviders() []coreconfig.ProviderConfig {
	provs := []coreconfig.ProviderConfig{{
		Name:    "groq",
		BaseURL: env("OFC_GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		APIKey:  env("OFC_GROQ_API_KEY", os.Getenv("GROQ_API_KEY")),
	}}
	if key := os.Getenv("OFC_DEEPSEEK_API_KEY"); key != "" {
		provs = append(provs, coreconfig.ProviderConfig{
			Name:    "deepseek",
			BaseURL: env("OFC_DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1"),
			APIKey:  key,
		})
	}
	return provs
}
