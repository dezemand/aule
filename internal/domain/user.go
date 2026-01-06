package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

type UserID uuid.UUID

type User struct {
	ID    UserID `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UserLogin struct {
	Provider string `json:"provider"`
	Sub      string `json:"sub"`
	UserID   UserID `json:"user_id"`
}

func (id UserID) MarshalJSON() ([]byte, error) {
	u := uuid.UUID(id)
	return json.Marshal(u.String())
}

func (id *UserID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = UserID(u)
	return nil
}

func (id UserID) String() string {
	u := uuid.UUID(id)
	return u.String()
}
