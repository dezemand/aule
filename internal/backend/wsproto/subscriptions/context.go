package wssubscriptions

import (
	"context"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

type subCtx struct {
	client *wsproto.Client
	subID  uuid.UUID
	req    any
}

func (s *subCtx) Body(v wsproto.WsMsg) error {

	return nil
}

func (s *subCtx) Client() *wsproto.Client {
	return s.client
}

func (s *subCtx) Context() context.Context {
	return context.Background()
}

func (s *subCtx) Event() wsproto.WsEvent {
	return wsproto.WsEventMessage
}

func (s *subCtx) Locals(key string) any {
	return nil
}

func (s *subCtx) Message() *wsproto.Envelope {
	return nil
}

func (s *subCtx) Next() error {
	return nil
}

func (s *subCtx) Reply(msg wsproto.WsMsg) error {
	return s.Send(msg)
}

func (s *subCtx) ReplyError(code string, message string, detail any) error {
	env, err := wsproto.ToErrorEnvelope("error", wsproto.NewMessageID(), code, message, detail)
	if err != nil {
		return err
	}
	env.SubscriptionID = &s.subID
	return s.client.Send(env)
}

func (s *subCtx) Send(msg wsproto.WsMsg) error {
	env, err := wsproto.ToEnvelope(msg.Type(), wsproto.NewMessageID(), msg)
	if err != nil {
		return err
	}
	env.SubscriptionID = &s.subID
	return s.client.Send(env)
}

func (s *subCtx) SetLocals(key string, value any) {
	// No-op: subscription context doesn't support locals
}

func (s *subCtx) Release() {
}

func newSubCtx(client *wsproto.Client, subID uuid.UUID, req any) wsproto.Ctx {
	return &subCtx{
		client: client,
		subID:  subID,
		req:    req,
	}
}
