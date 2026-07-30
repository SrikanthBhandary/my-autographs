package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/models"
)

type ShareLinkHandler struct {
	DB       *sql.DB
	ShareURL string // e.g. https://myapp.com/submit
}

type createShareLinkRequest struct {
	CategoryID string `json:"category_id"`
	ExpiresIn  *int   `json:"expires_in_hours,omitempty"`
	MaxUses    *int   `json:"max_uses,omitempty"`
}

func (h *ShareLinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)

	var req createShareLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CategoryID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category_id is required"})
		return
	}

	token := uuid.NewString()
	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		expiresAt = &t
	}

	var sl models.ShareLink
	err := h.DB.QueryRowContext(r.Context(), `
		INSERT INTO share_links (user_id, category_id, token, expires_at, max_uses)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, category_id, token, expires_at, max_uses, use_count, active, created_at
	`, userID, req.CategoryID, token, expiresAt, req.MaxUses).Scan(
		&sl.ID, &sl.UserID, &sl.CategoryID, &sl.Token, &sl.ExpiresAt, &sl.MaxUses, &sl.UseCount, &sl.Active, &sl.CreatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create share link"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"share_link": sl,
		"url":        fmt.Sprintf("%s/%s", h.ShareURL, sl.Token),
	})
}

func (h *ShareLinkHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(), `
		UPDATE share_links SET active = false WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to deactivate link"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "share link not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
}
