package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/model"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type stubWallet struct {
	balance  int64
	topUpErr error
	list     []model.WalletTransaction
	limit    int
}

func (s *stubWallet) Balance(context.Context, string) (int64, error) {
	return s.balance, nil
}

func (s *stubWallet) TopUp(context.Context, uuid.UUID) (int64, error) {
	if s.topUpErr != nil {
		return 0, s.topUpErr
	}
	return 500, nil
}

func (s *stubWallet) Transactions(_ context.Context, _ uuid.UUID, limit int) ([]model.WalletTransaction, error) {
	s.limit = limit
	return s.list, nil
}

func newEconomyRouter(t *testing.T, wallet *stubWallet) (http.Handler, *ws.Hub, string, uuid.UUID) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	hub := ws.NewHub(tokenManager, nil, nil)
	server := NewServer(":0", hub, Handlers{
		Table:  NewTableHandler(hub, wallet),
		Wallet: NewWalletHandler(wallet),
	}, tokenManager)

	userID := uuid.New()
	token, err := tokenManager.GenerateAccessToken(userID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return server.httpServer.Handler, hub, token, userID
}

func doRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func dialRoom(t *testing.T, router http.Handler, roomID, token string) *websocket.Conn {
	t.Helper()
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room_id=" + roomID + "&token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial room: %v", err)
	}
	return conn
}

func TestTables_StakeDefaultsToMinimum(t *testing.T) {
	router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})

	w := doRequest(router, http.MethodPost, "/api/v1/tables", token, `{"mode":"podkidnoy","deck":36,"maxPlayers":3}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	var info ws.RoomInfo
	_ = json.Unmarshal(w.Body.Bytes(), &info)
	if info.Stake != 100 {
		t.Errorf("stake = %d, want default 100", info.Stake)
	}
}

func TestTables_StakeRules(t *testing.T) {
	cases := []struct {
		name    string
		balance int64
		body    string
		want    int
	}{
		{"allowed stake", 2500, `{"mode":"podkidnoy","deck":36,"maxPlayers":3,"stake":500}`, http.StatusCreated},
		{"stake not in the list", 2500, `{"mode":"podkidnoy","deck":36,"maxPlayers":3,"stake":300}`, http.StatusBadRequest},
		{"negative stake", 2500, `{"mode":"podkidnoy","deck":36,"maxPlayers":3,"stake":-100}`, http.StatusBadRequest},
		{"balance below stake", 400, `{"mode":"podkidnoy","deck":36,"maxPlayers":3,"stake":500}`, http.StatusPaymentRequired},
		{"balance equals stake", 500, `{"mode":"podkidnoy","deck":36,"maxPlayers":3,"stake":500}`, http.StatusCreated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: tc.balance})
			w := doRequest(router, http.MethodPost, "/api/v1/tables", token, tc.body)
			if w.Code != tc.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestTables_RefusesPlayerSeatedElsewhere(t *testing.T) {
	router, hub, token, userID := newEconomyRouter(t, &stubWallet{balance: 2500})

	room := hub.CreateRoom(ws.RoomSettings{Mode: 1, Deck: 36, MaxPlayers: 2, Stake: 100})
	defer room.Close()
	conn := dialRoom(t, router, room.ID(), token)
	defer conn.Close()
	time.Sleep(30 * time.Millisecond)

	if !hub.SeatedElsewhere(userID.String(), "") {
		t.Fatal("player must be seated after joining over websocket")
	}

	w := doRequest(router, http.MethodPost, "/api/v1/tables", token, `{"mode":"podkidnoy","deck":36,"maxPlayers":3}`)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 (%s)", w.Code, w.Body.String())
	}
}

func TestTables_QuickMatch(t *testing.T) {
	router, hub, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})

	first := doRequest(router, http.MethodPost, "/api/v1/tables/quick", token, `{"mode":"any","deck":0,"stake":250}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first quick status = %d, want 201 (%s)", first.Code, first.Body.String())
	}
	var created ws.RoomInfo
	_ = json.Unmarshal(first.Body.Bytes(), &created)
	if created.MaxPlayers != 2 || created.Mode != "podkidnoy" || created.Deck != 36 || created.Stake != 250 {
		t.Errorf("created duel = %+v, want 2 players, podkidnoy, 36, stake 250", created)
	}
	if list := hub.ListOpenRooms(); len(list) != 0 {
		t.Errorf("duel tables must stay out of the public list, got %+v", list)
	}

	for _, body := range []string{
		`{"mode":"podkidnoy","deck":36,"stake":250}`,
		`{"mode":"any","deck":36,"stake":250}`,
		`{"stake":250}`,
	} {
		w := doRequest(router, http.MethodPost, "/api/v1/tables/quick", token, body)
		var got ws.RoomInfo
		_ = json.Unmarshal(w.Body.Bytes(), &got)
		if w.Code != http.StatusOK || got.ID != created.ID {
			t.Errorf("%s: status %d table %s, want the waiting duel %s", body, w.Code, got.ID, created.ID)
		}
	}

	other := doRequest(router, http.MethodPost, "/api/v1/tables/quick", token, `{"mode":"perevodnoy","stake":250}`)
	var otherInfo ws.RoomInfo
	_ = json.Unmarshal(other.Body.Bytes(), &otherInfo)
	if other.Code != http.StatusCreated || otherInfo.ID == created.ID || otherInfo.Mode != "perevodnoy" {
		t.Errorf("another mode must open a new duel, got %d %+v", other.Code, otherInfo)
	}
}

