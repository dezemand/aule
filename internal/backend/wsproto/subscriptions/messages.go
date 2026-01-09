package wssubscriptions

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	MsgTypeSubscribe      = "subscription.subscribe.req"
	MsgTypeSubscribeAck   = "subscription.subscribe.ack"
	MsgTypeUnsubscribe    = "subscription.unsubscribe.req"
	MsgTypeUnsubscribeAck = "subscription.unsubscribe.ack"
)

type SubscribeMsg struct {
	Topic   string          `json:"topic"`
	Query   json.RawMessage `json:"query,omitempty"`
	Initial bool            `json:"initial"`
}

type SubscribeAckMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

type UnsubscribeMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

type UnsubscribeAckMsg struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
}

func (e *SubscribeMsg) DecodeQuery(v any) error {
	if len(e.Query) == 0 {
		return nil
	}
	return json.Unmarshal(e.Query, v)
}

func (m *SubscribeMsg) Type() string {
	return MsgTypeSubscribe
}

func (e *UnsubscribeMsg) Type() string {
	return MsgTypeUnsubscribe
}

func (e *SubscribeAckMsg) Type() string {
	return MsgTypeSubscribeAck
}

func (e *UnsubscribeAckMsg) Type() string {
	return MsgTypeUnsubscribeAck
}
