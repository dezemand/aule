package domain

import "github.com/google/uuid"

type LLMProvider struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type LLModel struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}
