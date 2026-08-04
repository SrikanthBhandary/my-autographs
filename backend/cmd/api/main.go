package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourorg/autograph-backend/internal/config"
	"github.com/yourorg/autograph-backend/internal/db"
	"github.com/yourorg/autograph-backend/internal/handlers"
	"github.com/yourorg/autograph-backend/internal/mailer"
	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/queue"
	"github.com/yourorg/autograph-backend/internal/storage"
	"github.com/yourorg/autograph-backend/internal/ws"
)

func main() {
	// Load .env into the process environment if present. In production you'd
	// typically set real env vars instead and this becomes a no-op (the file
	// just won't exist), which is why we ignore the error here.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	store, err := storage.New(context.Background(), cfg.S3, os.Getenv("S3_PUBLIC_BASE_URL"))
	if err != nil {
		log.Fatalf("storage error: %v", err)
	}

	mail := mailer.New(mailer.Config{
		Host:      cfg.SMTP.Host,
		Port:      cfg.SMTP.Port,
		Username:  cfg.SMTP.Username,
		Password:  cfg.SMTP.Password,
		FromEmail: cfg.SMTP.FromEmail,
		FromName:  cfg.SMTP.FromName,
	})
	if !mail.Configured() {
		log.Println("SMTP_HOST not set — email notifications are disabled")
	}

	frontendOrigin := os.Getenv("FRONTEND_ORIGIN")

	// PDF export degrades gracefully if RabbitMQ isn't reachable — the rest
	// of the app (categories, entries, submissions, everything else) keeps
	// working fine without it.
	mq, err := queue.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Printf("rabbitmq unavailable (%v) — PDF export is disabled", err)
		mq = nil
	} else {
		defer mq.Close()
	}

	hub := ws.NewHub()
	if mq != nil {
		go forwardNotifications(mq, hub)
	}

	authH := &handlers.AuthHandler{DB: database, Cfg: cfg}
	catH := &handlers.CategoryHandler{DB: database}
	linkH := &handlers.ShareLinkHandler{DB: database, ShareURL: cfg.ShareURL}
	entryH := &handlers.EntryHandler{DB: database, Storage: store, Mailer: mail, FrontendURL: frontendOrigin}
	exportH := &handlers.ExportHandler{DB: database, Queue: mq}
	wsH := &handlers.WebSocketHandler{Hub: hub, JWTSecret: cfg.JWT.Secret}

	requireAuth := middleware.RequireAuth(cfg.JWT.Secret)
	requireShareLink := middleware.RequireValidShareLink(database)

	// Per-IP rate limits on the public, unauthenticated endpoints — these
	// are the ones a bad actor could hit without any credentials at all.
	// Owner routes are already gated by JWT so they're lower risk here.
	authLimiter := middleware.NewRateLimiter(10, time.Minute)   // login/signup: brute-force & spam-signup protection
	submitLimiter := middleware.NewRateLimiter(20, time.Minute) // guest submissions: spam protection

	mux := http.NewServeMux()

	// --- Public auth routes ---
	mux.Handle("POST /api/auth/signup", authLimiter.Middleware(http.HandlerFunc(authH.Signup)))
	mux.Handle("POST /api/auth/login", authLimiter.Middleware(http.HandlerFunc(authH.Login)))

	// --- Public guest submission routes (no login, gated by share-link token) ---
	mux.Handle("POST /api/submit/{token}", submitLimiter.Middleware(requireShareLink(http.HandlerFunc(entryH.Submit))))

	// --- Owner-only routes (JWT required) ---
	mux.Handle("GET /api/categories", requireAuth(http.HandlerFunc(catH.List)))
	mux.Handle("POST /api/categories", requireAuth(http.HandlerFunc(catH.Create)))
	mux.Handle("PUT /api/categories/{id}", requireAuth(http.HandlerFunc(catH.Update)))
	mux.Handle("DELETE /api/categories/{id}", requireAuth(http.HandlerFunc(catH.Delete)))

	mux.Handle("POST /api/sharelinks", requireAuth(http.HandlerFunc(linkH.Create)))
	mux.Handle("PATCH /api/sharelinks/{id}/deactivate", requireAuth(http.HandlerFunc(linkH.Deactivate)))

	mux.Handle("GET /api/entries", requireAuth(http.HandlerFunc(entryH.List)))
	mux.Handle("PATCH /api/entries/{id}/approve", requireAuth(http.HandlerFunc(entryH.Approve)))
	mux.Handle("PATCH /api/entries/{id}/reject", requireAuth(http.HandlerFunc(entryH.Reject)))

	mux.Handle("POST /api/export/pdf", requireAuth(http.HandlerFunc(exportH.CreatePDFJob)))
	mux.Handle("GET /api/export/pdf/{id}", requireAuth(http.HandlerFunc(exportH.GetPDFJob)))

	// Not wrapped in requireAuth: the WebSocket handshake can't carry a
	// custom Authorization header, so wsH.Serve validates the JWT itself
	// from a ?token= query param instead.
	mux.HandleFunc("GET /api/ws", wsH.Serve)

	handler := middleware.CORS(frontendOrigin)(mux)

	log.Printf("listening on :%s", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, handler); err != nil {
		log.Fatal(err)
	}
}

// forwardNotifications consumes PDF-job-completion events from RabbitMQ's
// fanout exchange (every API instance gets its own copy) and pushes them
// down to the relevant user's WebSocket connection, if this instance
// happens to be holding it.
func forwardNotifications(mq *queue.Queue, hub *ws.Hub) {
	deliveries, err := mq.ConsumeNotifications(context.Background())
	if err != nil {
		log.Printf("failed to start consuming pdf notifications: %v", err)
		return
	}
	for d := range deliveries {
		var n queue.JobNotification
		if err := json.Unmarshal(d.Body, &n); err != nil {
			log.Printf("failed to unmarshal pdf notification: %v", err)
			continue
		}
		hub.SendToUser(n.UserID, n)
	}
}
