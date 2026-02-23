package http

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/tokyosplif/fraud-core/internal/domain"
	"github.com/tokyosplif/fraud-core/pkg/closer"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *Hub) BroadcastAlert(alert domain.FraudAlert) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		if err := client.WriteJSON(alert); err != nil {
			slog.Warn("WS write error, closing client", "err", err)
			closer.SafeClose(client, "ws.client")
			delete(h.clients, client)
		}
	}
}

func RegisterRoutes(mux *http.ServeMux, hub *Hub) {
	mux.Handle("/", http.FileServer(http.Dir("./frontend")))

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Upgrade failed", "err", err)
			return
		}

		hub.mu.Lock()
		hub.clients[conn] = true
		hub.mu.Unlock()
		slog.Info("New dashboard client connected", "addr", r.RemoteAddr)

		go func() {
			defer func() {
				hub.mu.Lock()
				delete(hub.clients, conn)
				hub.mu.Unlock()
				closer.SafeClose(conn, "ws.client")
				slog.Info("Browser disconnected")
			}()

			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})
}
