package ws

import (
	"sort"

	"github.com/gorilla/websocket"
)

const maxWatchers = 30

type WatchersPayload struct {
	Count int `json:"count"`
}

func (r *Room) Watchers() int {
	return int(r.watcherCount.Load())
}

func (r *Room) addWatcher(c *Client) {
	old, again := r.watchers[c.ID()]
	if !again && len(r.watchers) >= maxWatchers {
		r.sendError(c, errTooManyWatchers)
		c.Close()
		return
	}
	if again {
		r.sendError(old, errOtherTab)
		old.Close()
	}

	r.watchers[c.ID()] = c
	r.watcherCount.Store(int32(len(r.watchers)))

	r.sendGameStateTo(c)
	if r.game != nil && r.finished.Load() {
		if data, err := NewMessage(TypeGameOver, r.gameOverPayload()); err == nil {
			c.Send(data)
		}
	}
	r.sendAll(TypeWatchers, WatchersPayload{Count: len(r.watchers)})
}

func (r *Room) removeWatcher(c *Client) {
	if r.watchers[c.ID()] != c {
		return
	}
	delete(r.watchers, c.ID())
	c.Close()
	r.watcherCount.Store(int32(len(r.watchers)))
	r.sendAll(TypeWatchers, WatchersPayload{Count: len(r.watchers)})
}

func (r *Room) broadcast(data []byte) {
	for _, c := range r.clients {
		c.Send(data)
	}
	for _, c := range r.watchers {
		c.Send(data)
	}
}

func (h *Hub) watch(conn *websocket.Conn, room *Room, playerID, playerName, key string) {
	if !room.MayEnter(playerID, key) {
		rejectConn(conn, errPrivateTable)
		return
	}
	client := NewWatcher(playerID, playerName, conn, room)
	if !room.RegisterClient(client) {
		client.Close()
		_ = conn.Close()
		return
	}
	go client.WritePump()
	go client.ReadPump()
}

func (h *Hub) ListLiveRooms() []RoomInfo {
	h.mu.RLock()
	rooms := make([]RoomInfo, 0)
	for _, room := range h.rooms {
		if room.Started() && !room.finished.Load() && !room.private {
			rooms = append(rooms, room.Info())
		}
	}
	h.mu.RUnlock()

	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Watchers > rooms[j].Watchers || rooms[i].Watchers == rooms[j].Watchers && rooms[i].CreatedAt.After(rooms[j].CreatedAt)
	})
	return rooms
}
