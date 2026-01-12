// Package modelsws defines WebSocket message types.
package modelsws

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Common message type constants.
const (
	MsgTypeError = "error"
)

// Envelope is the wire format for WebSocket messages.
type Envelope struct {
	Type           string          `json:"type"`
	MessageID      uuid.UUID       `json:"id"`
	RequestID      *uuid.UUID      `json:"reply_to,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	SubscriptionID *uuid.UUID      `json:"subscription_id,omitempty"`
	Seq            int64           `json:"seq,omitempty"`
	Timestamp      time.Time       `json:"time"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

// ErrorPayload is the standard error response format.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

// NewEnvelope creates a new envelope with the given type and payload.
func NewEnvelope(typ string, payload any) (*Envelope, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = b
	}

	return &Envelope{
		Type:      typ,
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Payload:   raw,
	}, nil
}

// NewErrorEnvelope creates an error response envelope.
func NewErrorEnvelope(code, message string, detail any) (*Envelope, error) {
	return NewEnvelope("error", &ErrorPayload{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
}

// Reply creates a response envelope linked to this message.
func (e *Envelope) Reply(typ string, payload any) (*Envelope, error) {
	env, err := NewEnvelope(typ, payload)
	if err != nil {
		return nil, err
	}
	env.RequestID = &e.MessageID
	return env, nil
}

// ReplyError creates an error response linked to this message.
func (e *Envelope) ReplyError(code, message string, detail any) (*Envelope, error) {
	env, err := NewErrorEnvelope(code, message, detail)
	if err != nil {
		return nil, err
	}
	env.RequestID = &e.MessageID
	return env, nil
}

// DecodePayload unmarshals the payload into the provided struct.
func (e *Envelope) DecodePayload(v any) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

// WithSubscription returns a copy with the subscription ID set.
func (e *Envelope) WithSubscription(subID uuid.UUID) *Envelope {
	copy := *e
	copy.SubscriptionID = &subID
	return &copy
}
