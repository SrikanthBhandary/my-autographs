// Package ws tracks which users currently have an open WebSocket connection
// to this API instance, and lets other parts of the app push JSON messages
// down to them — used to notify a user in real time when their PDF export
// finishes, instead of making the browser poll for it.
package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	mu    sync.Mutex
	conns map[string][]*websocket.Conn // userID -> open connections (multiple tabs allowed)
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string][]*websocket.Conn)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[userID] = append(h.conns[userID], conn)
}

func (h *Hub) Unregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.conns[userID]
	for i, c := range list {
		if c == conn {
			h.conns[userID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(h.conns[userID]) == 0 {
		delete(h.conns, userID)
	}
}

// SendToUser pushes a JSON payload to every connection this instance holds
// for that user. It's a silent no-op if the user isn't connected here —
// with multiple API instances, that's expected and fine: whichever instance
// actually holds their connection will have received the same fanout
// notification and will handle it.
func (h *Hub) SendToUser(userID string, payload interface{}) {
	h.mu.Lock()
	conns := append([]*websocket.Conn{}, h.conns[userID]...)
	h.mu.Unlock()

	if len(conns) == 0 {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws: failed to marshal payload: %v", err)
		return
	}
	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, body); err != nil {
			log.Printf("ws: write error (client likely disconnected): %v", err)
		}
	}
}
