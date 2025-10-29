package websocket

import (
	"encoding/json"
	"log"
	"sync"

	m "monserv/internal/metrics"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	mu sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket] Client registered, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Client unregistered, total: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client buffer full, close the connection
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastMetrics sends metrics update to all connected clients
func (h *Hub) BroadcastMetrics(state map[string]*m.ServerMetrics) {
	data, err := json.Marshal(map[string]interface{}{
		"type": "metrics_update",
		"data": state,
	})
	if err != nil {
		log.Printf("[WebSocket] Error marshaling metrics: %v", err)
		return
	}

	h.broadcast <- data
}

// BroadcastAlert sends alert notification to all connected clients
func (h *Hub) BroadcastAlert(alertType, subject, message string) {
	data, err := json.Marshal(map[string]interface{}{
		"type":       "alert",
		"alert_type": alertType,
		"subject":    subject,
		"message":    message,
	})
	if err != nil {
		log.Printf("[WebSocket] Error marshaling alert: %v", err)
		return
	}

	h.broadcast <- data
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
