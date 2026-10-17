package ws

import "sync"

type Lobby struct {
	clients    map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	stopOnce   sync.Once
	online     chan chan int
}

var _ endpoint = (*Lobby)(nil)

func NewLobby() *Lobby {
	return &Lobby{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		online:     make(chan chan int),
	}
}

func (l *Lobby) Run() {
	for {
		select {
		case c := <-l.register:
			conn, ok := l.clients[c.ID()]
			if !ok {
				conn = make(map[*Client]bool)
				l.clients[c.ID()] = conn
			}
			conn[c] = true
		case c := <-l.unregister:
			if conns, ok := l.clients[c.ID()]; ok {
				delete(conns, c)
				if len(conns) == 0 {
					delete(l.clients, c.ID())
				}
			}
			c.Close()
		case reply := <-l.online:
			reply <- len(l.clients)
		case <-l.stop:
			for _, conns := range l.clients {
				for c := range conns {
					c.Close()
				}
			}
			return
		}
	}
}

func (l *Lobby) Register(c *Client) bool {
	select {
	case l.register <- c:
		return true
	case <-l.stop:
		return false
	}
}

func (l *Lobby) Online() int {
	reply := make(chan int, 1)
	select {
	case l.online <- reply:
	case <-l.stop:
		return 0
	}
	select {
	case n := <-reply:
		return n
	case <-l.stop:
		return 0
	}
}

func (l *Lobby) Close() {
	l.stopOnce.Do(func() { close(l.stop) })
}

func (l *Lobby) leave(c *Client) {
	select {
	case l.unregister <- c:
	case <-l.stop:
	}
}

func (l *Lobby) deliver(IncomingMessage) bool {
	select {
	case <-l.stop:
		return false
	default:
		return true
	}
}
