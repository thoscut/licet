package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"licet/internal/services"
)

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	PingInterval    int  `mapstructure:"ping_interval"`    // Seconds
	UpdateInterval  int  `mapstructure:"update_interval"`  // Seconds for server status updates
	MaxConnections  int  `mapstructure:"max_connections"`
	ReadBufferSize  int  `mapstructure:"read_buffer_size"`
	WriteBufferSize int  `mapstructure:"write_buffer_size"`
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		Enabled:         true,
		PingInterval:    30,
		UpdateInterval:  10,
		MaxConnections:  100,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// WebSocket message types
const (
	MsgTypeServerStatus  = "server_status"
	MsgTypeFeatureUpdate = "feature_update"
	MsgTypeUserCheckout  = "user_checkout"
	MsgTypeAlert         = "alert"
	MsgTypeSubscribe     = "subscribe"
	MsgTypeUnsubscribe   = "unsubscribe"
	MsgTypePing          = "ping"
	MsgTypePong          = "pong"
	MsgTypeError         = "error"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// SubscribeMessage represents a subscription request
type SubscribeMessage struct {
	Channels []string `json:"channels"` // e.g., ["server:27000@flex.example.com", "alerts"]
}

// Client represents a connected WebSocket client
type Client struct {
	hub           *WebSocketHub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	config       WebSocketConfig
	query        *services.QueryService
	storage      *services.StorageService
	alertService *services.AlertService
	ctx          context.Context
	cancel       context.CancelFunc
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(config WebSocketConfig, query *services.QueryService, storage *services.StorageService, alertService *services.AlertService) *WebSocketHub {
	ctx, cancel := context.WithCancel(context.Background())

	hub := &WebSocketHub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		config:       config,
		query:        query,
		storage:      storage,
		alertService: alertService,
		ctx:          ctx,
		cancel:       cancel,
	}

	return hub
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	// Start the status update broadcaster
	go h.statusBroadcaster()

	for {
		select {
		case <-h.ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			if len(h.clients) < h.config.MaxConnections {
				h.clients[client] = true
				log.WithField("total_clients", len(h.clients)).Debug("WebSocket client connected")
			} else {
				log.Warn("WebSocket max connections reached, rejecting client")
				close(client.send)
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.WithField("total_clients", len(h.clients)).Debug("WebSocket client disconnected")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully stops the WebSocket hub
func (h *WebSocketHub) Stop() {
	h.cancel()
	h.mu.Lock()
	for client := range h.clients {
		close(client.send)
	}
	h.mu.Unlock()
}

// statusBroadcaster periodically broadcasts server status updates
func (h *WebSocketHub) statusBroadcaster() {
	ticker := time.NewTicker(time.Duration(h.config.UpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.broadcastServerStatus()
		}
	}
}

// broadcastServerStatus sends current server status to all subscribed clients
func (h *WebSocketHub) broadcastServerStatus() {
	servers, err := h.query.GetAllServers()
	if err != nil {
		log.WithError(err).Error("Failed to get servers for WebSocket broadcast")
		return
	}

	for _, server := range servers {
		result, err := h.query.QueryServer(server.Hostname, server.Type)
		if err != nil {
			continue
		}

		msg := WebSocketMessage{
			Type:      MsgTypeServerStatus,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"server":   server.Hostname,
				"type":     server.Type,
				"status":   result.Status,
				"features": result.Features,
				"users":    result.Users,
			},
		}

		h.BroadcastToChannel("server:"+server.Hostname, msg)
	}
}

// BroadcastToChannel sends a message to clients subscribed to a specific channel
func (h *WebSocketHub) BroadcastToChannel(channel string, msg WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		subscribed := client.subscriptions[channel] || client.subscriptions["all"]
		client.mu.RUnlock()

		if subscribed {
			select {
			case client.send <- data:
			default:
				// Client buffer full, skip
			}
		}
	}
}

// BroadcastAlert sends an alert to all clients subscribed to alerts
func (h *WebSocketHub) BroadcastAlert(alert interface{}) {
	msg := WebSocketMessage{
		Type:      MsgTypeAlert,
		Timestamp: time.Now(),
		Data:      alert,
	}
	h.BroadcastToChannel("alerts", msg)
}

// BroadcastAll sends a message to all connected clients
func (h *WebSocketHub) BroadcastAll(msg WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	h.broadcast <- data
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	upgrader.ReadBufferSize = h.config.ReadBufferSize
	upgrader.WriteBufferSize = h.config.WriteBufferSize

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := &Client{
		hub:           h,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	// Default subscriptions
	client.subscriptions["all"] = true

	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcome := WebSocketMessage{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message":       "Connected to Licet WebSocket",
			"subscriptions": []string{"all"},
		},
	}
	if data, err := json.Marshal(welcome); err == nil {
		client.send <- data
	}
}

// readPump handles incoming messages from the client
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithError(err).Debug("WebSocket read error")
			}
			break
		}

		// Parse incoming message
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.WithError(err).Debug("Failed to parse WebSocket message")
			continue
		}

		c.handleMessage(msg)
	}
}

// writePump handles outgoing messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(time.Duration(c.hub.config.PingInterval) * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch pending messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming client messages
func (c *Client) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case MsgTypeSubscribe:
		c.handleSubscribe(msg)
	case MsgTypeUnsubscribe:
		c.handleUnsubscribe(msg)
	case MsgTypePing:
		c.handlePing()
	default:
		log.WithField("type", msg.Type).Debug("Unknown WebSocket message type")
	}
}

// handleSubscribe adds subscriptions for the client
func (c *Client) handleSubscribe(msg WebSocketMessage) {
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if channels, ok := data["channels"].([]interface{}); ok {
			c.mu.Lock()
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					c.subscriptions[channel] = true
					log.WithField("channel", channel).Debug("Client subscribed to channel")
				}
			}
			c.mu.Unlock()

			// Send confirmation
			response := WebSocketMessage{
				Type:      "subscribed",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"channels": channels,
				},
			}
			if respData, err := json.Marshal(response); err == nil {
				c.send <- respData
			}
		}
	}
}

// handleUnsubscribe removes subscriptions for the client
func (c *Client) handleUnsubscribe(msg WebSocketMessage) {
	if data, ok := msg.Data.(map[string]interface{}); ok {
		if channels, ok := data["channels"].([]interface{}); ok {
			c.mu.Lock()
			for _, ch := range channels {
				if channel, ok := ch.(string); ok {
					delete(c.subscriptions, channel)
					log.WithField("channel", channel).Debug("Client unsubscribed from channel")
				}
			}
			c.mu.Unlock()
		}
	}
}

// handlePing responds to client ping
func (c *Client) handlePing() {
	response := WebSocketMessage{
		Type:      MsgTypePong,
		Timestamp: time.Now(),
	}
	if data, err := json.Marshal(response); err == nil {
		c.send <- data
	}
}

// WebSocketHandler returns the HTTP handler for WebSocket connections
func WebSocketHandler(hub *WebSocketHub) http.HandlerFunc {
	return hub.HandleWebSocket
}

// WebSocketStatsHandler returns statistics about WebSocket connections
func WebSocketStatsHandler(hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := map[string]interface{}{
			"connected_clients": hub.GetClientCount(),
			"max_connections":   hub.config.MaxConnections,
			"update_interval":   hub.config.UpdateInterval,
			"ping_interval":     hub.config.PingInterval,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}
