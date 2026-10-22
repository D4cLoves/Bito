package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 2048
	sendBufferSize = 256
)

type endpoint interface {
	deliver(IncomingMessage) bool
	leave(*Client)
}

type Client struct {
	id     string
	name   string
	conn   *websocket.Conn
	owner  endpoint
	watch  bool
	send   chan []byte
	mu     sync.Mutex
	closed bool
}

func NewClient(id, name string, conn *websocket.Conn, ep endpoint) *Client {
	return &Client{
		id:    id,
		name:  name,
		conn:  conn,
		owner: ep,
		send:  make(chan []byte, sendBufferSize),
	}
}

func NewWatcher(id, name string, conn *websocket.Conn, ep endpoint) *Client {
	c := NewClient(id, name, conn, ep)
	c.watch = true
	return c
}

func (c *Client) Watching() bool {
	return c.watch
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Send(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}

	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.owner.leave(c)
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	if c.conn == nil {
		return
	}

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if !c.owner.deliver(IncomingMessage{Client: c, Data: message}) {
			return
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if c.conn != nil {
			c.conn.Close()
		}
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if c.conn == nil {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			if c.conn == nil {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
