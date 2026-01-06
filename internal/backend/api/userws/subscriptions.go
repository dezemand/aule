package userws

import (
	"context"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

type subClient struct {
	id   uuid.UUID
	subs map[string]struct{}
}

type SubscriptionsHandler struct {
	clients map[uuid.UUID]*subClient
}

func NewSubscriptionsHandler() *SubscriptionsHandler {
	return &SubscriptionsHandler{}
}

type SubscribeMessage struct {
}

type SubscribeAckMessage struct {
	ID uuid.UUID `json:"id"`
}

func (h *SubscriptionsHandler) OnSubscribe(ctx context.Context, msg *wsproto.Envelope) error {
	var subscribeMsg SubscribeMessage
	if err := msg.DecodePayload(&subscribeMsg); err != nil {
		return err
	}

	subId := uuid.New()

	ack, err := wsproto.ToEnvelope(
		"subscribe_ack",
		wsproto.MessageID(uuid.New()),
		&SubscribeAckMessage{
			ID: subId,
		},
	)
	if err != nil {
		return err
	}

	ack.RequestID = msg.MessageID
	wsproto.GetClient(ctx).Send(ack)

	return nil
}

func (h *SubscriptionsHandler) OnUnsubscribe(ctx context.Context, msg *wsproto.Envelope) error {
	return nil
}

func (h *SubscriptionsHandler) OnClientConnect(ctx context.Context) error {
	id := wsproto.GetClientID(ctx)
	h.clients[id] = &subClient{
		id:   id,
		subs: make(map[string]struct{}),
	}

	return nil
}

func (h *SubscriptionsHandler) OnClientDisconnect(ctx context.Context) error {
	id := wsproto.GetClientID(ctx)
	delete(h.clients, id)

	return nil
}
