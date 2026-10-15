package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/pkg/jwt"

	"github.com/google/uuid"
)

func newTestTablesRouter(t *testing.T) (http.Handler, string, uuid.UUID) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	hub := ws.NewHub(tokenManager, nil, nil)
	server := NewServer(":0", hub, Handlers{Table: NewTableHandler(hub, nil)}, tokenManager)

	userID := uuid.New()
	token, err := tokenManager.GenerateAccessToken(userID, "Alice")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return server.httpServer.Handler, token, userID
}

func doTablesRequest(router http.Handler, method, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/tables", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestTables_RequireAuth(t *testing.T) {
	router, _, _ := newTestTablesRouter(t)

	for _, method := range []string{http.MethodPost, http.MethodGet} {
		w := doTablesRequest(router, method, "", `{"mode":"podkidnoy","deck":36,"maxPlayers":2}`)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s without token: status = %d, want 401", method, w.Code)
		}
	}
}

func TestTables_CreateValidation(t *testing.T) {
	router, token, _ := newTestTablesRouter(t)

	cases := []struct {
		name string
		body string
	}{
		{"unknown mode", `{"mode":"chess","deck":36,"maxPlayers":2}`},
		{"unknown deck", `{"mode":"podkidnoy","deck":40,"maxPlayers":2}`},
		{"too few players", `{"mode":"podkidnoy","deck":36,"maxPlayers":1}`},
		{"too many players", `{"mode":"podkidnoy","deck":52,"maxPlayers":7}`},
		{"not enough cards", `{"mode":"podkidnoy","deck":24,"maxPlayers":6}`},
		{"unknown field", `{"mode":"podkidnoy","deck":36,"maxPlayers":2,"bet":100}`},
		{"broken json", `{"mode":`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doTablesRequest(router, http.MethodPost, token, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (%s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestTables_CreateAndList(t *testing.T) {
	router, token, userID := newTestTablesRouter(t)

	w := doTablesRequest(router, http.MethodPost, token, `{"mode":"perevodnoy","deck":36,"maxPlayers":3}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (%s)", w.Code, w.Body.String())
	}

	var created ws.RoomInfo
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created table: %v", err)
	}
	if created.ID == "" || created.Mode != "perevodnoy" || created.Deck != 36 || created.MaxPlayers != 3 {
		t.Errorf("unexpected created table: %+v", created)
	}
	if created.CreatedBy != userID.String() {
		t.Errorf("createdBy = %q, want %q", created.CreatedBy, userID.String())
	}

	w = doTablesRequest(router, http.MethodGet, token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", w.Code)
	}

	var tables []ws.RoomInfo
	if err := json.Unmarshal(w.Body.Bytes(), &tables); err != nil {
		t.Fatalf("failed to decode tables: %v", err)
	}
	if len(tables) != 1 || tables[0].ID != created.ID {
		t.Errorf("list = %+v, want only %s", tables, created.ID)
	}
}

func TestTables_OneOpenTablePerPlayer(t *testing.T) {
	router, token, _ := newTestTablesRouter(t)
	body := `{"mode":"podkidnoy","deck":36,"maxPlayers":3}`

	if w := doTablesRequest(router, http.MethodPost, token, body); w.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want 201 (%s)", w.Code, w.Body.String())
	}

	w := doTablesRequest(router, http.MethodPost, token, body)
	if w.Code != http.StatusConflict {
		t.Fatalf("second create status = %d, want 409 (%s)", w.Code, w.Body.String())
	}
	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Error != msgOtherTable {
		t.Errorf("error = %q, want %q", resp.Error, msgOtherTable)
	}

	var tables []ws.RoomInfo
	_ = json.Unmarshal(doTablesRequest(router, http.MethodGet, token, "").Body.Bytes(), &tables)
	if len(tables) != 1 || tables[0].Players != 1 {
		t.Errorf("list = %+v, want one table with the creator's reserved seat", tables)
	}
}
