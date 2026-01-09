package wssubscriptions

import (
	"github.com/dezemandje/aule/internal/backend/wsproto"
)

type Handler struct {
	service *Service
}

func (h *Handler) OnSubscribe(c wsproto.Ctx) error {
	var body SubscribeMsg
	if err := c.Body(&body); err != nil {
		return c.Reply(wsproto.Error("error.invalid_payload", "invalid payload", err.Error()))
	}

	sub, err := h.service.subscribe(c.Client(), body.Topic, body.Query)
	if err != nil {
		return c.Reply(wsproto.FromError(err))
	}

	err = c.Reply(&SubscribeAckMsg{
		SubscriptionID: sub.ID(),
	})

	if body.Initial {
		h.service.sendInitial(c.Client(), sub)
	}

	return err
}

func (h *Handler) OnUnsubscribe(c wsproto.Ctx) error {
	var body UnsubscribeMsg
	if err := c.Body(&body); err != nil {
		return c.Reply(wsproto.Error("error.invalid_payload", "invalid payload", err.Error()))
	}

	if err := h.service.unsubscribe(c.Client().ID(), body.SubscriptionID); err != nil {
		return c.Reply(wsproto.FromError(err))
	}

	return c.Reply(&UnsubscribeAckMsg{
		SubscriptionID: body.SubscriptionID,
	})
}

func (h *Handler) OnClose(c wsproto.Ctx) error {
	h.service.unsubscribeAll(c.Client().ID())
	return nil
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}
