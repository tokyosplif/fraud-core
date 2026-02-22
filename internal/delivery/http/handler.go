package http

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run(ctx context.Context, reader *kafka.Reader) {
	go func() {
		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					slog.Info("Kafka reader context cancelled")
					return
				}
				slog.Error("Kafka dashboard reader error", "err", err)
				select {
				case <-time.After(1 * time.Second):
					continue
				case <-ctx.Done():
					return
				}
			}
			h.broadcast <- m.Value
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			if client == nil {
				continue
			}
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if client == nil {
				h.mu.Unlock()
				continue
			}
			if _, ok := h.clients[client]; ok {
				if err := client.Close(); err != nil {
					slog.Error("WS close error", "err", err)
				}
				delete(h.clients, client)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					slog.Error("WS write error", "err", err)
					if cerr := client.Close(); cerr != nil {
						slog.Error("WS close error", "err", cerr)
					}
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func RegisterRoutes(mux *http.ServeMux, reader *kafka.Reader) {
	hub := NewHub()
	go hub.Run(context.Background(), reader)

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Upgrade failed", "err", err)
			return
		}

		hub.register <- conn

		<-r.Context().Done()
		hub.unregister <- conn
	})

	mux.Handle("/", http.FileServer(http.Dir("./frontend")))
}
