package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/handlers"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/services"
	"github.com/toulibre/libreregistration/templates/public"
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
		Secure:   strings.HasPrefix(cfg.BaseURL, "https://"),
		MaxAge:   86400 * 7, // 7 days
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	authHandler := handlers.NewAuthHandler(authService, registrationService, settingsService, cfg)
	eventHandler := handlers.NewEventHandler(eventService, registrationService, authService, settingsService, cfg.UploadDir, cfg.BaseURL)
	registrationHandler := handlers.NewRegistrationHandler(registrationService, eventService, authService, settingsService)
	adminHandler := handlers.NewAdminHandler(eventService, registrationService, authService, settingsService, cfg.UploadDir)
	accountHandler := handlers.NewAccountHandler(authService, eventService, registrationService, settingsService, cfg.UploadDir)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Logging)
	r.Use(middleware.MethodOverride)
	r.Use(middleware.Session(sessionStore))
	r.Use(middleware.Locale)
	r.Use(middleware.CSRF([]byte(cfg.CSRFKey), cfg.BaseURL))
	r.Use(middleware.LoadUser)
	r.Use(middleware.InjectSelfRegistration(settingsService.AllowSelfRegistration))

	// Custom 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		siteName, accentColor := settingsService.GetSiteSettings()
		w.WriteHeader(http.StatusNotFound)
		public.NotFound(siteName, accentColor).Render(r.Context(), w)
	})

	// Health check
	r.Get("/healthz", healthHandler.Healthz)

	// Static files
	staticFS := http.Dir("static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

	// Uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	// Public routes
	r.Get("/", eventHandler.Home)
	r.Get("/event/{slug}", eventHandler.Show)
	r.Get("/event/{slug}/ical", eventHandler.ICal)
	r.Post("/event/{slug}/register", registrationHandler.Register)
	r.Post("/event/{slug}/cancel", registrationHandler.CancelByUser)
	r.Get("/cancel/{token}", registrationHandler.Cancel)

	// Self-registration and email verification
	r.Get("/register", authHandler.RegisterForm)
	r.Post("/register", authHandler.RegisterUser)
	r.Get("/verify-email/{token}", authHandler.VerifyEmail)

	// Logout (any authenticated user)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Post("/logout", authHandler.Logout)
	})

	// User account area (any authenticated user)
	r.Route("/account", func(r chi.Router) {
		r.Use(middleware.RequireAuth)

		r.Get("/", accountHandler.Dashboard)
		r.Get("/profile", accountHandler.ProfileForm)
		r.Put("/profile", accountHandler.UpdateProfile)
		r.Get("/password", accountHandler.PasswordForm)
		r.Put("/password", accountHandler.ChangePassword)
		r.Post("/resend-verification", authHandler.ResendVerification)
		r.Get("/delete", accountHandler.DeleteForm)
		r.Post("/delete", accountHandler.DeleteAccount)
		r.Get("/events/{id}/attendees", accountHandler.OrganizerAttendees)
		r.Get("/events/{id}/attendees/csv", accountHandler.OrganizerAttendeesCSV)
	})

	// Admin routes
	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", authHandler.LoginForm)
		r.Post("/login", authHandler.Login)

		// Authenticated staff routes (admin + manager)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Use(middleware.RequireStaff)

			r.Get("/", adminHandler.Dashboard)
			r.Get("/password", adminHandler.PasswordForm)
			r.Put("/password", adminHandler.ChangePassword)

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
				r.Get("/users/{id}/edit", adminHandler.EditUserForm)
				r.Put("/users/{id}", adminHandler.UpdateUser)
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
