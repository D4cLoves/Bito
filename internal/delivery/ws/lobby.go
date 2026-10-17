package ws

import (
	"bytes"
	"log"
	"sync"
	"time"
)

const lobbyRefresh = 2 * time.Second

type tableSource interface {
	ListOpenRooms() []RoomInfo
}

type Lobby struct {
	clients    map[string]map[*Client]bool
	tables     tableSource
	interval   time.Duration
	last       []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	stopOnce   sync.Once
	online     chan chan int
}

var _ endpoint = (*Lobby)(nil)

func NewLobby(tables tableSource, interval time.Duration) *Lobby {
	return &Lobby{
		clients:    make(map[string]map[*Client]bool),
		tables:     tables,
		interval:   interval,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		online:     make(chan chan int),
	}
}

func (l *Lobby) Run() {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	for {
		select {
		case c := <-l.register:
			l.add(c)
			if !l.publish() {
				l.sendTo(c, l.last)
			}
		case c := <-l.unregister:
			if l.remove(c) {
				l.publish()
			}
			c.Close()
		case <-ticker.C:
			l.publish()
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

func (l *Lobby) add(c *Client) {
	conns, ok := l.clients[c.ID()]
	if !ok {
		conns = make(map[*Client]bool)
		l.clients[c.ID()] = conns
	}
	conns[c] = true
}

func (l *Lobby) remove(c *Client) (wentOffline bool) {
	conns, ok := l.clients[c.ID()]
	if !ok || !conns[c] {
		return false
	}
	delete(conns, c)
	if len(conns) > 0 {
		return false
	}
	delete(l.clients, c.ID())
	return true
}

func (l *Lobby) publish() bool {
	data := l.snapshot()
	if data == nil || bytes.Equal(data, l.last) {
		return false
	}
	l.last = data
	for _, conns := range l.clients {
		for c := range conns {
			l.sendTo(c, data)
		}
	}
	return true
}

func (l *Lobby) snapshot() []byte {
	tables := l.tables.ListOpenRooms()
	if tables == nil {
		tables = []RoomInfo{}
	}
	data, err := NewMessage(TypeLobbyState, LobbyStatePayload{Online: len(l.clients), Tables: tables})
	if err != nil {
		log.Printf("[Lobby] не удалось собрать состояние: %v", err)
		return nil
	}
	return data
}

func (l *Lobby) sendTo(c *Client, data []byte) {
	if data == nil || c.Send(data) {
		return
	}
	log.Printf("[Lobby] клиент %s не успевает читать, отключаем", c.ID())
	l.remove(c)
	c.Close()
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
