package wsproto

import (
	"sync"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type read struct {
	messageType int
	message     []byte
	err         error
}

type Client struct {
	id      uuid.UUID
	conn    *websocket.Conn
	user    *auth.UserToken
	alive   bool
	readCh  chan read
	seq     int64
	writeMu *sync.Mutex
}

func (c *Client) ID() uuid.UUID {
	return c.id
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) UserID() domain.UserID {
	return c.user.ID()
}

func (c *Client) Connected() bool {
	return c.alive
}

func (c *Client) Send(message *Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	message.Seq = c.seq
	c.seq++
	err := c.conn.WriteJSON(message)
	return err
}

func (c *Client) readLoop() {
	for {
		messageType, message, err := c.conn.ReadMessage()
		c.readCh <- read{
			messageType: messageType,
			message:     message,
			err:         err,
		}
		if err != nil {
			return
		}
	}
}
