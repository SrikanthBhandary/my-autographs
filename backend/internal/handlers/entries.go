package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/lib/pq"
	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/models"
	"github.com/yourorg/autograph-backend/internal/storage"
)

type EntryHandler struct {
	DB      *sql.DB
	Storage *storage.Storage
}

const maxUploadSize = 20 << 20 // 20MB per submission (images + audio combined)

// Submit is the PUBLIC, no-login endpoint a guest hits after opening a share
// link. middleware.RequireValidShareLink has already validated the token and
// put the resolved ShareLink in the request context.
func (h *EntryHandler) Submit(w http.ResponseWriter, r *http.Request) {
	sl := middleware.ShareLinkFrom(r)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "form too large or malformed"})
		return
	}

	guestName := r.FormValue("guest_name")
	note := r.FormValue("note")
	if guestName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "guest_name is required"})
		return
	}

	ctx := r.Context()

	// Upload any images (field name "images", can appear multiple times).
	var imageURLs []string
	if r.MultipartForm != nil {
		for _, fh := range r.MultipartForm.File["images"] {
			url, err := h.Storage.UploadFile(ctx, sl.CategoryID, fh)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to upload image"})
				return
			}
			imageURLs = append(imageURLs, url)
		}
	}
	fmt.Println("Test", imageURLs)

	// Optional single audio file (field name "audio").
	var audioURL *string
	if files := r.MultipartForm.File["audio"]; len(files) > 0 {
		url, err := h.Storage.UploadFile(ctx, sl.CategoryID, files[0])
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to upload audio"})
			return
		}
		audioURL = &url
	}

	var entry models.Entry
	err := h.DB.QueryRowContext(ctx, `
		INSERT INTO entries (category_id, share_link_id, guest_name, note, image_urls, audio_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, category_id, share_link_id, guest_name, note, image_urls, audio_url, status, created_at
	`, sl.CategoryID, sl.ID, guestName, note, pq.Array(imageURLs), audioURL).Scan(
		&entry.ID, &entry.CategoryID, &entry.ShareLinkID, &entry.GuestName, &entry.Note,
		pq.Array(&entry.ImageURLs), &entry.AudioURL, &entry.Status, &entry.CreatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save entry"})
		return
	}

	// Bump the link's use count so max_uses enforcement stays accurate.
	_, _ = h.DB.ExecContext(ctx, `UPDATE share_links SET use_count = use_count + 1 WHERE id = $1`, sl.ID)

	writeJSON(w, http.StatusCreated, map[string]string{
		"status":  "submitted",
		"message": "Thanks! Your autograph is awaiting approval.",
	})
}

// List returns entries for the authenticated owner, optionally filtered by
// ?status=pending|approved|rejected and/or ?category_id=<uuid>. Used for the
// moderation queue and the book view (status=approved, optionally scoped to
// one category).
func (h *EntryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)
	status := r.URL.Query().Get("status")
	categoryID := r.URL.Query().Get("category_id")

	query := `
		SELECT e.id, e.category_id, e.share_link_id, e.guest_name, e.note, e.image_urls, e.audio_url, e.status, e.created_at
		FROM entries e
		JOIN categories c ON c.id = e.category_id
		WHERE c.user_id = $1
	`
	args := []interface{}{userID}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND e.status = $%d", len(args))
	}
	if categoryID != "" {
		args = append(args, categoryID)
		query += fmt.Sprintf(" AND e.category_id = $%d", len(args))
	}
	query += " ORDER BY e.created_at DESC"

	rows, err := h.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list entries"})
		return
	}
	defer rows.Close()

	entries := []models.Entry{}
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.CategoryID, &e.ShareLinkID, &e.GuestName, &e.Note,
			pq.Array(&e.ImageURLs), &e.AudioURL, &e.Status, &e.CreatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read entries"})
			return
		}
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, entries)
}

func (h *EntryHandler) setStatus(w http.ResponseWriter, r *http.Request, status models.EntryStatus) {
	userID := middleware.UserIDFrom(r)
	id := r.PathValue("id")

	res, err := h.DB.ExecContext(r.Context(), `
		UPDATE entries e SET status = $1
		FROM categories c
		WHERE e.category_id = c.id AND e.id = $2 AND c.user_id = $3
	`, status, id, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update entry"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "entry not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (h *EntryHandler) Approve(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, models.StatusApproved)
}

func (h *EntryHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, models.StatusRejected)
}
