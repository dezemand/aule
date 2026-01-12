package wssubscriptions

import (
	"context"
	"encoding/json"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/dezemandje/aule/internal/event"
	eventsws "github.com/dezemandje/aule/internal/model/events/ws"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
)

// Handler handles WebSocket subscription lifecycle via event bus.
type Handler struct {
	bus       *event.Bus
	service   *Service
	wsHandler *wsproto.Handler
	subs      []event.Subscription
}

// NewHandler creates a new subscription handler.
func NewHandler(bus *event.Bus, service *Service, wsHandler *wsproto.Handler) *Handler {
	return &Handler{
		bus:       bus,
		service:   service,
		wsHandler: wsHandler,
	}
}

// SetupEventHandlers registers handlers for subscription-related events.
func (h *Handler) SetupEventHandlers() {
	// Clean up subscriptions when clients disconnect
	h.subs = []event.Subscription{
		event.Subscribe(h.bus, eventsws.TopicDisconnect, h.handleDisconnect),
		event.Subscribe(h.bus, eventsws.TopicSubscribe, h.handleSubscribe),
		event.Subscribe(h.bus, eventsws.TopicUnsubscribe, h.handleUnsubscribe),

		wsproto.WsToEvent[modelsws.SubscribeMsg, eventsws.SubscribeEvent](
			h.bus,
			modelsws.MsgTypeSubscribe,
			eventsws.TopicSubscribe,
			func(payload modelsws.SubscribeMsg, evt event.Event[eventsws.IncomingEvent]) eventsws.SubscribeEvent {
				return eventsws.SubscribeEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					Topic:        payload.Topic,
					Query:        payload.Query,
					Initial:      payload.Initial,
				}
			}),

		wsproto.WsToEvent[modelsws.UnsubscribeMsg, eventsws.UnsubscribeEvent](
			h.bus,
			modelsws.MsgTypeUnsubscribe,
			eventsws.TopicUnsubscribe,
			func(payload modelsws.UnsubscribeMsg, evt event.Event[eventsws.IncomingEvent]) eventsws.UnsubscribeEvent {
				return eventsws.UnsubscribeEvent{
					ClientID:       evt.Payload().ClientID,
					SubscriptionID: payload.SubscriptionID,
					RequestMsgID:   evt.Payload().Message.MessageID,
				}
			}),

		wsproto.EventToWs(h.bus, eventsws.TopicSubscribeAck, func(evt event.Event[eventsws.SubscribeAckEvent]) (*eventsws.OutgoingEvent, error) {
			payload, err := json.Marshal(modelsws.SubscribeAckMsg{
				SubscriptionID: evt.Payload().SubscriptionID,
			})
			if err != nil {
				return nil, err
			}
			req := evt.Payload().RequestMsgID
			return &eventsws.OutgoingEvent{
				To: []eventsws.OutgoingTo{
					{ID: evt.Payload().ClientID},
				},
				Type:    modelsws.MsgTypeSubscribeAck,
				Payload: payload,
				ReplyTo: &req,
			}, nil
		}),

		wsproto.EventToWs(h.bus, eventsws.TopicUnsubscribeAck, func(evt event.Event[eventsws.UnsubscribeAckEvent]) (*eventsws.OutgoingEvent, error) {
			payload, err := json.Marshal(modelsws.UnsubscribeAckMsg{
				SubscriptionID: evt.Payload().SubscriptionID,
			})
			if err != nil {
				return nil, err
			}
			req := evt.Payload().RequestMsgID
			return &eventsws.OutgoingEvent{
				To: []eventsws.OutgoingTo{
					{ID: evt.Payload().ClientID},
				},
				Type:    modelsws.MsgTypeUnsubscribeAck,
				Payload: json.RawMessage(payload),
				ReplyTo: &req,
			}, nil
		}),

		// Send error responses to clients
		wsproto.EventToWs(h.bus, eventsws.TopicSubscribeError, func(evt event.Event[eventsws.SubscribeErrorEvent]) (*eventsws.OutgoingEvent, error) {
			payload, err := json.Marshal(modelsws.ErrorPayload{
				Code:    evt.Payload().Code,
				Message: evt.Payload().Message,
			})
			if err != nil {
				return nil, err
			}
			req := evt.Payload().RequestMsgID
			return &eventsws.OutgoingEvent{
				To: []eventsws.OutgoingTo{
					{ID: evt.Payload().ClientID},
				},
				Type:    modelsws.MsgTypeError,
				Payload: payload,
				ReplyTo: &req,
			}, nil
		}),
	}

}

func (h *Handler) handleDisconnect(ctx context.Context, e event.Event[eventsws.DisconnectEvent]) error {
	return h.service.UnsubscribeAll(e.Payload().ClientID)
}

func (h *Handler) handleSubscribe(ctx context.Context, evt event.Event[eventsws.SubscribeEvent]) error {
	client, ok := h.wsHandler.GetClient(evt.Payload().ClientID)
	if !ok {
		// Client not found; ignore
		return nil
	}

	sub, err := h.service.Subscribe(client, evt.Payload().Topic, evt.Payload().Query)
	if err != nil {
		// Send error back to client
		event.Publish(h.bus, eventsws.TopicSubscribeError.Event(eventsws.SubscribeErrorEvent{
			ClientID:     evt.Payload().ClientID,
			RequestMsgID: evt.Payload().RequestMsgID,
			Code:         "subscription_failed",
			Message:      err.Error(),
		}, event.WithSource(evt)))
		return nil // Don't propagate error, we've handled it
	}

	ackEvt := eventsws.TopicSubscribeAck.Event(eventsws.SubscribeAckEvent{
		ClientID:       evt.Payload().ClientID,
		SubscriptionID: sub.ID(),
		RequestMsgID:   evt.Payload().RequestMsgID,
	}, event.WithSource(evt))

	event.Publish(h.bus, ackEvt)

	if evt.Payload().Initial {
		if err := h.service.SendInitial(sub); err != nil {
			// Log error but don't fail - subscription is still valid
			// The client will get the subscription ack and can retry initial fetch
			_ = err
		}
	}

	return nil
}

// HandleUnsubscribe processes an unsubscribe request.
func (h *Handler) handleUnsubscribe(ctx context.Context, evt event.Event[eventsws.UnsubscribeEvent]) error {
	if err := h.service.Unsubscribe(evt.Payload().ClientID, evt.Payload().SubscriptionID); err != nil {
		return err
	}

	event.Publish(h.bus, eventsws.TopicUnsubscribeAck.Event(eventsws.UnsubscribeAckEvent{
		ClientID:       evt.Payload().ClientID,
		SubscriptionID: evt.Payload().SubscriptionID,
		RequestMsgID:   evt.Payload().RequestMsgID,
	}, event.WithSource(evt)))

	return nil
}
