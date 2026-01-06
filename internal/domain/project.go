package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProjectID uuid.UUID

type Project struct {
	ID          ProjectID
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Goal        string `json:"goal,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectMemberRole string

const (
	ProjectMemberRoleOwner ProjectMemberRole = "owner"
)

type ProjectMember struct {
	ID        uuid.UUID
	ProjectID ProjectID
	UserID    UserID
	Role      ProjectMemberRole `json:"role"`
}

func (id ProjectID) MarshalJSON() ([]byte, error) {
	u := uuid.UUID(id)
	return json.Marshal(u.String())
}

func (id *ProjectID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = ProjectID(u)
	return nil
}

func (id ProjectID) String() string {
	u := uuid.UUID(id)
	return u.String()
}
