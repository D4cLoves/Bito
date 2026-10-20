//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"

	"bito/internal/model"

	"github.com/google/uuid"
)

func TestFriends_RequestAcceptRemove(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewFriendRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)
	alice, bob := ids[0], ids[1]

	state, err := repo.Request(ctx, alice, bob)
	if err != nil || state != model.FriendPending {
		t.Fatalf("request = %q, %v; want pending", state, err)
	}
	if _, err = repo.Request(ctx, alice, bob); !errors.Is(err, model.ErrFriendRequestExists) {
		t.Fatalf("repeat request err = %v, want ErrFriendRequestExists", err)
	}
	if err = repo.Accept(ctx, alice, bob); !errors.Is(err, model.ErrFriendRequestNotFound) {
		t.Fatalf("sender accepted own request: err = %v", err)
	}

	aliceView, err := repo.List(ctx, alice)
	if err != nil || len(aliceView) != 1 || !aliceView[0].Outgoing || aliceView[0].UserID != bob {
		t.Fatalf("alice list = %+v, %v; want one outgoing to bob", aliceView, err)
	}
	bobView, err := repo.List(ctx, bob)
	if err != nil || len(bobView) != 1 || bobView[0].Outgoing || bobView[0].UserID != alice {
		t.Fatalf("bob list = %+v, %v; want one incoming from alice", bobView, err)
	}

	if err = repo.Accept(ctx, bob, alice); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err = repo.Request(ctx, bob, alice); !errors.Is(err, model.ErrAlreadyFriends) {
		t.Fatalf("request to a friend err = %v, want ErrAlreadyFriends", err)
	}
	bobView, _ = repo.List(ctx, bob)
	if len(bobView) != 1 || bobView[0].State != model.FriendAccepted {
		t.Fatalf("bob list after accept = %+v", bobView)
	}

	if err = repo.Remove(ctx, bob, alice); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err = repo.Remove(ctx, bob, alice); err != nil {
		t.Fatalf("second remove should be a no-op: %v", err)
	}
	if list, _ := repo.List(ctx, alice); len(list) != 0 {
		t.Fatalf("alice list after remove = %+v, want empty", list)
	}
}

func TestFriends_CounterRequestBecomesFriendship(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewFriendRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)

	if _, err := repo.Request(ctx, ids[0], ids[1]); err != nil {
		t.Fatalf("request: %v", err)
	}
	state, err := repo.Request(ctx, ids[1], ids[0])
	if err != nil || state != model.FriendAccepted {
		t.Fatalf("counter request = %q, %v; want accepted", state, err)
	}
}

func TestFriends_SimultaneousRequestsMakeOneRow(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewFriendRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = repo.Request(ctx, ids[i], ids[1-i])
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatalf("simultaneous request failed: %v", err)
		}
	}

	var rows int
	var status string
	if err := pool.QueryRow(ctx, `SELECT COUNT(*), MAX(status) FROM friendships`).Scan(&rows, &status); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || status != "accepted" {
		t.Fatalf("rows = %d, status = %q; want one accepted row", rows, status)
	}
}

func TestFriends_DeletedUserDisappears(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewFriendRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)

	if _, err := repo.Request(ctx, ids[0], ids[1]); err != nil {
		t.Fatal(err)
	}
	if err := NewUserRepository(pool).Delete(ctx, ids[1]); err != nil {
		t.Fatal(err)
	}
	if list, _ := repo.List(ctx, ids[0]); len(list) != 0 {
		t.Fatalf("deleted user still listed: %+v", list)
	}
}

func TestFriends_DatabaseRejectsBrokenRows(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)
	low, high := orderedPair(ids[0], ids[1])

	cases := map[string][]any{
		"reversed pair":      {high, low, high, "pending"},
		"self":               {low, low, low, "pending"},
		"stranger requested": {low, high, createUsers(t, pool, 1)[0], "pending"},
		"unknown status":     {low, high, low, "blocked"},
	}
	for name, args := range cases {
		_, err := pool.Exec(ctx, `INSERT INTO friendships (user_low, user_high, requested_by, status) VALUES ($1, $2, $3, $4)`, args...)
		if err == nil {
			t.Errorf("%s: insert succeeded, want a constraint violation", name)
		}
	}
}

func TestFriends_BetweenFromBothSides(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewFriendRepository(pool)
	ctx := context.Background()
	ids := createUsers(t, pool, 2)
	alice, bob := ids[0], ids[1]

	check := func(me, other uuid.UUID, want model.Relation) {
		t.Helper()
		got, err := repo.Between(ctx, me, other)
		if err != nil || got != want {
			t.Fatalf("between = %q, %v; want %q", got, err, want)
		}
	}

	check(alice, bob, model.RelationNone)
	if _, err := repo.Request(ctx, alice, bob); err != nil {
		t.Fatal(err)
	}
	check(alice, bob, model.RelationOutgoing)
	check(bob, alice, model.RelationIncoming)
	if err := repo.Accept(ctx, bob, alice); err != nil {
		t.Fatal(err)
	}
	check(alice, bob, model.RelationFriend)
	check(bob, alice, model.RelationFriend)
}
