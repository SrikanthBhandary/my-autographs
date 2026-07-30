package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/yourorg/autograph-backend/internal/config"
	"github.com/yourorg/autograph-backend/internal/middleware"
)

type AuthHandler struct {
	DB  *sql.DB
	Cfg *config.Config
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email, password, and name are required"})
		return
	}

	hash, err := middleware.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not hash password"})
		return
	}

	var userID string
	err = h.DB.QueryRowContext(r.Context(), `
		INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id
	`, req.Email, hash, req.Name).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already in use"})
		return
	}

	h.respondWithToken(w, userID, req.Email, req.Name)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	var userID, hash, name string
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, password_hash, name FROM users WHERE email = $1
	`, req.Email).Scan(&userID, &hash, &name)
	if err == sql.ErrNoRows || !middleware.CheckPassword(hash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}
	if err != nil && err != sql.ErrNoRows {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login failed"})
		return
	}

	h.respondWithToken(w, userID, req.Email, name)
}

func (h *AuthHandler) respondWithToken(w http.ResponseWriter, userID, email, name string) {
	token, err := middleware.IssueToken(h.Cfg.JWT.Secret, h.Cfg.JWT.ExpiryHrs, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue token"})
		return
	}
	var resp authResponse
	resp.Token = token
	resp.User.ID = userID
	resp.User.Email = email
	resp.User.Name = name
	writeJSON(w, http.StatusOK, resp)
}
