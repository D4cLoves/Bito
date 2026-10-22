package service

import (
	"context"
	"errors"
	"testing"

	"bito/internal/economy"
	"bito/internal/model"

	"github.com/google/uuid"
)

type memShop struct {
	owned   []string
	look    *model.Look
	charged map[string]int64
	avatars map[uuid.UUID]string
}

func (m *memShop) Buy(_ context.Context, _ uuid.UUID, item string, price int64) (int64, error) {
	m.charged[item] = price
	m.owned = append(m.owned, item)
	return 1000, nil
}

func (m *memShop) Owned(context.Context, uuid.UUID) ([]string, error) { return m.owned, nil }

func (m *memShop) Look(context.Context, uuid.UUID) (model.Look, bool, error) {
	if m.look == nil {
		return model.Look{}, false, nil
	}
	return *m.look, true, nil
}

func (m *memShop) SaveLook(_ context.Context, _ uuid.UUID, look model.Look) error {
	m.look = &look
	return nil
}

func (m *memShop) Avatar(_ context.Context, id uuid.UUID) (string, bool, error) {
	avatar, ok := m.avatars[id]
	return avatar, ok, nil
}

func (m *memShop) SaveAvatar(_ context.Context, id uuid.UUID, avatar string) error {
	if m.avatars == nil {
		m.avatars = map[uuid.UUID]string{}
	}
	m.avatars[id] = avatar
	return nil
}

func (m *memShop) Avatars(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		if avatar, ok := m.avatars[id]; ok {
			out[id] = avatar
		}
	}
	return out, nil
}

func TestShopService_PriceComesFromTheServerCatalog(t *testing.T) {
	repo := &memShop{charged: map[string]int64{}}
	s := NewShopService(repo)

	if _, err := s.Buy(context.Background(), uuid.New(), "theme:noir"); err != nil {
		t.Fatal(err)
	}
	want, _ := economy.ItemByKey("theme:noir")
	if repo.charged["theme:noir"] != want.Price {
		t.Fatalf("charged %d, want catalog price %d", repo.charged["theme:noir"], want.Price)
	}
}

func TestShopService_BuyRefusals(t *testing.T) {
	s := NewShopService(&memShop{charged: map[string]int64{}})
	cases := map[string]error{
		"theme:gold-plated": model.ErrUnknownItem,
		"noir":              model.ErrUnknownItem,
		"theme:paper":       model.ErrAlreadyOwned,
	}
	for key, want := range cases {
		if _, err := s.Buy(context.Background(), uuid.New(), key); !errors.Is(err, want) {
			t.Errorf("%s: err = %v, want %v", key, err, want)
		}
	}
}

func TestShopService_WearOnlyWhatYouOwn(t *testing.T) {
	repo := &memShop{owned: []string{"felt:wine"}, charged: map[string]int64{}}
	s := NewShopService(repo)
	ctx := context.Background()

	if err := s.Wear(ctx, uuid.New(), model.Look{Theme: "hall", Felt: "wine", Back: "bito"}); err != nil {
		t.Fatalf("free and owned items: %v", err)
	}
	if err := s.Wear(ctx, uuid.New(), model.Look{Theme: "noir", Felt: "wine", Back: "bito"}); !errors.Is(err, model.ErrNotOwned) {
		t.Fatalf("not owned theme: err = %v", err)
	}
	if err := s.Wear(ctx, uuid.New(), model.Look{Theme: "hall", Felt: "classic", Back: "bito"}); !errors.Is(err, model.ErrUnknownItem) {
		t.Fatalf("a card back used as felt: err = %v", err)
	}
}

func TestShopService_DefaultLookForNewPlayers(t *testing.T) {
	s := NewShopService(&memShop{charged: map[string]int64{}})
	state, err := s.State(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if state.Look != (model.Look{Theme: "paper", Felt: "linen", Back: "classic"}) || state.Owned != nil && len(state.Owned) != 0 {
		t.Fatalf("state = %+v", state)
	}
}

func TestShopService_AvatarFreeOwnedOrRefused(t *testing.T) {
	repo := &memShop{owned: []string{"avatar:cat"}, charged: map[string]int64{}}
	s := NewShopService(repo)
	ctx := context.Background()
	me := uuid.New()

	cases := []struct {
		avatar string
		want   error
	}{
		{"spade", nil},
		{"cat", nil},
		{"crown", model.ErrNotOwned},
		{"noir", model.ErrUnknownItem},
		{"", model.ErrUnknownItem},
	}
	for _, tc := range cases {
		if err := s.SetAvatar(ctx, me, tc.avatar); !errors.Is(err, tc.want) {
			t.Errorf("%q: err = %v, want %v", tc.avatar, err, tc.want)
		}
	}
	if repo.avatars[me] != "cat" {
		t.Fatalf("saved avatar = %q, want the last allowed one", repo.avatars[me])
	}
}

func TestShopService_AvatarsDefaultForPlayersWithoutChoice(t *testing.T) {
	picked, fresh := uuid.New(), uuid.New()
	repo := &memShop{charged: map[string]int64{}, avatars: map[uuid.UUID]string{picked: "skull"}}
	s := NewShopService(repo)

	got, err := s.Avatars(context.Background(), []uuid.UUID{picked, fresh})
	if err != nil {
		t.Fatal(err)
	}
	if got[picked] != "skull" || got[fresh] != "initial" {
		t.Fatalf("avatars = %v", got)
	}

	state, err := s.State(context.Background(), fresh)
	if err != nil || state.Avatar != "initial" {
		t.Fatalf("new player avatar = %q, %v", state.Avatar, err)
	}
}

func TestShopService_AvatarLookupIsCapped(t *testing.T) {
	ids := make([]uuid.UUID, maxAvatarLookup+20)
	for i := range ids {
		ids[i] = uuid.New()
	}
	got, err := NewShopService(&memShop{charged: map[string]int64{}}).Avatars(context.Background(), ids)
	if err != nil || len(got) != maxAvatarLookup {
		t.Fatalf("got %d avatars, err %v; want %d", len(got), err, maxAvatarLookup)
	}
}
