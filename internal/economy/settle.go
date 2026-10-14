package economy

type Result struct {
	Places []string
	Losers map[string]bool
}

type Payout struct {
	UserID string
	Place  int
	Credit int64
	Net    int64
	Win    bool
}

type Settlement struct {
	Payouts []Payout
	Fee     int64
	Draw    bool
}

func Settle(stake int64, r Result) Settlement {
	winners := make([]string, 0, len(r.Places))
	losers := make([]string, 0, len(r.Places))
	for _, id := range r.Places {
		if r.Losers[id] {
			losers = append(losers, id)
		} else {
			winners = append(winners, id)
		}
	}

	if len(winners) == 0 || len(losers) == 0 {
		return refundAll(stake, r.Places)
	}

	m := int64(len(winners))
	pool := stake * int64(len(losers))
	fee := pool * FeePercent / 100
	shares := splitByPlace(pool-fee, m)

	payouts := make([]Payout, 0, len(r.Places))
	for i, id := range winners {
		payouts = append(payouts, Payout{
			UserID: id,
			Place:  i + 1,
			Credit: stake + shares[i],
			Net:    shares[i],
			Win:    true,
		})
	}
	for i, id := range losers {
		payouts = append(payouts, Payout{
			UserID: id,
			Place:  len(winners) + i + 1,
			Net:    -stake,
		})
	}

	return Settlement{Payouts: payouts, Fee: fee}
}

func splitByPlace(dist, m int64) []int64 {
	total := m * (m + 1) / 2
	shares := make([]int64, m)
	var given int64
	for i := int64(0); i < m; i++ {
		shares[i] = dist * (m - i) / total
		given += shares[i]
	}
	shares[0] += dist - given
	return shares
}

func refundAll(stake int64, places []string) Settlement {
	payouts := make([]Payout, 0, len(places))
	for i, id := range places {
		payouts = append(payouts, Payout{
			UserID: id,
			Place:  i + 1,
			Credit: stake,
		})
	}
	return Settlement{Payouts: payouts, Draw: true}
}
