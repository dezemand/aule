package wsproto

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2/log"
)

type HandlerFunc func(ctx Ctx) error
type ConnectFunc func(ctx Ctx) error
type DisconnectFunc func(ctx Ctx) error

type Router struct {
	middleware         []HandlerFunc
	handlers           map[string][]HandlerFunc
	connectHandlers    []ConnectFunc
	disconnectHandlers []DisconnectFunc
}

func NewRouter() *Router {
	return &Router{
		middleware:         make([]HandlerFunc, 0),
		handlers:           make(map[string][]HandlerFunc),
		connectHandlers:    make([]ConnectFunc, 0),
		disconnectHandlers: make([]DisconnectFunc, 0),
	}
}

// Use registers middleware that runs for all message handlers.
// Middleware is executed in the order it is registered.
// Call ctx.Next() to continue to the next middleware/handler.
func (r *Router) Use(handlers ...HandlerFunc) {
	r.middleware = append(r.middleware, handlers...)
}

// On registers handlers for a specific message type.
// Multiple handlers can be registered and will be chained.
// Call ctx.Next() to continue to the next handler.
func (r *Router) On(messageType string, handlers ...HandlerFunc) {
	if len(handlers) == 0 {
		return
	}
	r.handlers[messageType] = append(r.handlers[messageType], handlers...)
}

// OnConnect registers a handler that runs when a client connects.
func (r *Router) OnConnect(handler ConnectFunc) {
	r.connectHandlers = append(r.connectHandlers, handler)
}

// OnDisconnect registers a handler that runs when a client disconnects.
func (r *Router) OnDisconnect(handler DisconnectFunc) {
	r.disconnectHandlers = append(r.disconnectHandlers, handler)
}

func (r *Router) onConnect(ctx context.Context, client *Client) {
	c := connectCtx(ctx, client)

	for _, handler := range r.connectHandlers {
		if err := handler(c); err != nil {
			log.Errorf("connect handler error: %v", err)
		}
	}
}

func (r *Router) onMessage(ctx context.Context, client *Client, message *Envelope) error {
	handlers, exists := r.handlers[message.Type]
	if !exists {
		c := newChainCtx(ctx, client, message, nil)
		defer c.Release()
		return c.ReplyError(
			"error.unknown_message",
			fmt.Sprintf("no handler for message type: %s", message.Type),
			nil,
		)
	}

	// Build the handler chain: middleware + route handlers
	chain := make([]HandlerFunc, 0, len(r.middleware)+len(handlers))
	chain = append(chain, r.middleware...)
	chain = append(chain, handlers...)

	c := newChainCtx(ctx, client, message, chain)
	defer c.Release()

	// Start the chain
	if err := c.Next(); err != nil {
		log.Errorf("handler error for %s: %v", message.Type, err)
		return c.Reply(FromError(err))
	}

	return nil
}

func (r *Router) onDisconnect(ctx context.Context, client *Client) {
	c := disconnectCtx(ctx, client)

	for _, handler := range r.disconnectHandlers {
		if err := handler(c); err != nil {
			log.Errorf("disconnect handler error: %v", err)
		}
	}
}
