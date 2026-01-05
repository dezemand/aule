package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthRole string

const (
	RoleUser  AuthRole = "user"
	RoleAgent AuthRole = "agent"
)

func (r AuthRole) String() string {
	return string(r)
}

func ToAuthRole(roleStr string) (AuthRole, bool) {
	switch r := AuthRole(roleStr); r {
	case RoleUser, RoleAgent:
		return r, true
	default:
		return "", false
	}
}

type AuthToken interface {
	Role() AuthRole
	Token() *jwt.Token
	Expires() time.Time
}

type UserToken struct {
	id      string
	expires time.Time
}

type AgentToken struct {
	id      string
	expires time.Time
}

func newUserToken(id string) *UserToken {
	return &UserToken{
		id:      id,
		expires: time.Now().Add(time.Minute * 15),
	}
}

func newAgentToken(id string) *AgentToken {
	return &AgentToken{
		id:      id,
		expires: time.Now().Add(time.Hour * 24),
	}
}

func (ut *UserToken) Role() AuthRole {
	return RoleUser
}

func (at *AgentToken) Role() AuthRole {
	return RoleAgent
}

func (ut *UserToken) Expires() time.Time {
	return ut.expires
}

func (at *AgentToken) Expires() time.Time {
	return at.expires
}

func (ut *UserToken) ID() string {
	return ut.id
}

func (at *AgentToken) ID() string {
	return at.id
}

func (ut *UserToken) Token() *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": ut.Role(),
		"id":   ut.id,
		"exp":  ut.expires.Unix(),
	})
}

func (at *AgentToken) Token() *jwt.Token {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": at.Role(),
		"id":   at.id,
		"exp":  at.expires.Unix(),
	})
}
