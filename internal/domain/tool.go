package domain

import "github.com/google/uuid"

type ToolID uuid.UUID

type ToolType string

const (
	ToolTypeInternal ToolType = "internal"
	ToolTypeExternal ToolType = "external"
)

type Tool struct {
	ID   ToolID   `json:"id"`
	Type ToolType `json:"type"`
	Name string   `json:"name"`
}