func TestTables_QuickValidation(t *testing.T) {
	router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})
	for _, body := range []string{
		`{"mode":"chess","stake":100}`,
		`{"mode":"any","deck":40,"stake":100}`,
		`{"mode":"any","stake":300}`,
		`{"mode":"any","maxPlayers":4}`,
	} {
		if w := doRequest(router, http.MethodPost, "/api/v1/tables/quick", token, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, w.Code)
		}
	}
}

func TestTables_CreateIsForCompaniesOnly(t *testing.T) {
	router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})
	if w := doRequest(router, http.MethodPost, "/api/v1/tables", token, `{"mode":"podkidnoy","deck":36,"maxPlayers":2}`); w.Code != http.StatusBadRequest {
		t.Errorf("two-player table via create: status = %d, want 400", w.Code)
	}
}

func TestWallet_TopUp(t *testing.T) {
	next := time.Date(2026, 10, 14, 13, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		err      error
		want     int
		wantNext bool
	}{
		{"granted", nil, http.StatusOK, false},
		{"balance too high", model.ErrTopUpNotAllowed, http.StatusConflict, false},
		{"cooldown", &model.TopUpCooldownError{NextAt: next}, http.StatusConflict, true},
		{"database down", errors.New("boom"), http.StatusInternalServerError, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router, _, token, _ := newEconomyRouter(t, &stubWallet{topUpErr: tc.err})
			w := doRequest(router, http.MethodPost, "/api/v1/wallet/top-up", token, "")
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.want, w.Body.String())
			}
			var refusal TopUpRefusal
			_ = json.Unmarshal(w.Body.Bytes(), &refusal)
			if tc.wantNext && (refusal.NextAt == nil || !refusal.NextAt.Equal(next)) {
				t.Errorf("nextAt = %v, want %v", refusal.NextAt, next)
			}
		})
	}
}

func TestWallet_TransactionsLimit(t *testing.T) {
	wallet := &stubWallet{list: []model.WalletTransaction{{ID: 1, Amount: 2500, Kind: model.TxSignupBonus, BalanceAfter: 2500}}}
	router, _, token, _ := newEconomyRouter(t, wallet)

	w := doRequest(router, http.MethodGet, "/api/v1/wallet/transactions", token, "")
	if w.Code != http.StatusOK || wallet.limit != defaultTransactionsLimit {
		t.Fatalf("status = %d limit = %d, want 200 and default limit", w.Code, wallet.limit)
	}
	var list []model.WalletTransaction
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list) != 1 || list[0].Kind != model.TxSignupBonus {
		t.Errorf("body = %s, want one signup bonus", w.Body.String())
	}

	if w = doRequest(router, http.MethodGet, "/api/v1/wallet/transactions?limit=abc", token, ""); w.Code != http.StatusBadRequest {
		t.Errorf("bad limit status = %d, want 400", w.Code)
	}
}

