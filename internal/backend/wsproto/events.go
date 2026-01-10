package wsproto

import (
	"context"
	"time"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/eventhandler"
	"github.com/google/uuid"
)

var TopicConnected = eventhandler.NewTopic[ConnectedEvent]("websocket.connected")
var TopicDisconnected = eventhandler.NewTopic[DisconnectedEvent]("websocket.disconnected")
var TopicIncoming = eventhandler.NewTopic[IncomingEvent]("websocket.incoming")
var TopicOutgoing = eventhandler.NewTopic[IncomingEvent]("websocket.outgoing")

type DisconnectedEvent struct {
	Reason int
}

type ConnectedEvent struct {
}

type IncomingEvent struct {
	Message Envelope
}

type OutgoingEvent struct {
	Message Envelope
}

type ClientMetadata interface {
	eventhandler.Metadata
	ClientID() uuid.UUID
	UserID() domain.UserID
	TimeSent() time.Time
	OriginalMessageID() uuid.UUID
	IdempotencyKey() string
}

type clientMetadata struct {
	messageID     uuid.UUID
	origMessageID uuid.UUID
	timeReceived  time.Time
	timeSent      time.Time
	clientID      uuid.UUID
	userID        domain.UserID
}

type clientEvent[T any] struct {
	ctx     context.Context
	topic   eventhandler.Topic[T]
	payload T
}

func (c *clientEvent[T]) Context() context.Context {
	return c.ctx
}

func (c *clientEvent[T]) Metadata() eventhandler.Metadata {
	panic("unimplemented")
}

func (c *clientEvent[T]) Payload() T {
	return c.payload
}

func (c *clientEvent[T]) Topic() eventhandler.Topic[T] {
	return c.topic
}

func NewEvent[T any](ctx context.Context, topic eventhandler.Topic[T], client *Client, payload T) eventhandler.Event[T] {
	return &clientEvent[T]{
		topic:   topic,
		ctx:     ctx,
		payload: payload,
		metadata: &clientMetadata{
			messageID:     uuid.New(),
			origMessageID: GetClientMessageID(ctx),
			timeReceived:  time.Now().UTC(),
			timeSent:      time.Now().UTC(),
			clientID:      client.id,
			userID:        client.userID,
		}
	}
}
