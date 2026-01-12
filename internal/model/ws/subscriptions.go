// Package modelsws defines WebSocket message types.
package modelsws

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Message type constants for subscription operations.
const (
	MsgTypeSubscribe      = "subscription.subscribe.req"
	MsgTypeSubscribeAck   = "subscription.subscribe.ack"
	MsgTypeUnsubscribe    = "subscription.unsubscribe.req"
	MsgTypeUnsubscribeAck = "subscription.unsubscribe.ack"
)

// SubscribeMsg requests a subscription to a topic.
type SubscribeMsg struct {
	Topic   string          `json:"topic"`
	Query   json.RawMessage `json:"query,omitempty"`
	Initial bool            `json:"initial"`
}

func (m *SubscribeMsg) Type() string { return MsgTypeSubscribe }

// DecodeQuery decodes the query JSON into the provided struct.
func (m *SubscribeMsg) DecodeQuery(v any) error {
	if len(m.Query) == 0 {
		return nil
	}
	return json.Unmarshal(m.Query, v)
}

// SubscribeAckMsg acknowledges a successful subscription.
type SubscribeAckMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

func (m *SubscribeAckMsg) Type() string { return MsgTypeSubscribeAck }

// UnsubscribeMsg requests unsubscription from a topic.
type UnsubscribeMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

func (m *UnsubscribeMsg) Type() string { return MsgTypeUnsubscribe }

// UnsubscribeAckMsg acknowledges a successful unsubscription.
type UnsubscribeAckMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

func (m *UnsubscribeAckMsg) Type() string { return MsgTypeUnsubscribeAck }
