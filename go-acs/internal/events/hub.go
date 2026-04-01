package events

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

type EventType string

const (
	EventDeviceInform      EventType = "device.inform"
	EventDeviceOnline      EventType = "device.online"
	EventDeviceOffline     EventType = "device.offline"
	EventConnectionRequest EventType = "device.connection_request"
	EventTaskDispatched    EventType = "task.dispatched"
	EventTaskComplete      EventType = "task.complete"
	EventTaskFailed        EventType = "task.failed"
	EventParametersUpdated EventType = "device.parameters_updated"
)

type Message struct {
	Type     EventType   `json:"type"`
	TenantID string      `json:"-"`
	Payload  interface{} `json:"payload"`
	Time     time.Time   `json:"time"`
}

type client struct {
	conn     *websocket.Conn
	tenantID string
	send     chan []byte
	once     sync.Once
}

func (cl *client) close() {
	cl.once.Do(func() {
		close(cl.send)
		_ = cl.conn.Close()
	})
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[string]map[*client]bool
	broadcast  chan *Message
	register   chan *client
	unregister chan *client
	log        *zap.Logger
}

func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *client, 64),
		unregister: make(chan *client, 64),
		log:        log,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case cl := <-h.register:
			h.mu.Lock()
			if h.clients[cl.tenantID] == nil {
				h.clients[cl.tenantID] = make(map[*client]bool)
			}
			h.clients[cl.tenantID][cl] = true
			h.mu.Unlock()
			h.log.Debug("ws client connected", zap.String("tenant_id", cl.tenantID))

		case cl := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.clients[cl.tenantID]; ok {
				delete(room, cl)
				if len(room) == 0 {
					delete(h.clients, cl.tenantID)
				}
			}
			h.mu.Unlock()
			cl.close()
			h.log.Debug("ws client disconnected", zap.String("tenant_id", cl.tenantID))

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			h.mu.RLock()
			room := h.clients[msg.TenantID]
			h.mu.RUnlock()

			for cl := range room {
				select {
				case cl.send <- data:
				default:
					h.unregister <- cl
				}
			}
		}
	}
}

func (h *Hub) Broadcast(tenantID string, eventType EventType, payload interface{}) {
	select {
	case h.broadcast <- &Message{
		Type:     eventType,
		TenantID: tenantID,
		Payload:  payload,
		Time:     time.Now(),
	}:
	default:
		h.log.Warn("event hub broadcast buffer full", zap.String("tenant_id", tenantID), zap.String("type", string(eventType)))
	}
}

func (h *Hub) ServeWS(conn *websocket.Conn, tenantID string) {
	cl := &client{
		conn:     conn,
		tenantID: tenantID,
		send:     make(chan []byte, 64),
	}

	h.register <- cl

	go func() {
		defer func() { h.unregister <- cl }()
		for data := range cl.send {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	h.unregister <- cl
}
