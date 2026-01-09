package wsproto

import (
	"context"
	"errors"
)

var ErrMessageTypeMismatch = errors.New("message type mismatch")

type ctxKey string

const CtxWsMessage ctxKey = "ws_message"
const CtxWsEvent ctxKey = "ws_event"

type WsEvent string

const (
	WsEventConnect    WsEvent = "connect"
	WsEventDisconnect WsEvent = "disconnect"
	WsEventMessage    WsEvent = "message"
)

// Ctx is the context passed to WebSocket handlers.
// It provides access to the client, message, and methods for sending responses.
type Ctx interface {
	// Event returns the type of event (connect, disconnect, message).
	Event() WsEvent

	// Client returns the WebSocket client.
	Client() *Client

	// Context returns the underlying context.Context.
	Context() context.Context

	// Message returns the incoming message envelope (nil for connect/disconnect).
	Message() *Envelope

	// Body decodes the message payload into the provided struct.
	// The struct must implement WsMsg and match the message type.
	Body(v WsMsg) error

	// Send sends a message to the client.
	Send(msg WsMsg) error

	// Reply sends a response message to the client, linking it to the incoming message.
	Reply(msg WsMsg) error

	// ReplyError sends an error response to the client.
	ReplyError(code string, message string, detail any) error

	// Next calls the next handler in the chain.
	// Must be called in middleware to continue processing.
	Next() error

	// Locals returns a value stored in the context, or nil if not found.
	Locals(key string) any

	// SetLocals stores a value in the context.
	SetLocals(key string, value any)

	// release cleans up the context (called internally).
	Release()
}

// chainCtx implements Ctx with middleware chain support.
type chainCtx struct {
	event   WsEvent
	client  *Client
	message *Envelope
	ctx     context.Context
	chain   []HandlerFunc
	index   int
	locals  map[string]any
}

func newChainCtx(ctx context.Context, client *Client, message *Envelope, chain []HandlerFunc) *chainCtx {
	return &chainCtx{
		event:   WsEventMessage,
		client:  client,
		message: message,
		ctx:     context.WithValue(ctx, CtxWsMessage, message),
		chain:   chain,
		index:   0,
		locals:  make(map[string]any),
	}
}

func (c *chainCtx) Event() WsEvent {
	return c.event
}

func (c *chainCtx) Client() *Client {
	return c.client
}

func (c *chainCtx) Context() context.Context {
	return c.ctx
}

func (c *chainCtx) Message() *Envelope {
	return c.message
}

func (c *chainCtx) Body(v WsMsg) error {
	if c.message == nil {
		return errors.New("no message available")
	}
	if v.Type() != c.message.Type {
		return ErrMessageTypeMismatch
	}
	return c.message.DecodePayload(v)
}

func (c *chainCtx) Send(msg WsMsg) error {
	env, err := ToEnvelope(msg.Type(), NewMessageID(), msg)
	if err != nil {
		return err
	}
	return c.client.Send(env)
}

func (c *chainCtx) Reply(msg WsMsg) error {
	env, err := ToEnvelope(msg.Type(), NewMessageID(), msg)
	if err != nil {
		return err
	}
	if c.message != nil {
		reqID := c.message.MessageID
		env.RequestID = &reqID
	}
	return c.client.Send(env)
}

func (c *chainCtx) ReplyError(code string, message string, detail any) error {
	env, err := ToErrorEnvelope("error", NewMessageID(), code, message, detail)
	if err != nil {
		return err
	}
	if c.message != nil {
		reqID := c.message.MessageID
		env.RequestID = &reqID
	}
	return c.client.Send(env)
}

func (c *chainCtx) Next() error {
	if c.index >= len(c.chain) {
		return nil
	}
	handler := c.chain[c.index]
	c.index++
	return handler(c)
}

func (c *chainCtx) Locals(key string) any {
	return c.locals[key]
}

func (c *chainCtx) SetLocals(key string, value any) {
	c.locals[key] = value
}

func (c *chainCtx) Release() {
	// Cleanup if needed
}

// simpleCtx is used for connect/disconnect events (no chain).
type simpleCtx struct {
	event  WsEvent
	client *Client
	ctx    context.Context
	locals map[string]any
}

func connectCtx(ctx context.Context, client *Client) *simpleCtx {
	return &simpleCtx{
		event:  WsEventConnect,
		client: client,
		ctx:    context.WithValue(ctx, CtxWsEvent, WsEventConnect),
		locals: make(map[string]any),
	}
}

func disconnectCtx(ctx context.Context, client *Client) *simpleCtx {
	return &simpleCtx{
		event:  WsEventDisconnect,
		client: client,
		ctx:    context.WithValue(ctx, CtxWsEvent, WsEventDisconnect),
		locals: make(map[string]any),
	}
}

func (c *simpleCtx) Event() WsEvent {
	return c.event
}

func (c *simpleCtx) Client() *Client {
	return c.client
}

func (c *simpleCtx) Context() context.Context {
	return c.ctx
}

func (c *simpleCtx) Message() *Envelope {
	return nil
}

func (c *simpleCtx) Body(v WsMsg) error {
	return errors.New("no message available for connect/disconnect events")
}

func (c *simpleCtx) Send(msg WsMsg) error {
	env, err := ToEnvelope(msg.Type(), NewMessageID(), msg)
	if err != nil {
		return err
	}
	return c.client.Send(env)
}

func (c *simpleCtx) Reply(msg WsMsg) error {
	// No incoming message to reply to
	return c.Send(msg)
}

func (c *simpleCtx) ReplyError(code string, message string, detail any) error {
	env, err := ToErrorEnvelope("error", NewMessageID(), code, message, detail)
	if err != nil {
		return err
	}
	return c.client.Send(env)
}

func (c *simpleCtx) Next() error {
	// No chain for connect/disconnect
	return nil
}

func (c *simpleCtx) Locals(key string) any {
	return c.locals[key]
}

func (c *simpleCtx) SetLocals(key string, value any) {
	c.locals[key] = value
}

func (c *simpleCtx) Release() {
	// Cleanup if needed
}
