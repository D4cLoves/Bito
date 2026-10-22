//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"bito/internal/model"
)

func TestShop_BuyDebitsOnceAndJournals(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	wallet := NewWalletRepository(pool)
	ctx := context.Background()
	id := createUsers(t, pool, 1)[0]
	start := balanceOf(t, wallet, id)

	balance, err := shop.Buy(ctx, id, "theme:noir", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if balance != start-1000 || balanceOf(t, wallet, id) != start-1000 {
		t.Fatalf("balance = %d, want %d", balance, start-1000)
	}
	if _, err = shop.Buy(ctx, id, "theme:noir", 1000); !errors.Is(err, model.ErrAlreadyOwned) {
		t.Fatalf("second purchase err = %v, want ErrAlreadyOwned", err)
	}
	if got := balanceOf(t, wallet, id); got != start-1000 {
		t.Fatalf("a repeated purchase charged again: balance = %d", got)
	}

	owned, err := shop.Owned(ctx, id)
	if err != nil || len(owned) != 1 || owned[0] != "theme:noir" {
		t.Fatalf("owned = %v, %v", owned, err)
	}
	assertJournalMatchesBalances(t, pool)
}

func TestShop_NotEnoughChipsGrantsNothing(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	wallet := NewWalletRepository(pool)
	ctx := context.Background()
	id := createUsers(t, pool, 1)[0]
	start := balanceOf(t, wallet, id)

	if _, err := shop.Buy(ctx, id, "back:royal", start+1); !errors.Is(err, model.ErrInsufficientFunds) {
		t.Fatalf("err = %v, want ErrInsufficientFunds", err)
	}
	if owned, _ := shop.Owned(ctx, id); len(owned) != 0 {
		t.Fatalf("item granted without payment: %v", owned)
	}
	if got := balanceOf(t, wallet, id); got != start {
		t.Fatalf("balance changed to %d", got)
	}
	assertJournalMatchesBalances(t, pool)
}

func TestShop_TwoPurchasesRaceForTheLastChips(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	wallet := NewWalletRepository(pool)
	ctx := context.Background()
	id := createUsers(t, pool, 1)[0]
	start := balanceOf(t, wallet, id)
	price := start/2 + 1

	items := []string{"felt:royal", "back:royal"}
	errs := make([]error, len(items))
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = shop.Buy(ctx, id, item, price)
		}()
	}
	wg.Wait()

	ok, poor := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, model.ErrInsufficientFunds):
			poor++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || poor != 1 {
		t.Fatalf("succeeded %d, refused %d; want exactly one purchase", ok, poor)
	}
	if owned, _ := shop.Owned(ctx, id); len(owned) != 1 {
		t.Fatalf("owned = %v, want one item", owned)
	}
	if got := balanceOf(t, wallet, id); got != start-price {
		t.Fatalf("balance = %d, want %d", got, start-price)
	}
	assertJournalMatchesBalances(t, pool)
}

func TestShop_SameItemTwiceAtOnceChargesOnce(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	wallet := NewWalletRepository(pool)
	ctx := context.Background()
	id := createUsers(t, pool, 1)[0]
	start := balanceOf(t, wallet, id)

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = shop.Buy(ctx, id, "theme:ocean", 300)
		}()
	}
	wg.Wait()

	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatalf("errors = %v, %v; want one success and one ErrAlreadyOwned", errs[0], errs[1])
	}
	if got := balanceOf(t, wallet, id); got != start-300 {
		t.Fatalf("balance = %d, want one charge of 300", got)
	}
	assertJournalMatchesBalances(t, pool)
}

func TestShop_AvatarsAreSavedAndLookedUp(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)

	if _, found, err := shop.Avatar(ctx, ids[0]); err != nil || found {
		t.Fatalf("new user avatar found = %v, err = %v", found, err)
	}
	for _, avatar := range []string{"cat", "crown"} {
		if err := shop.SaveAvatar(ctx, ids[0], avatar); err != nil {
			t.Fatal(err)
		}
	}
	got, err := shop.Avatars(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[ids[0]] != "crown" {
		t.Fatalf("avatars = %v, want only the replaced choice of the first user", got)
	}
}

func TestShop_LookIsSavedAndReplaced(t *testing.T) {
	pool := setupTestDB(t)
	shop := NewShopRepository(pool)
	ctx := context.Background()
	id := createUsers(t, pool, 1)[0]

	if _, found, err := shop.Look(ctx, id); err != nil || found {
		t.Fatalf("new user look found = %v, err = %v", found, err)
	}
	for _, look := range []model.Look{{Theme: "hall", Felt: "green", Back: "bito"}, {Theme: "paper", Felt: "linen", Back: "classic"}} {
		if err := shop.SaveLook(ctx, id, look); err != nil {
			t.Fatal(err)
		}
		got, found, err := shop.Look(ctx, id)
		if err != nil || !found || got != look {
			t.Fatalf("look = %+v, %v, %v; want %+v", got, found, err, look)
		}
	}
}
