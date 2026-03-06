package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/toulibre/libreregistration/internal/config"
	"github.com/toulibre/libreregistration/internal/database"
	"github.com/toulibre/libreregistration/internal/handlers"
	"github.com/toulibre/libreregistration/internal/middleware"
	"github.com/toulibre/libreregistration/internal/models"
	"github.com/toulibre/libreregistration/internal/services"
)

const (
	port      = "18921"
	baseURL   = "http://localhost:18921"
	adminUser = "admin"
	adminPass = "screenshot-password"
)

// eventIDs stores created event IDs for use in screenshot URLs.
var eventIDs []string

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Create temp directory for database and uploads
	tmpDir, err := os.MkdirTemp("", "libreregistration-screenshots-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	uploadDir := filepath.Join(tmpDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("create upload dir: %w", err)
	}

	// Open database and run migrations
	db, err := database.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Initialize stores and services
	userStore := database.NewUserStore(db)
	eventStore := database.NewEventStore(db)
	registrationStore := database.NewRegistrationStore(db)
	settingStore := database.NewSettingStore(db)

	cfg := &config.Config{
		Port:          port,
		BaseURL:       baseURL,
		SessionSecret: "screenshot-session-secret-32chars!",
		CSRFKey:       "screenshot-csrf-key-32-chars!!!!",
		UploadDir:     uploadDir,
	}

	authService := services.NewAuthService(userStore)
	eventService := services.NewEventService(eventStore)
	registrationService := services.NewRegistrationService(registrationStore, eventStore, cfg)
	settingsService := services.NewSettingsService(settingStore)

	// Seed test data
	if err := seedData(authService, eventService, registrationService); err != nil {
		return fmt.Errorf("seed data: %w", err)
	}

	// Start HTTP server
	srv := startServer(cfg, authService, eventService, registrationService, settingsService, uploadDir)
	defer srv.Close()

	waitForServer()

	// Take screenshots
	if err := takeScreenshots(); err != nil {
		return fmt.Errorf("take screenshots: %w", err)
	}

	log.Println("Done! Screenshots saved to screenshots/")
	return nil
}

func seedData(auth *services.AuthService, events *services.EventService, regs *services.RegistrationService) error {
	// Create admin user
	if err := auth.SeedAdmin(adminUser, adminPass); err != nil {
		return err
	}

	users, err := auth.ListUsers()
	if err != nil {
		return err
	}
	adminID := users[0].ID

	now := time.Now()

	// Event 1: large community meetup with map coordinates
	deadline1 := now.Add(30 * 24 * time.Hour)
	capacity1 := 50
	lat1 := 43.5985
	lng1 := 1.4468
	event1 := &models.Event{
		Title: "Rencontres du Logiciel Libre 2026",
		Description: `Venez découvrir le monde du logiciel libre lors de notre rencontre annuelle !

## Programme

- **14h00** — Accueil et café
- **14h30** — Conférences éclair (lightning talks)
- **16h00** — Pause
- **16h30** — Table ronde : « Le libre dans l'éducation »
- **18h00** — Apéritif et networking

Événement ouvert à toutes et tous, débutant·e·s bienvenu·e·s !`,
		Location:             "Espace des Diversités, 33 rue des Tourneurs, Toulouse",
		EventDate:            now.Add(45 * 24 * time.Hour),
		RegistrationDeadline: &deadline1,
		MaxCapacity:          &capacity1,
		AttendeeListPublic:   true,
		RegistrationOpen:     true,
		Latitude:             &lat1,
		Longitude:            &lng1,
		CreatedBy:            adminID,
	}

	// Event 2: small workshop
	deadline2 := now.Add(14 * 24 * time.Hour)
	capacity2 := 20
	event2 := &models.Event{
		Title: "Atelier Git & GitHub pour débutants",
		Description: `Initiez-vous à Git et GitHub lors de cet atelier pratique.

## Ce que vous apprendrez

- Créer un dépôt Git
- Les commandes essentielles : ` + "`add`" + `, ` + "`commit`" + `, ` + "`push`" + `, ` + "`pull`" + `
- Travailler avec les branches
- Contribuer à un projet open source

**Prérequis** : apportez votre ordinateur portable avec Git installé.`,
		Location:             "La Cantine Toulouse, 27 rue d'Aubuisson",
		EventDate:            now.Add(20 * 24 * time.Hour),
		RegistrationDeadline: &deadline2,
		MaxCapacity:          &capacity2,
		AttendeeListPublic:   true,
		RegistrationOpen:     true,
		CreatedBy:            adminID,
	}

	// Event 3: conference, no capacity limit
	event3 := &models.Event{
		Title: "Conférence LibreOffice : trucs et astuces",
		Description: `Découvrez les fonctionnalités méconnues de LibreOffice avec notre intervenant spécialisé.

## Au programme

- Les styles et modèles avancés dans Writer
- Formules et macros dans Calc
- Présentations dynamiques avec Impress
- Questions / réponses`,
		Location:           "Médiathèque José Cabanis, Toulouse",
		EventDate:          now.Add(60 * 24 * time.Hour),
		AttendeeListPublic: false,
		RegistrationOpen:   true,
		CreatedBy:          adminID,
	}

	for _, e := range []*models.Event{event1, event2, event3} {
		if err := events.Create(e); err != nil {
			return fmt.Errorf("create event %q: %w", e.Title, err)
		}
	}

	eventIDs = []string{event1.ID, event2.ID, event3.ID}

	// Register attendees
	ctx := context.Background()
	attendees := []struct {
		eventID, name, email, comment string
	}{
		{event1.ID, "Marie Dupont", "marie@example.fr", "J'ai hâte d'y être !"},
		{event1.ID, "Pierre Martin", "pierre.m@example.fr", ""},
		{event1.ID, "Sophie", "", "Première participation"},
		{event1.ID, "Lucas Bernard", "lucas@example.fr", "Je viendrai avec un ami"},
		{event1.ID, "Camille Leroy", "", ""},
		{event1.ID, "Thomas Moreau", "thomas@example.fr", "Très intéressé par la table ronde"},
		{event1.ID, "Léa Petit", "", ""},
		{event1.ID, "Hugo Roux", "hugo.roux@example.fr", ""},

		{event2.ID, "Alice Fournier", "alice@example.fr", "Complètement débutante, c'est OK ?"},
		{event2.ID, "Maxime Girard", "", ""},
		{event2.ID, "Emma Bonnet", "emma.b@example.fr", "J'ai déjà utilisé Git en ligne de commande"},
		{event2.ID, "Nathan", "", ""},
		{event2.ID, "Chloé Mercier", "chloe@example.fr", ""},

		{event3.ID, "Julie Lambert", "julie@example.fr", ""},
		{event3.ID, "Antoine", "", "Fan de LibreOffice depuis 10 ans !"},
		{event3.ID, "Manon Faure", "manon@example.fr", ""},
	}

	for _, a := range attendees {
		if _, err := regs.Register(ctx, a.eventID, a.name, a.email, a.comment); err != nil {
			return fmt.Errorf("register %q: %w", a.name, err)
		}
	}

	log.Printf("Seeded 3 events and %d registrations", len(attendees))
	return nil
}

