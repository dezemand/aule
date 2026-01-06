package wsproto

import "context"

const CtxWsMessage = "ws_message"

type HandlerFunc func(ctx context.Context, message *Envelope) error

type Router struct {
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) On(messageType string, handler HandlerFunc) {
}

func (r *Router) OnConnect(handler func(ctx context.Context) error) {
}

func (r *Router) OnDisconnect(handler func(ctx context.Context) error) {

}

func (r *Router) onConnect(ctx context.Context) {
}

func (r *Router) onMessage(ctx context.Context, message *Envelope) error {
	ctx = context.WithValue(ctx, CtxWsMessage, message)
	return nil
}

func (r *Router) onDisconnect(ctx context.Context) {

}

func GetMessage(ctx context.Context) *Envelope {
	msg, ok := ctx.Value(CtxWsMessage).(*Envelope)
	if !ok {
		return nil
	}
	return msg
}
