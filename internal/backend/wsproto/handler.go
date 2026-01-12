// Package wsproto provides WebSocket connection handling with event bus integration.
package wsproto

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/event"
	eventsws "github.com/dezemandje/aule/internal/model/events/ws"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"

	fasthttpwebsocket "github.com/fasthttp/websocket"
)

// ErrAuthExpired indicates the client's authentication has expired.
var ErrAuthExpired = errors.New("authentication expired")

// Context keys for accessing client info.
const (
	CtxWsClient   = "ws_client"
	CtxWsClientID = "ws_client_id"
)

// Handler manages WebSocket connections and publishes events to the bus.
type Handler struct {
	bus   *event.Bus
	store ClientStore
}

// NewHandler creates a new WebSocket handler.
func NewHandler(bus *event.Bus, store ClientStore) *Handler {
	h := &Handler{
		bus:   bus,
		store: store,
	}

	event.Subscribe(bus, eventsws.TopicOutgoing, h.handleOutgoingEvent)

	return h
}

// Handle is the Fiber handler for WebSocket upgrade requests.
func (h *Handler) Handle(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	user := auth.GetUser(c)
	if user == nil {
		return fiber.ErrUnauthorized
	}

	return h.wsHandler()(c)
}

func (h *Handler) handleOutgoingEvent(ctx context.Context, evt event.Event[eventsws.OutgoingEvent]) error {
	outgoing := evt.Payload()

	for _, to := range outgoing.To {
		client, ok := h.GetClient(to.ID)
		if !ok {
			continue
		}
		client.Send(&modelsws.Envelope{
			Type:           outgoing.Type,
			MessageID:      uuid.New(),
			Timestamp:      time.Now().UTC(),
			Payload:        outgoing.Payload,
			SubscriptionID: outgoing.SubscriptionID,
			RequestID:      outgoing.ReplyTo,
		})
	}

	return nil
}

// GetClient returns a client by ID, or nil if not found.
func (h *Handler) GetClient(id uuid.UUID) (*Client, bool) {
	return h.store.GetClient(id)
}

// publishConnect publishes a connect event.
func (h *Handler) publishConnect(client *Client) {
	if h.bus == nil {
		return
	}
	event.Publish(h.bus, event.NewEvent(eventsws.TopicConnect, eventsws.ConnectEvent{
		WsEvent: eventsws.WsEvent{
			ClientID: client.ID(),
			UserID:   client.UserID(),
		},
	}))
}

// publishDisconnect publishes a disconnect event.
func (h *Handler) publishDisconnect(client *Client, code int, reason string) {
	if h.bus == nil {
		return
	}
	event.Publish(h.bus, event.NewEvent(eventsws.TopicDisconnect, eventsws.DisconnectEvent{
		WsEvent: eventsws.WsEvent{
			ClientID: client.ID(),
			UserID:   client.UserID(),
		},
		Code:   code,
		Reason: reason,
	}))
}

// publishMessage publishes a message received event.
func (h *Handler) publishMessage(client *Client, envelope *modelsws.Envelope) {
	if h.bus == nil {
		return
	}
	event.Publish(h.bus, event.NewEvent(eventsws.TopicIncoming, eventsws.IncomingEvent{
		WsEvent: eventsws.WsEvent{
			ClientID: client.ID(),
			UserID:   client.UserID(),
		},
		Message: envelope,
	}))
}

func (h *Handler) wsHandler() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		defer c.Close()
		userToken := c.Locals("auth").(*auth.UserToken)

		id := uuid.New()
		client := &Client{
			id:      id,
			conn:    c,
			user:    userToken,
			alive:   true,
			readCh:  make(chan readResult),
			seq:     1,
			writeMu: &sync.Mutex{},
		}

		h.store.AddClient(client)
		defer func() {
			client.alive = false
			h.store.RemoveClient(id)
		}()

		ctx := context.WithValue(context.Background(), CtxWsClient, client)
		ctx = context.WithValue(ctx, CtxWsClientID, id)
		ctx, cancel := context.WithDeadlineCause(ctx, userToken.Expires(), ErrAuthExpired)
		defer cancel()

		c.SetReadLimit(1 << 20)
		c.SetReadDeadline(userToken.Expires())
		c.SetPongHandler(func(appData string) error {
			return nil
		})

		go client.readLoop()

		// Publish connect event
		h.publishConnect(client)

		var disconnectCode int
		var disconnectReason string

		for {
			select {
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					env, _ := modelsws.NewEnvelope("close", nil)
					client.Send(env)
					disconnectReason = "auth_expired"
				}
				h.publishDisconnect(client, disconnectCode, disconnectReason)
				return

			case r := <-client.readCh:
				if r.err != nil {
					if closeErr, ok := r.err.(*fasthttpwebsocket.CloseError); ok {
						disconnectCode = closeErr.Code
						disconnectReason = closeErr.Text
					}

					if websocket.IsCloseError(r.err, websocket.CloseNormalClosure) {
						disconnectReason = "normal_closure"
					} else {
						log.Infof("Read error from %v: %v", client.id.String(), r.err)
					}

					h.publishDisconnect(client, disconnectCode, disconnectReason)
					return
				}

				if err := h.onMessage(ctx, client, r.messageType, r.message); err != nil {
					log.Infof("Message handling error from %v: %v", client.id.String(), err)
					h.publishDisconnect(client, 0, "message_error")
					return
				}
			}
		}
	})
}

func (h *Handler) onMessage(ctx context.Context, client *Client, messageType int, message []byte) error {
	switch messageType {
	case websocket.TextMessage, websocket.BinaryMessage:
		var envelope modelsws.Envelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			log.Info("Invalid JSON from client")
			return nil
		}

		// Publish message event - handlers subscribe via event bus
		h.publishMessage(client, &envelope)
		return nil

	default:
		return nil
	}
}

// GetClient retrieves the WebSocket client from a context.
func GetClient(ctx context.Context) *Client {
	client, ok := ctx.Value(CtxWsClient).(*Client)
	if !ok {
		return nil
	}
	return client
}

// GetClientID retrieves the WebSocket client ID from a context.
func GetClientID(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(CtxWsClientID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

func WsToEvent[B any, T any](bus *event.Bus, messageType string, topic event.Topic[T], payloadFunc func(payload B, evt event.Event[eventsws.IncomingEvent]) T) event.Subscription {
	return event.Subscribe(bus, eventsws.TopicIncoming, func(ctx context.Context, evt event.Event[eventsws.IncomingEvent]) error {
		if evt.Payload().Message.Type != messageType {
			return nil
		}

		var payload B
		if err := json.Unmarshal(evt.Payload().Message.Payload, &payload); err != nil {
			return err
		}

		newEvt := topic.Event(payloadFunc(payload, evt), event.WithSource(evt))
		event.Publish(bus, newEvt)
		return nil
	})
}

func EventToWs[B any](bus *event.Bus, topic event.Topic[B], evFunc func(evt event.Event[B]) (*eventsws.OutgoingEvent, error)) event.Subscription {
	return event.Subscribe(bus, topic, func(ctx context.Context, evt event.Event[B]) error {
		outgoing, err := evFunc(evt)
		if err != nil {
			return err
		}
		event.Publish(bus, event.NewEvent(eventsws.TopicOutgoing, *outgoing))
		return nil
	})
}
