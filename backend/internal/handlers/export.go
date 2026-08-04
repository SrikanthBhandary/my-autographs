package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/yourorg/autograph-backend/internal/middleware"
	"github.com/yourorg/autograph-backend/internal/models"
	"github.com/yourorg/autograph-backend/internal/queue"
)

type ExportHandler struct {
	DB    *sql.DB
	Queue *queue.Queue // nil if RabbitMQ wasn't reachable at startup — export is disabled gracefully in that case
}

type createExportRequest struct {
	CategoryID *string `json:"category_id,omitempty"` // omitted/null = export the whole book
}

// CreatePDFJob records a new export job and hands it off to the queue. It
// returns immediately (202 Accepted) — the actual PDF is built asynchronously
// by a worker process, and the browser is notified over WebSocket when it's
// ready (falling back to polling GetPDFJob if the notification is missed).
func (h *ExportHandler) CreatePDFJob(w http.ResponseWriter, r *http.Request) {
	if h.Queue == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "PDF export is temporarily unavailable"})
		return
	}

	userID := middleware.UserIDFrom(r)

	var req createExportRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // an empty/missing body is fine — means "whole book"

	var job models.PDFJob
	err := h.DB.QueryRowContext(r.Context(), `
		INSERT INTO pdf_jobs (user_id, category_id, status)
		VALUES ($1, $2, 'pending')
		RETURNING id, user_id, category_id, status, file_url, error, created_at, completed_at
	`, userID, req.CategoryID).Scan(
		&job.ID, &job.UserID, &job.CategoryID, &job.Status, &job.FileURL, &job.Error, &job.CreatedAt, &job.CompletedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create export job"})
		return
	}

	if err := h.Queue.PublishPDFJob(r.Context(), job.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to queue export job"})
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

// GetPDFJob lets the frontend poll a job's status — used as a fallback if
// the WebSocket connection was closed/missed the completion notification.
func (h *ExportHandler) GetPDFJob(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFrom(r)
	id := r.PathValue("id")

	var job models.PDFJob
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, user_id, category_id, status, file_url, error, created_at, completed_at
		FROM pdf_jobs WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(
		&job.ID, &job.UserID, &job.CategoryID, &job.Status, &job.FileURL, &job.Error, &job.CreatedAt, &job.CompletedAt,
	)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch job"})
		return
	}

	writeJSON(w, http.StatusOK, job)
}
