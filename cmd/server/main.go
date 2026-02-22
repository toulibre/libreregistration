package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/handlers"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/services"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()

	// Open database
	driver := cfg.DatabaseDriver
	dsn := cfg.DatabasePath
	if driver == "postgres" {
		driver = "pgx"
		dsn = cfg.DatabaseURL
	}

	db, err := database.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize stores
	userStore := database.NewUserStore(db)
	eventStore := database.NewEventStore(db)
	registrationStore := database.NewRegistrationStore(db)
	settingStore := database.NewSettingStore(db)

	// Initialize services
	authService := services.NewAuthService(userStore)
	eventService := services.NewEventService(eventStore)
	registrationService := services.NewRegistrationService(registrationStore, eventStore, cfg)
	settingsService := services.NewSettingsService(settingStore)

	// Seed admin user if configured
	if cfg.AdminUsername != "" && cfg.AdminPassword != "" {
		if err := authService.SeedAdmin(cfg.AdminUsername, cfg.AdminPassword); err != nil {
			log.Printf("Warning: could not seed admin user: %v", err)
		}
	}

	// Session store
	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7, // 7 days
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, settingsService)
	eventHandler := handlers.NewEventHandler(eventService, registrationService, settingsService, cfg.UploadDir)
	registrationHandler := handlers.NewRegistrationHandler(registrationService, eventService, settingsService)
	adminHandler := handlers.NewAdminHandler(eventService, registrationService, authService, settingsService)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logging)
	r.Use(middleware.MethodOverride)
	r.Use(middleware.Session(sessionStore))
	r.Use(middleware.Locale)
	r.Use(middleware.CSRF([]byte(cfg.CSRFKey), false))

	// Static files
	staticFS := http.Dir("static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

	// Uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	// Public routes
	r.Get("/", eventHandler.Home)
	r.Get("/event/{slug}", eventHandler.Show)
	r.Post("/event/{slug}/register", registrationHandler.Register)
	r.Get("/cancel/{token}", registrationHandler.Cancel)

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", authHandler.LoginForm)
		r.Post("/login", authHandler.Login)

		// Authenticated admin routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)

			r.Post("/logout", authHandler.Logout)
			r.Get("/", adminHandler.Dashboard)

			// Event management
			r.Get("/events", eventHandler.List)
			r.Get("/events/new", eventHandler.NewForm)
			r.Post("/events", eventHandler.Create)
			r.Get("/events/{id}/edit", eventHandler.EditForm)
			r.Put("/events/{id}", eventHandler.Update)
			r.Delete("/events/{id}", eventHandler.Delete)
			r.Post("/events/{id}/clone", eventHandler.Clone)
			r.Get("/events/{id}/attendees", adminHandler.Attendees)
			r.Get("/events/{id}/attendees/csv", adminHandler.AttendeesCSV)
			r.Delete("/events/{id}/attendees/{regID}", adminHandler.DeleteAttendee)

			// User management (admin only)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/users", adminHandler.Users)
				r.Get("/users/new", adminHandler.NewUserForm)
				r.Post("/users", adminHandler.CreateUser)
				r.Delete("/users/{id}", adminHandler.DeleteUser)
			})

			// Settings (admin only)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/settings", adminHandler.Settings)
				r.Put("/settings", adminHandler.UpdateSettings)
			})
		})
	})

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)

	return http.ListenAndServe(addr, r)
}
