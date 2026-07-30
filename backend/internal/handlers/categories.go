package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/models"
)

type CategoryHandler struct {
	DB *sql.DB
}

// List returns every category belonging to the authenticated owner,
// including nested sub-categories (e.g. "School" -> "School A").
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, user_id, parent_id, name, type, created_at
		FROM categories WHERE user_id = $1 ORDER BY created_at ASC
	`, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list categories"})
		return
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.UserID, &c.ParentID, &c.Name, &c.Type, &c.CreatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read categories"})
			return
		}
		categories = append(categories, c)
	}

	writeJSON(w, http.StatusOK, categories)
}

type createCategoryRequest struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	ParentID *string `json:"parent_id,omitempty"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.Type == "" {
		req.Type = "custom"
	}

	var c models.Category
	err := h.DB.QueryRowContext(r.Context(), `
		INSERT INTO categories (user_id, parent_id, name, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, parent_id, name, type, created_at
	`, userID, req.ParentID, req.Name, req.Type).Scan(&c.ID, &c.UserID, &c.ParentID, &c.Name, &c.Type, &c.CreatedAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create category"})
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)
	id := r.PathValue("id")

	var req createCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	res, err := h.DB.ExecContext(r.Context(), `
		UPDATE categories SET name = $1, type = $2, parent_id = $3
		WHERE id = $4 AND user_id = $5
	`, req.Name, req.Type, req.ParentID, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update category"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(), `
		DELETE FROM categories WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete category"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "category not found"})
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
