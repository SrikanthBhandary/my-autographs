package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // e.g. "school", "college", "company", "gym", or custom
	CreatedAt time.Time `json:"created_at"`
}

type ShareLink struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	CategoryID string     `json:"category_id"`
	Token      string     `json:"token"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	MaxUses    *int       `json:"max_uses,omitempty"`
	UseCount   int        `json:"use_count"`
	Active     bool       `json:"active"`
	CreatedAt  time.Time  `json:"created_at"`
}

type EntryStatus string

const (
	StatusPending  EntryStatus = "pending"
	StatusApproved EntryStatus = "approved"
	StatusRejected EntryStatus = "rejected"
)

type Entry struct {
	ID          string      `json:"id"`
	CategoryID  string      `json:"category_id"`
	ShareLinkID string      `json:"share_link_id"`
	GuestName   string      `json:"guest_name"`
	Note        string      `json:"note"`
	ImageURLs   []string    `json:"image_urls"`
	AudioURL    *string     `json:"audio_url,omitempty"`
	Status      EntryStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}
