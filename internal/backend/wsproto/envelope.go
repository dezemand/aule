package wsproto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Status string
type MessageID uuid.UUID

const (
	StatusOK  Status = ""
	StatusErr Status = "error"
)

type Envelope struct {
	Status         Status          `json:"status,omitempty"`
	Type           string          `json:"type"`
	MessageID      MessageID       `json:"message_id"`
	RequestID      MessageID       `json:"request_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
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
		Status:    StatusOK,
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
		Status:    StatusErr,
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
