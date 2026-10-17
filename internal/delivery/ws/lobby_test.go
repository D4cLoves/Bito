package ws

import "testing"

func startLobby(t *testing.T) (*Lobby, <-chan struct{}) {
	t.Helper()
	l := NewLobby()
	done := make(chan struct{})
	go func() {
		l.Run()
		close(done)
	}()
	t.Cleanup(func() {
		l.Close()
		<-done
	})
	return l, done
}

func lobbyClient(id string, l *Lobby) *Client {
	return &Client{
		id:    id,
		name:  id,
		owner: l,
		send:  make(chan []byte, 16),
	}
}

func isClosed(c *Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func TestLobby_OnlineCountsPlayersNotTabs(t *testing.T) {
	l, _ := startLobby(t)

	tab1 := lobbyClient("alice", l)
	tab2 := lobbyClient("alice", l)
	bob := lobbyClient("bob", l)
	for _, c := range []*Client{tab1, tab2, bob} {
		if !l.Register(c) {
			t.Fatalf("register %s failed", c.ID())
		}
	}
	if got := l.Online(); got != 2 {
		t.Fatalf("online after 3 tabs of 2 players = %d, want 2", got)
	}

	l.leave(tab1)
	if got := l.Online(); got != 2 {
		t.Fatalf("online after alice closed one tab = %d, want 2", got)
	}

	l.leave(tab2)
	if got := l.Online(); got != 1 {
		t.Fatalf("online after alice closed both tabs = %d, want 1", got)
	}
}

func TestLobby_LeaveClosesClient(t *testing.T) {
	l, _ := startLobby(t)
	c := lobbyClient("alice", l)
	l.Register(c)

	l.leave(c)
	l.Online()

	if !isClosed(c) {
		t.Fatal("client is still open after leaving the lobby")
	}
}

func TestLobby_LeaveTwiceIsSafe(t *testing.T) {
	l, _ := startLobby(t)
	c := lobbyClient("alice", l)
	l.Register(c)

	l.leave(c)
	l.leave(c)

	if got := l.Online(); got != 0 {
		t.Fatalf("online = %d, want 0", got)
	}
}

func TestLobby_CloseClosesEveryClient(t *testing.T) {
	l, done := startLobby(t)
	a := lobbyClient("alice", l)
	b := lobbyClient("bob", l)
	l.Register(a)
	l.Register(b)
	l.Online()

	l.Close()
	<-done

	if !isClosed(a) || !isClosed(b) {
		t.Fatalf("clients left open after lobby closed: alice=%v bob=%v", isClosed(a), isClosed(b))
	}
}

func TestLobby_CallsAfterCloseDoNotHang(t *testing.T) {
	l, done := startLobby(t)
	l.Close()
	<-done

	c := lobbyClient("alice", l)
	if l.Register(c) {
		t.Fatal("register succeeded on a closed lobby")
	}
	if got := l.Online(); got != 0 {
		t.Fatalf("online on a closed lobby = %d, want 0", got)
	}
	l.leave(c)
	if l.deliver(IncomingMessage{Client: c}) {
		t.Fatal("deliver reports a closed lobby as alive")
	}
}
