package wsproto

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusOK  Status = "ok"
	StatusErr Status = "error"
)

type Envelope struct {
	Status         Status          `json:"status"`
	Kind           string          `json:"kind"`
	Type           string          `json:"type"`
	MessageID      string          `json:"message_id"`
	RequestID      string          `json:"request_id,omitempty"`
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

func ToEnvelope(kind, typ, messageID string, payload any) (*Envelope, error) {
	b, _ := json.Marshal(payload)
	return &Envelope{
		Status:    StatusOK,
		Kind:      kind,
		Type:      typ,
		MessageID: messageID,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(b),
	}, nil
}

func ToErrorEnvelope(kind, typ, messageID string, code, message string, detail any) (*Envelope, error) {
	b, _ := json.Marshal(&EnvelopeErr{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
	return &Envelope{
		Status:    StatusErr,
		Kind:      kind,
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
