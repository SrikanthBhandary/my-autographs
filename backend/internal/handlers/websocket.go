package handlers

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/yourorg/autograph-backend/internal/ws"
)

type WebSocketHandler struct {
	Hub       *ws.Hub
	JWTSecret string
}

var wsUpgrader = websocket.Upgrader{
	// CORS for normal HTTP requests is already handled by middleware.CORS;
	// this only governs the WebSocket handshake itself, which doesn't go
	// through that middleware chain the same way.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Serve upgrades to a WebSocket connection. Auth comes from a ?token=<jwt>
// query param rather than an Authorization header, since browsers can't set
// custom headers on the WebSocket handshake request.
func (h *WebSocketHandler) Serve(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "invalid token claims", http.StatusUnauthorized)
		return
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		http.Error(w, "invalid token claims", http.StatusUnauthorized)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote an HTTP error response on failure
	}

	h.Hub.Register(userID, conn)
	defer func() {
		h.Hub.Unregister(userID, conn)
		conn.Close()
	}()

	// We don't expect the browser to send anything meaningful over this
	// connection, but we still need to keep reading — it's the only way to
	// detect the client closing the tab/connection and clean up promptly.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