func TestUsers_ActiveTable(t *testing.T) {
	router, hub, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})
	bobToken, err := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour).GenerateAccessToken(uuid.New(), "Bob")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	if w := doRequest(router, http.MethodGet, "/api/v1/users/me/active-table", token, ""); w.Code != http.StatusNoContent {
		t.Fatalf("no table: status = %d, want 204", w.Code)
	}

	room := hub.CreateRoom(ws.RoomSettings{Mode: 2, Deck: 36, MaxPlayers: 2, Stake: 500})
	defer room.Close()
	alice := dialRoom(t, router, room.ID(), token)
	defer alice.Close()
	time.Sleep(30 * time.Millisecond)

	if w := doRequest(router, http.MethodGet, "/api/v1/users/me/active-table", token, ""); w.Code != http.StatusNoContent {
		t.Fatalf("waiting table: status = %d, want 204 until the game starts", w.Code)
	}

	bob := dialRoom(t, router, room.ID(), bobToken)
	defer bob.Close()
	time.Sleep(50 * time.Millisecond)

	w := doRequest(router, http.MethodGet, "/api/v1/users/me/active-table", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("running game: status = %d, want 200", w.Code)
	}
	var info ws.RoomInfo
	_ = json.Unmarshal(w.Body.Bytes(), &info)
	if info.ID != room.ID() || info.Mode != "perevodnoy" || info.Stake != 500 || info.MaxPlayers != 2 {
		t.Errorf("active table = %+v, want the running room", info)
	}

	if err := bob.WriteJSON(map[string]string{"type": "surrender"}); err != nil {
		t.Fatalf("surrender: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if w := doRequest(router, http.MethodGet, "/api/v1/users/me/active-table", token, ""); w.Code != http.StatusNoContent {
		t.Errorf("finished game: status = %d, want 204", w.Code)
	}
}

type stubGames struct {
	limit int
	list  []model.GameHistoryEntry
}

func (s *stubGames) History(_ context.Context, _ uuid.UUID, limit int) ([]model.GameHistoryEntry, error) {
	s.limit = limit
	return s.list, nil
}

func TestUsers_GameHistory(t *testing.T) {
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	games := &stubGames{list: []model.GameHistoryEntry{{ID: uuid.New(), Mode: "podkidnoy", Deck: 24, Players: 2, Stake: 500, Place: 1, Payout: 475}}}
	server := NewServer(":0", ws.NewHub(tokenManager, nil, nil), Handlers{Game: NewGameHandler(games)}, tokenManager)
	token, _ := tokenManager.GenerateAccessToken(uuid.New(), "Alice")

	w := doRequest(server.httpServer.Handler, http.MethodGet, "/api/v1/users/me/games?limit=1", token, "")
	if w.Code != http.StatusOK || games.limit != 1 {
		t.Fatalf("status = %d limit = %d, want 200 and limit 1", w.Code, games.limit)
	}
	var list []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["payout"] != float64(475) || list[0]["players"] != float64(2) || list[0]["finishedAt"] == nil {
		t.Errorf("body = %s, want one entry in the frontend format", w.Body.String())
	}

	if w = doRequest(server.httpServer.Handler, http.MethodGet, "/api/v1/users/me/games?limit=0", token, ""); w.Code != http.StatusBadRequest {
		t.Errorf("limit=0 status = %d, want 400", w.Code)
	}
}

func TestTables_GetByID(t *testing.T) {
	router, _, token, _ := newEconomyRouter(t, &stubWallet{balance: 2500})

	w := doRequest(router, http.MethodPost, "/api/v1/tables/quick", token, `{"mode":"perevodnoy","deck":52,"stake":1000}`)
	var duel ws.RoomInfo
	_ = json.Unmarshal(w.Body.Bytes(), &duel)

	w = doRequest(router, http.MethodGet, "/api/v1/tables/"+duel.ID, token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", w.Code, w.Body.String())
	}
	var got ws.RoomInfo
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.ID != duel.ID || !got.Matchmaking || got.Stake != 1000 || got.Mode != "perevodnoy" || got.Deck != 52 {
		t.Errorf("table = %+v, want the hidden duel with its settings", got)
	}

	if w = doRequest(router, http.MethodGet, "/api/v1/tables/"+uuid.NewString(), token, ""); w.Code != http.StatusNotFound {
		t.Errorf("unknown table: status = %d, want 404", w.Code)
	}
}
