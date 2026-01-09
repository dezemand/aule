package wsproto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Status string
type MessageID uuid.UUID

func NewMessageID() MessageID {
	return MessageID(uuid.New())
}

type Envelope struct {
	Type           string          `json:"type"`
	MessageID      MessageID       `json:"id"`
	RequestID      *MessageID      `json:"reply_to,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	SubscriptionID *uuid.UUID      `json:"subscription_id,omitempty"`
	Seq            int64           `json:"seq,omitempty"`
	Timestamp      time.Time       `json:"time"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

type EnvelopeErr struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

func ToEnvelope(typ string, messageID MessageID, payload any) (*Envelope, error) {
	b, _ := json.Marshal(payload)
	return &Envelope{
		Type:      typ,
		MessageID: messageID,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(b),
	}, nil
}

func ToErrorEnvelope(typ string, messageID MessageID, code, message string, detail any) (*Envelope, error) {
	b, _ := json.Marshal(&EnvelopeErr{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
	return &Envelope{
		Type:      typ,
		MessageID: messageID,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(b),
	}, nil
}

func (e *Envelope) DecodePayload(v any) error {
	if len(e.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(e.Payload, v)
}

func (id MessageID) MarshalJSON() ([]byte, error) {
	u := uuid.UUID(id)
	return json.Marshal(u.String())
}

func (id *MessageID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = MessageID(u)
	return nil
}

func (id MessageID) String() string {
	u := uuid.UUID(id)
	return u.String()
}
