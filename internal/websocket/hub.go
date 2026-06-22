package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"

	"locator/internal/service"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

type messageEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Hub struct {
	logger   *slog.Logger
	upgrader gorillaws.Upgrader

	mu      sync.RWMutex
	clients map[*client]struct{}
}

type client struct {
	conn *gorillaws.Conn
	send chan []byte
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	return &Hub{
		logger: logger,
		upgrader: gorillaws.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		clients: make(map[*client]struct{}),
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn("websocket upgrade failed", "error", err)
		return
	}

	client := &client{
		conn: conn,
		send: make(chan []byte, 32),
	}

	h.addClient(client)

	go h.writePump(client)
	h.readPump(client)
}

func (h *Hub) Shutdown(_ context.Context) error {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
		delete(h.clients, client)
	}
	h.mu.Unlock()

	for _, client := range clients {
		close(client.send)
		_ = client.conn.Close()
	}

	return nil
}

func (h *Hub) PublishLocation(event service.LocationEvent) {
	h.publish("location", event)
}

func (h *Hub) PublishDeviceStatus(event service.DeviceStatusEvent) {
	h.publish("device_status", event)
}

func (h *Hub) PublishAlarm(event service.AlarmEvent) {
	h.publish("alarm", event)
}

func (h *Hub) publish(eventType string, data any) {
	body, err := json.Marshal(messageEnvelope{
		Type: eventType,
		Data: data,
	})
	if err != nil {
		h.logger.Error("marshal websocket message", "type", eventType, "error", err)
		return
	}

	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- body:
		default:
			h.removeClient(client)
		}
	}
}

func (h *Hub) addClient(client *client) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	count := len(h.clients)
	h.mu.Unlock()

	h.logger.Info("websocket client connected", "clients", count)
}

func (h *Hub) removeClient(client *client) {
	h.mu.Lock()
	_, exists := h.clients[client]
	if exists {
		delete(h.clients, client)
	}
	count := len(h.clients)
	h.mu.Unlock()

	if exists {
		_ = client.conn.Close()
		h.logger.Info("websocket client disconnected", "clients", count)
	}
}

func (h *Hub) readPump(client *client) {
	defer h.removeClient(client)

	client.conn.SetReadLimit(maxMessageSize)
	_ = client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (h *Hub) writePump(client *client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer h.removeClient(client)

	for {
		select {
		case message, ok := <-client.send:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = client.conn.WriteMessage(gorillaws.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(gorillaws.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
