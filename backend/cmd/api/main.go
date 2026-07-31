package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/yourorg/autograph-backend/internal/config"
	"github.com/yourorg/autograph-backend/internal/db"
	"github.com/yourorg/autograph-backend/internal/handlers"
	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/storage"
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

	authH := &handlers.AuthHandler{DB: database, Cfg: cfg}
	catH := &handlers.CategoryHandler{DB: database}
	linkH := &handlers.ShareLinkHandler{DB: database, ShareURL: cfg.ShareURL}
	entryH := &handlers.EntryHandler{DB: database, Storage: store}

	requireAuth := middleware.RequireAuth(cfg.JWT.Secret)
	requireShareLink := middleware.RequireValidShareLink(database)

	mux := http.NewServeMux()

	// --- Public auth routes ---
	mux.HandleFunc("POST /api/auth/signup", authH.Signup)
	mux.HandleFunc("POST /api/auth/login", authH.Login)

	// --- Public guest submission routes (no login, gated by share-link token) ---
	mux.Handle("POST /api/submit/{token}", requireShareLink(http.HandlerFunc(entryH.Submit)))

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

	handler := middleware.CORS(os.Getenv("FRONTEND_ORIGIN"))(mux)

	log.Printf("listening on :%s", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, handler); err != nil {
		log.Fatal(err)
	}
}
