package ws

import (
	"log"
	"sort"

	"bito/internal/game"
)

type rematchStarter func(s RoomSettings, players []string) (string, error)

type RematchStatePayload struct {
	Votes  []string `json:"votes"`
	Needed int      `json:"needed"`
}

type RematchCancelledPayload struct {
	Reason string `json:"reason"`
}

type RematchStartPayload struct {
	TableID string `json:"tableId"`
}

func (r *Room) rematchVoters() []string {
	ids := make([]string, 0, len(r.clients))
	for id := range r.clients {
		if p, _ := r.game.PlayerByID(game.PlayerID(id)); p != nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (r *Room) handleRematch(c *Client, agree bool) {
	switch {
	case r.rematch == nil || r.rematchOver:
		r.sendError(c, errRematchClosed)
		return
	case r.game == nil || r.game.Phase() != game.PhaseFinished:
		r.sendError(c, errRematchTooEarly)
		return
	}

	if !agree {
		r.cancelRematch(c.Name() + " не хочет реванш")
		return
	}
	if r.rematchVotes == nil {
		r.rematchVotes = make(map[string]bool)
	}
	r.rematchVotes[c.ID()] = true
	r.checkRematch()
}

func (r *Room) checkRematch() {
	if r.rematchOver || r.rematchVotes == nil {
		return
	}

	voters := r.rematchVoters()
	if len(voters) < 2 {
		r.cancelRematch("соперник ушёл из-за стола")
		return
	}

	agreed := make([]string, 0, len(voters))
	for _, id := range voters {
		if r.rematchVotes[id] {
			agreed = append(agreed, id)
		}
	}
	r.sendAll(TypeRematchState, RematchStatePayload{Votes: agreed, Needed: len(voters)})
	if len(agreed) < len(voters) {
		return
	}

	if reason := r.rematchFundsProblem(voters); reason != "" {
		r.cancelRematch(reason)
		return
	}

	settings := RoomSettings{
		Mode:       r.gameMode,
		Deck:       r.deckType,
		MaxPlayers: len(voters),
		Stake:      r.stake,
		CreatedBy:  voters[0],
	}
	tableID, err := r.rematch(settings, voters)
	if err != nil {
		log.Printf("[Room %s] реванш не создан: %v", r.id, err)
		r.cancelRematch(errRematchFailed)
		return
	}

	r.rematchOver = true
	r.sendAll(TypeRematchStart, RematchStartPayload{TableID: tableID})
}

func (r *Room) rematchFundsProblem(voters []string) string {
	if r.stake <= 0 {
		return ""
	}
	ctx, cancel := bankContext()
	defer cancel()
	for _, id := range voters {
		balance, err := r.bank.Balance(ctx, id)
		if err != nil {
			log.Printf("[Room %s] баланс %s для реванша: %v", r.id, id, err)
			return errBankUnavailable
		}
		if balance < r.stake {
			return r.clients[id].Name() + ": не хватает фишек на ставку"
		}
	}
	return ""
}

func (r *Room) cancelRematch(reason string) {
	r.rematchOver = true
	r.sendAll(TypeRematchCancelled, RematchCancelledPayload{Reason: reason})
}

func (r *Room) sendAll(msgType string, payload any) {
	data, err := NewMessage(msgType, payload)
	if err != nil {
		return
	}
	for _, c := range r.clients {
		c.Send(data)
	}
}
