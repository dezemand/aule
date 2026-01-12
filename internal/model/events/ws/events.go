// Package eventsws defines WebSocket event types for the event bus.
package eventsws

import (
	"encoding/json"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/google/uuid"
)

// Topics for WebSocket lifecycle events.
var (
	TopicConnect    = event.NewTopic[ConnectEvent]("ws.connect")
	TopicDisconnect = event.NewTopic[DisconnectEvent]("ws.disconnect")
	TopicIncoming   = event.NewTopic[IncomingEvent]("ws.message.incoming")
	TopicOutgoing   = event.NewTopic[OutgoingEvent]("ws.message.outgoing")
)

// WsEvent is the base type for all WebSocket events.
type WsEvent struct {
	ClientID uuid.UUID
	UserID   domain.UserID
}

// ConnectEvent is published when a client connects.
type ConnectEvent struct {
	WsEvent
}

// DisconnectEvent is published when a client disconnects.
type DisconnectEvent struct {
	WsEvent
	Code   int    // WebSocket close code
	Reason string // Close reason if provided
}

// IncomingEvent is published when a message is received from a client.
type IncomingEvent struct {
	WsEvent
	Message *modelsws.Envelope
}

// OutgoingEvent is published to send a message to clients.
type OutgoingEvent struct {
	To             []OutgoingTo
	Type           string
	Payload        json.RawMessage
	ReplyTo        *uuid.UUID
	SubscriptionID *uuid.UUID
}

// OutgoingTo specifies a target for an outgoing message.
type OutgoingTo struct {
	Type string // ClientID, UserID
	ID   uuid.UUID
}
