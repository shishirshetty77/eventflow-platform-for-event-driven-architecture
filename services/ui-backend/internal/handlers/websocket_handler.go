// Package handlers provides WebSocket handling.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	sharedkafka "github.com/microservices-platform/pkg/shared/kafka"
	"github.com/microservices-platform/pkg/shared/logging"
	"github.com/microservices-platform/pkg/shared/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSHub manages WebSocket connections.
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	logger     *logging.Logger
	mu         sync.RWMutex
}

// NewWSHub creates a new WSHub.
func NewWSHub(logger *logging.Logger) *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		logger:     logger,
	}
}

// Run starts the hub.
func (h *WSHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client registered", zap.String("id", client.id))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("client unregistered", zap.String("id", client.id))
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

// Broadcast sends a message to all clients.
func (h *WSHub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		h.logger.Warn("broadcast channel full, dropping message")
	}
}

// ClientCount returns the number of connected clients.
func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// WSClient represents a WebSocket client.
type WSClient struct {
	hub    *WSHub
	conn   *websocket.Conn
	send   chan []byte
	id     string
	userID string
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("websocket error", zap.Error(err))
			}
			break
		}

		// Handle client messages (subscriptions, etc.)
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			c.handleMessage(&msg)
		}
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) handleMessage(msg *WSMessage) {
	// Handle different message types
	switch msg.Type {
	case "subscribe":
		// Handle subscription requests
	case "unsubscribe":
		// Handle unsubscription requests
	case "ping":
		// Respond with pong
		response := WSMessage{Type: "pong", Payload: time.Now().Unix()}
		data, _ := json.Marshal(response)
		c.send <- data
	}
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub    *WSHub
	logger *logging.Logger
}

// NewWSHandler creates a new WSHandler.
func NewWSHandler(hub *WSHub, logger *logging.Logger) *WSHandler {
	return &WSHandler{
		hub:    hub,
		logger: logger,
	}
}

// ServeWS handles WebSocket requests.
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	userID, _ := r.Context().Value("user_id").(string)

	client := &WSClient{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		id:     generateClientID(),
		userID: userID,
	}

	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func generateClientID() string {
	return time.Now().Format("20060102150405.999999999")
}

// MetricsStreamer streams metrics from Kafka to WebSocket clients.
type MetricsStreamer struct {
	hub           *WSHub
	metricsReader *kafka.Reader
	alertsReader  *kafka.Reader
	logger        *logging.Logger
	running       bool
	mu            sync.Mutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewMetricsStreamer creates a new MetricsStreamer.
func NewMetricsStreamer(
	brokers []string,
	metricsTopic, alertsTopic, consumerGroup string,
	hub *WSHub,
	logger *logging.Logger,
) *MetricsStreamer {
	metricsConfig := sharedkafka.DefaultConsumerConfig(brokers, metricsTopic, consumerGroup+"-metrics")
	metricsConfig.StartOffset = kafka.LastOffset

	alertsConfig := sharedkafka.DefaultConsumerConfig(brokers, alertsTopic, consumerGroup+"-alerts")
	alertsConfig.StartOffset = kafka.LastOffset

	return &MetricsStreamer{
		hub:           hub,
		metricsReader: kafka.NewReader(metricsConfig),
		alertsReader:  kafka.NewReader(alertsConfig),
		logger:        logger,
	}
}

// Start starts the streamer.
func (s *MetricsStreamer) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.logger.Info("starting metrics streamer")

	s.wg.Add(2)
	go s.streamMetrics(ctx)
	go s.streamAlerts(ctx)

	return nil
}

// Stop stops the streamer.
func (s *MetricsStreamer) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()

	s.metricsReader.Close()
	s.alertsReader.Close()

	s.logger.Info("metrics streamer stopped")
	return nil
}

func (s *MetricsStreamer) streamMetrics(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
			msg, err := s.metricsReader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			var metric models.ServiceMetric
			if err := json.Unmarshal(msg.Value, &metric); err != nil {
				s.metricsReader.CommitMessages(ctx, msg)
				continue
			}

			wsMsg := WSMessage{
				Type:    "metric",
				Payload: metric,
			}
			data, _ := json.Marshal(wsMsg)
			s.hub.Broadcast(data)

			s.metricsReader.CommitMessages(ctx, msg)
		}
	}
}

func (s *MetricsStreamer) streamAlerts(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		default:
			msg, err := s.alertsReader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			var alert models.Alert
			if err := json.Unmarshal(msg.Value, &alert); err != nil {
				s.alertsReader.CommitMessages(ctx, msg)
				continue
			}

			wsMsg := WSMessage{
				Type:    "alert",
				Payload: alert,
			}
			data, _ := json.Marshal(wsMsg)
			s.hub.Broadcast(data)

			s.alertsReader.CommitMessages(ctx, msg)
		}
	}
}
