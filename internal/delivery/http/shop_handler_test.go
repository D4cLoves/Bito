package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"bito/internal/delivery/ws"
	"bito/internal/economy"
	"bito/internal/model"
	"bito/internal/pkg/jwt"
	"bito/internal/service"

	"github.com/google/uuid"
)

type stubShop struct {
	err     error
	lastKey string
	look    model.Look
}

func (s *stubShop) State(context.Context, uuid.UUID) (service.ShopState, error) {
	return service.ShopState{Owned: []string{"theme:noir"}, Look: model.Look{Theme: "noir", Felt: "linen", Back: "classic"}}, nil
}

func (s *stubShop) Buy(_ context.Context, _ uuid.UUID, key string) (int64, error) {
	s.lastKey = key
	return 1500, s.err
}

func (s *stubShop) Wear(_ context.Context, _ uuid.UUID, look model.Look) error {
	s.look = look
	return s.err
}

func newShopRouter(t *testing.T, stub *stubShop) (http.Handler, string) {
	t.Helper()
	tokenManager := jwt.NewTokenManager("test-secret-key-super-secure-12345", 15*time.Minute, time.Hour)
	server := NewServer(":0", ws.NewHub(tokenManager, nil, nil), Handlers{Shop: NewShopHandler(stub)}, tokenManager)
	token, err := tokenManager.GenerateAccessToken(uuid.New(), "Alice")
	if err != nil {
		t.Fatal(err)
	}
	return server.httpServer.Handler, token
}

func TestShop_StateListsCatalogOwnedAndLook(t *testing.T) {
	router, token := newShopRouter(t, &stubShop{})
	w := doUsersRequest(router, http.MethodGet, "/api/v1/shop", token, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var resp ShopResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != len(economy.Catalog) || len(resp.Owned) != 1 || resp.Look.Theme != "noir" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestShop_BuyStatuses(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, http.StatusOK},
		{model.ErrUnknownItem, http.StatusNotFound},
		{model.ErrAlreadyOwned, http.StatusConflict},
		{&model.InsufficientFundsError{}, http.StatusPaymentRequired},
	}
	for _, tc := range cases {
		stub := &stubShop{err: tc.err}
		router, token := newShopRouter(t, stub)
		w := doUsersRequest(router, http.MethodPost, "/api/v1/shop/buy", token, `{"item":"theme:noir"}`)
		if w.Code != tc.want {
			t.Errorf("%v: status = %d, want %d", tc.err, w.Code, tc.want)
		}
		if stub.lastKey != "theme:noir" {
			t.Errorf("service got %q", stub.lastKey)
		}
	}
	router, token := newShopRouter(t, &stubShop{})
	if w := doUsersRequest(router, http.MethodPost, "/api/v1/shop/buy", token, `{"item":"theme:noir","price":0}`); w.Code != http.StatusBadRequest {
		t.Errorf("a client-sent price must be rejected, status = %d", w.Code)
	}
}

func TestShop_Wear(t *testing.T) {
	stub := &stubShop{}
	router, token := newShopRouter(t, stub)
	if w := doUsersRequest(router, http.MethodPut, "/api/v1/shop/look", token, `{"theme":"hall","felt":"wine","back":"bito"}`); w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	if stub.look != (model.Look{Theme: "hall", Felt: "wine", Back: "bito"}) {
		t.Fatalf("look = %+v", stub.look)
	}
	stub.err = model.ErrNotOwned
	if w := doUsersRequest(router, http.MethodPut, "/api/v1/shop/look", token, `{"theme":"noir","felt":"wine","back":"bito"}`); w.Code != http.StatusForbidden {
		t.Fatalf("not owned status = %d, want 403", w.Code)
	}
}
