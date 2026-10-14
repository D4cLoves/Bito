package economy

import (
	"fmt"
	"testing"
)

func losersOf(ids ...string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func netByUser(s Settlement) map[string]int64 {
	out := make(map[string]int64, len(s.Payouts))
	for _, p := range s.Payouts {
		out[p.UserID] = p.Net
	}
	return out
}

func TestSettle_Examples(t *testing.T) {
	cases := []struct {
		name    string
		stake   int64
		places  []string
		losers  []string
		wantNet map[string]int64
		wantFee int64
	}{
		{
			name:    "features.md 3.3: four players, stake 500",
			stake:   500,
			places:  []string{"a", "b", "c", "d"},
			losers:  []string{"d"},
			wantNet: map[string]int64{"a": 238, "b": 158, "c": 79, "d": -500},
			wantFee: 25,
		},
		{
			name:    "two players, stake 100",
			stake:   100,
			places:  []string{"a", "b"},
			losers:  []string{"b"},
			wantNet: map[string]int64{"a": 95, "b": -100},
			wantFee: 5,
		},
		{
			name:    "fool and a surrendered player both pay",
			stake:   500,
			places:  []string{"a", "b", "c", "d"},
			losers:  []string{"c", "d"},
			wantNet: map[string]int64{"a": 634, "b": 316, "c": -500, "d": -500},
			wantFee: 50,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Settle(tc.stake, Result{Places: tc.places, Losers: losersOf(tc.losers...)})
			if s.Draw {
				t.Fatal("unexpected draw")
			}
			if s.Fee != tc.wantFee {
				t.Errorf("fee = %d, want %d", s.Fee, tc.wantFee)
			}
			got := netByUser(s)
			for id, want := range tc.wantNet {
				if got[id] != want {
					t.Errorf("net[%s] = %d, want %d", id, got[id], want)
				}
			}
		})
	}
}

func TestSettle_PlacesAndCredit(t *testing.T) {
	s := Settle(500, Result{Places: []string{"a", "b", "c", "d"}, Losers: losersOf("b")})

	want := []struct {
		id     string
		place  int
		credit int64
		win    bool
	}{
		{"a", 1, 500 + 238, true},
		{"c", 2, 500 + 158, true},
		{"d", 3, 500 + 79, true},
		{"b", 4, 0, false},
	}
	if len(s.Payouts) != len(want) {
		t.Fatalf("payouts = %d, want %d", len(s.Payouts), len(want))
	}
	for i, w := range want {
		p := s.Payouts[i]
		if p.UserID != w.id || p.Place != w.place || p.Credit != w.credit || p.Win != w.win {
			t.Errorf("payout[%d] = %+v, want id=%s place=%d credit=%d win=%v", i, p, w.id, w.place, w.credit, w.win)
		}
	}
}

func TestSettle_DrawRefundsEveryone(t *testing.T) {
	cases := []struct {
		name   string
		losers []string
	}{
		{"nobody lost", nil},
		{"everybody lost", []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Settle(250, Result{Places: []string{"a", "b", "c"}, Losers: losersOf(tc.losers...)})
			if !s.Draw || s.Fee != 0 {
				t.Fatalf("draw = %v, fee = %d, want draw with zero fee", s.Draw, s.Fee)
			}
			for _, p := range s.Payouts {
				if p.Credit != 250 || p.Net != 0 || p.Win {
					t.Errorf("payout %+v, want full refund", p)
				}
			}
		})
	}
}

func TestSettle_IgnoresLosersOutsidePlaces(t *testing.T) {
	s := Settle(100, Result{Places: []string{"a", "b"}, Losers: losersOf("b", "ghost")})
	if s.Fee != 5 {
		t.Errorf("fee = %d, want 5: a loser who is not at the table must not grow the pool", s.Fee)
	}
}

func TestSettle_MoneyInvariant(t *testing.T) {
	for _, stake := range AllowedStakes {
		for players := 2; players <= 6; players++ {
			for losers := 1; losers < players; losers++ {
				name := fmt.Sprintf("stake=%d players=%d losers=%d", stake, players, losers)
				t.Run(name, func(t *testing.T) {
					places := make([]string, players)
					lost := make(map[string]bool, losers)
					for i := range places {
						places[i] = fmt.Sprintf("p%d", i)
						if i >= players-losers {
							lost[places[i]] = true
						}
					}

					s := Settle(stake, Result{Places: places, Losers: lost})

					var sumNet, sumCredit int64
					for _, p := range s.Payouts {
						sumNet += p.Net
						sumCredit += p.Credit
						if p.Win && p.Net <= 0 {
							t.Errorf("winner %s got %d, a winner must never lose money", p.UserID, p.Net)
						}
						if p.Credit != stake+p.Net {
							t.Errorf("%s: credit %d != stake %d + net %d", p.UserID, p.Credit, stake, p.Net)
						}
					}
					if s.Fee < 0 {
						t.Errorf("fee = %d, must not be negative", s.Fee)
					}
					if sumNet != -s.Fee {
						t.Errorf("sum of net = %d, want -fee = %d", sumNet, -s.Fee)
					}
					if locked := stake * int64(players); sumCredit+s.Fee != locked {
						t.Errorf("credited %d + fee %d != locked %d", sumCredit, s.Fee, locked)
					}
				})
			}
		}
	}
}
