package wsproto

import (
	"sync"

	"github.com/dezemandje/aule/internal/backend/auth"
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
	writeMu *sync.Mutex
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Send(message *Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
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
