package eventsws

import (
	"encoding/json"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	"github.com/google/uuid"
)

// Topics for WebSocket subscription events.
var (
	TopicSubscribe      = event.NewTopic[SubscribeEvent]("ws.subscribe.request")
	TopicSubscribeAck   = event.NewTopic[SubscribeAckEvent]("ws.subscribe.ack")
	TopicSubscribeError = event.NewTopic[SubscribeErrorEvent]("ws.subscribe.error")
	TopicUnsubscribe    = event.NewTopic[UnsubscribeEvent]("ws.unsubscribe.request")
	TopicUnsubscribeAck = event.NewTopic[UnsubscribeAckEvent]("ws.unsubscribe.ack")
)

// SubscribeEvent is published when a client requests a subscription.
type SubscribeEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	Topic        string
	Query        json.RawMessage
	Initial      bool
}

// SubscribeAckEvent is published to acknowledge a subscription.
type SubscribeAckEvent struct {
	ClientID       uuid.UUID
	SubscriptionID uuid.UUID
	RequestMsgID   uuid.UUID
}

// SubscribeErrorEvent is published when a subscription fails.
type SubscribeErrorEvent struct {
	ClientID     uuid.UUID
	RequestMsgID uuid.UUID
	Code         string
	Message      string
}

// UnsubscribeEvent is published when a client requests to unsubscribe.
type UnsubscribeEvent struct {
	ClientID       uuid.UUID
	SubscriptionID uuid.UUID
	RequestMsgID   uuid.UUID
}

// UnsubscribeAckEvent is published to acknowledge an unsubscription.
type UnsubscribeAckEvent struct {
	ClientID       uuid.UUID
	SubscriptionID uuid.UUID
	RequestMsgID   uuid.UUID
}
