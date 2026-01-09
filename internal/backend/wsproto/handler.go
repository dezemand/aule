package wsproto

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

var ErrAuthExpired = errors.New("authentication expired")

const CtxWsClient = "ws_client"
const CtxWsClientID = "ws_client_id"

type Handler struct {
	router  *Router
	mu      *sync.RWMutex
	clients map[uuid.UUID]*Client
}

func NewHandler(router *Router) *Handler {
	return &Handler{
		router:  router,
		clients: make(map[uuid.UUID]*Client),
		mu:      &sync.RWMutex{},
	}
}

func (h *Handler) Handle(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	user := auth.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	return h.handler()(c)
}

func (h *Handler) addClient(c *Client) {
	h.mu.Lock()
	h.clients[c.id] = c
	h.mu.Unlock()
}

func (h *Handler) removeClient(id uuid.UUID) {
	h.mu.Lock()
	delete(h.clients, id)
	h.mu.Unlock()
}

func (h *Handler) GetClient(id uuid.UUID) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client := h.clients[id]
	return client
}

func (h *Handler) ConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Handler) Broadcast(message *Envelope) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		err := h.clients[client].Send(message)
		if err != nil {
			log.Errorf("Broadcast to %v failed: %v", client.String(), err)
		}
	}
	return nil
}

func (h *Handler) handler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		defer c.Close()
		auth := c.Locals("auth").(*auth.UserToken)

		id := uuid.New()
		client := &Client{
			id:      id,
			conn:    c,
			user:    auth,
			alive:   true,
			readCh:  make(chan read),
			seq:     1,
			writeMu: &sync.Mutex{},
		}

		h.addClient(client)
		defer func() {
			client.alive = false
			h.removeClient(id)
		}()

		ctx := context.WithValue(context.Background(), CtxWsClient, client)
		ctx = context.WithValue(ctx, CtxWsClientID, id)
		ctx, cancel := context.WithDeadlineCause(ctx, auth.Expires(), ErrAuthExpired)
		defer cancel()

		c.SetReadLimit(1 << 20)
		c.SetReadDeadline(auth.Expires())
		c.SetPongHandler(func(appData string) error {
			return nil
		})

		go client.readLoop()
		h.router.onConnect(ctx, client)
		defer h.router.onDisconnect(ctx, client)

		for {
			select {
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					client.Send(&Envelope{
						Type:      "close",
						MessageID: MessageID(uuid.New()),
						Timestamp: time.Now(),
					})
					return // Auth expired
				}
				return
			case r := <-client.readCh:
				if websocket.IsCloseError(r.err, websocket.CloseNormalClosure) {
					return
				} else if r.err != nil {
					log.Infof("Read error from %v: %v", client.id.String(), r.err)
					return
				}

				if err := h.onMessage(ctx, r.messageType, r.message); err != nil {
					log.Infof("Message handling error from %v: %v", client.id.String(), err)
					return
				}
			}
		}
	})
}

func (h *Handler) onMessage(ctx context.Context, messageType int, message []byte) error {
	switch messageType {
	case websocket.TextMessage, websocket.BinaryMessage:
		var data Envelope
		err := json.Unmarshal(message, &data)
		if err != nil {
			log.Info("Invalid JSON")
			return nil
		}
		client := GetClient(ctx)
		err = h.router.onMessage(ctx, client, &data)
		if err != nil {
			log.Info("Invalid routing")
			return nil
		}
		return nil
	default:
		return nil
	}
}

func GetClient(ctx context.Context) *Client {
	client, ok := ctx.Value(CtxWsClient).(*Client)
	if !ok {
		return nil
	}
	return client
}

func GetClientID(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(CtxWsClientID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
