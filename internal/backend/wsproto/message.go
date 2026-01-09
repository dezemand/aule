package wsproto

import "github.com/google/uuid"

type WsMsg interface {
	Type() string
}

type Message interface {
	Type() string
	Err() error
	IdempotencyKey() string
}

type messageImpl struct {
	id     uuid.UUID
	origId uuid.UUID
	reqId  uuid.UUID
}

type wsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

func FromError(err error) WsMsg {
	return &wsError{
		Code:    "error.internal",
		Message: err.Error(),
	}
}

func Error(code string, message string, detail ...any) WsMsg {
	var d any
	if len(detail) > 0 {
		d = detail[0]
	}
	return &wsError{
		Code:    code,
		Message: message,
		Detail:  d,
	}
}

func (m *wsError) Type() string {
	return "error"
}
