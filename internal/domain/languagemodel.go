package domain

import "github.com/google/uuid"

type LanguageModelID uuid.UUID

type LanguageModel struct {
	ID   LanguageModelID `json:"id"`
	Name string          `json:"name"`
}