func startServer(cfg *config.Config, auth *services.AuthService, events *services.EventService, regs *services.RegistrationService, settings *services.SettingsService, uploadDir string) *http.Server {
	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   86400 * 7,
	}

	authHandler := handlers.NewAuthHandler(auth, settings)
	eventHandler := handlers.NewEventHandler(events, regs, settings, uploadDir)
	registrationHandler := handlers.NewRegistrationHandler(regs, events, settings)
	adminHandler := handlers.NewAdminHandler(events, regs, auth, settings)

	r := chi.NewRouter()
	r.Use(middleware.Logging)
	r.Use(middleware.MethodOverride)
	r.Use(middleware.Session(sessionStore))
	r.Use(middleware.Locale)
	r.Use(middleware.CSRF([]byte(cfg.CSRFKey), false))

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	r.Get("/", eventHandler.Home)
	r.Get("/event/{slug}", eventHandler.Show)
	r.Post("/event/{slug}/register", registrationHandler.Register)
	r.Get("/cancel/{token}", registrationHandler.Cancel)

	r.Route("/admin", func(r chi.Router) {
		r.Get("/login", authHandler.LoginForm)
		r.Post("/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)
			r.Post("/logout", authHandler.Logout)
			r.Get("/", adminHandler.Dashboard)
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

			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/users", adminHandler.Users)
				r.Get("/users/new", adminHandler.NewUserForm)
				r.Post("/users", adminHandler.CreateUser)
				r.Delete("/users/{id}", adminHandler.DeleteUser)
			})
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin)
				r.Get("/settings", adminHandler.Settings)
				r.Put("/settings", adminHandler.UpdateSettings)
			})
		})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return srv
}

func waitForServer() {
	for range 50 {
		resp, err := http.Get(baseURL)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Fatal("Server did not start in time")
}

func takeScreenshots() error {
	if err := os.MkdirAll("screenshots", 0755); err != nil {
		return err
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1280, 900),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Public home page
	log.Println("Capturing: home page")
	if err := captureScreenshot(ctx, baseURL, "screenshots/01-home.png"); err != nil {
		return err
	}

	// Event detail page (first event)
	log.Println("Capturing: event detail page")
	if err := captureScreenshot(ctx, baseURL+"/event/rencontres-du-logiciel-libre-2026", "screenshots/02-event.png"); err != nil {
		return err
	}

	// Admin login
	log.Println("Logging in to admin panel")
	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/admin/login"),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
		chromedp.SendKeys(`#username`, adminUser, chromedp.ByID),
		chromedp.SendKeys(`#password`, adminPass, chromedp.ByID),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`main`, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("admin login: %w", err)
	}

	// Admin dashboard
	log.Println("Capturing: admin dashboard")
	if err := captureScreenshot(ctx, baseURL+"/admin/", "screenshots/03-admin-dashboard.png"); err != nil {
		return err
	}

	// Admin events list
	log.Println("Capturing: admin events list")
	if err := captureScreenshot(ctx, baseURL+"/admin/events", "screenshots/04-admin-events.png"); err != nil {
		return err
	}

	// Admin attendees (first event)
	log.Println("Capturing: admin attendees")
	if err := captureScreenshot(ctx, fmt.Sprintf("%s/admin/events/%s/attendees", baseURL, eventIDs[0]), "screenshots/05-admin-attendees.png"); err != nil {
		return err
	}

	return nil
}

func captureScreenshot(ctx context.Context, url, path string) error {
	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady(`body`, chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.CaptureScreenshot(&buf),
	); err != nil {
		return fmt.Errorf("screenshot %s: %w", path, err)
	}
	log.Printf("  -> %s (%d KB)", path, len(buf)/1024)
	return os.WriteFile(path, buf, 0644)
}
