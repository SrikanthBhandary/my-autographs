package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/yourorg/autograph-backend/internal/models"
)

type shareLinkCtxKey string

const ShareLinkKey shareLinkCtxKey = "shareLink"

// RequireValidShareLink looks up the :token path value, checks it exists,
// is active, isn't expired, and hasn't hit its usage cap — then injects the
// resolved ShareLink into the request context. This is the ONLY gate on the
// guest submission routes; no JWT/login is involved at all.
func RequireValidShareLink(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.PathValue("token")
			if token == "" {
				writeError(w, http.StatusBadRequest, "missing share token")
				return
			}

			var sl models.ShareLink
			err := db.QueryRowContext(r.Context(), `
				SELECT id, user_id, category_id, token, expires_at, max_uses, use_count, active, created_at
				FROM share_links WHERE token = $1
			`, token).Scan(&sl.ID, &sl.UserID, &sl.CategoryID, &sl.Token, &sl.ExpiresAt, &sl.MaxUses, &sl.UseCount, &sl.Active, &sl.CreatedAt)

			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "this link is invalid")
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to validate link")
				return
			}
			if !sl.Active {
				writeError(w, http.StatusGone, "this link has been deactivated")
				return
			}
			if sl.ExpiresAt != nil && time.Now().After(*sl.ExpiresAt) {
				writeError(w, http.StatusGone, "this link has expired")
				return
			}
			if sl.MaxUses != nil && sl.UseCount >= *sl.MaxUses {
				writeError(w, http.StatusGone, "this link has reached its submission limit")
				return
			}

			ctx := context.WithValue(r.Context(), ShareLinkKey, sl)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ShareLinkFrom(r *http.Request) models.ShareLink {
	sl, _ := r.Context().Value(ShareLinkKey).(models.ShareLink)
	return sl
}
