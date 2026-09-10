package api

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"thingspan/internal/config"
	"thingspan/internal/security"
	"thingspan/internal/services"
)

func NewRouter(
	cfg *config.Config,
	db *sql.DB,
	pm *security.PasswordManager,
	jwtm *security.JWTManager,
	rl *security.RateLimiter,
	scanner *services.ReminderScanner,
) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Recoverer)
	r.Use(SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	healthHandler := NewHealthHandler(db)
	authHandler := NewAuthHandler(pm, jwtm, rl)
	categoriesHandler := NewCategoriesHandler(db, cfg)
	assetsHandler := NewAssetsHandler(db, cfg)
	dashboardHandler := NewDashboardHandler(db, cfg)
	iconsHandler := NewIconsHandler()
	remindersHandler := NewRemindersHandler(db, scanner)

	// Health probes
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/api/health", healthHandler.APIHealth)

	// Auth routes (public)
	r.Route("/api/auth", func(sub chi.Router) {
		sub.Post("/login", authHandler.Login)
		sub.Post("/refresh", authHandler.Refresh)
	})

	// Protected API routes
	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(RequireAuth(jwtm))

		apiRouter.Get("/icons", iconsHandler.List)

		apiRouter.Route("/categories", func(sub chi.Router) {
			sub.Get("/", categoriesHandler.List)
			sub.Post("/", categoriesHandler.Create)
			sub.Put("/{id}", categoriesHandler.Update)
			sub.Delete("/{id}", categoriesHandler.Delete)
		})

		apiRouter.Route("/assets", func(sub chi.Router) {
			sub.Get("/", assetsHandler.List)
			sub.Post("/", assetsHandler.Create)
			sub.Get("/{id}", assetsHandler.Get)
			sub.Put("/{id}", assetsHandler.Update)
			sub.Delete("/{id}", assetsHandler.Delete)
		})

		apiRouter.Get("/dashboard", dashboardHandler.Get)

		apiRouter.Route("/reminders", func(sub chi.Router) {
			sub.Get("/", remindersHandler.List)
			sub.Post("/{id}/dismiss", remindersHandler.Dismiss)
		})

		// Any unhandled /api/* request must return 404 JSON
		apiRouter.NotFound(func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, http.StatusNotFound, "Not Found")
		})
	})

	// Static & SPA fallback
	staticDir := cfg.StaticDir
	// Check possible fallback locations if default staticDir not found
	if _, err := os.Stat(filepath.Join(staticDir, "index.html")); os.IsNotExist(err) {
		candidates := []string{
			"./static",
			"./app/static",
			"../frontend/dist",
			"../../frontend/dist",
		}
		for _, c := range candidates {
			if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
				staticDir = c
				break
			}
		}
	}

	indexFile := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexFile); err == nil {
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			// API routes must never serve index.html
			if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
				WriteError(w, http.StatusNotFound, "Not Found")
				return
			}

			cleanPath := strings.TrimPrefix(r.URL.Path, "/")
			candidate := filepath.Join(staticDir, cleanPath)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}

			http.ServeFile(w, r, indexFile)
		})
	}

	return r
}

