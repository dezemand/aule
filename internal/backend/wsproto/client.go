package wsproto

import (
	"sync"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/domain"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

// readResult holds the result of a WebSocket read operation.
type readResult struct {
	messageType int
	message     []byte
	err         error
}

// Client represents a connected WebSocket client.
type Client struct {
	id      uuid.UUID
	conn    *websocket.Conn
	user    *auth.UserToken
	alive   bool
	readCh  chan readResult
	seq     int64
	writeMu *sync.Mutex
}

// ID returns the client's unique identifier.
func (c *Client) ID() uuid.UUID {
	return c.id
}

// Close closes the WebSocket connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// UserID returns the authenticated user's ID.
func (c *Client) UserID() domain.UserID {
	return c.user.ID()
}

// Connected returns whether the client is still connected.
func (c *Client) Connected() bool {
	return c.alive
}

// Send sends an envelope to the client.
// Thread-safe; can be called from multiple goroutines.
func (c *Client) Send(message *modelsws.Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	message.Seq = c.seq
	c.seq++
	return c.conn.WriteJSON(message)
}

// SendPayload creates an envelope and sends it to the client.
func (c *Client) SendPayload(typ string, payload any) error {
	env, err := modelsws.NewEnvelope(typ, payload)
	if err != nil {
		return err
	}
	return c.Send(env)
}

// SendError sends an error envelope to the client.
func (c *Client) SendError(code, message string, detail any) error {
	env, err := modelsws.NewErrorEnvelope(code, message, detail)
	if err != nil {
		return err
	}
	return c.Send(env)
}

// readLoop continuously reads messages from the WebSocket connection.
func (c *Client) readLoop() {
	for {
		messageType, message, err := c.conn.ReadMessage()
		c.readCh <- readResult{
			messageType: messageType,
			message:     message,
			err:         err,
		}
		if err != nil {
			return
		}
	}
}
