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
