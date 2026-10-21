package ws

import (
	"context"
	"math"
	"time"

	"bito/internal/economy"
)

const bankTimeout = 5 * time.Second

const (
	errNoFunds         = "не хватает фишек на ставку"
	errOtherTable      = "вы уже играете за другим столом"
	errStartFailed     = "не удалось начать партию, попробуйте ещё раз"
	errBankUnavailable = "касса недоступна, попробуйте позже"
	errStakesRefunded  = "не удалось провести расчёт, ставки возвращены"
	errOtherTab        = "стол открыт в другой вкладке"
	errPrivateTable    = "стол закрытый: нужно приглашение"
	errRematchClosed   = "реванш уже недоступен"
	errRematchTooEarly = "партия ещё не закончилась"
	errRematchFailed   = "не удалось начать реванш"
)

type Bank interface {
	Balance(ctx context.Context, userID string) (int64, error)
	LockStakes(ctx context.Context, gameID, mode string, deck int, stake int64, players []string) error
	Settle(ctx context.Context, gameID string, stake int64, r economy.Result) (economy.Settlement, error)
	RefundGame(ctx context.Context, gameID string) (bool, error)
}

type freeBank struct{}

func (freeBank) Balance(context.Context, string) (int64, error) {
	return math.MaxInt64, nil
}

func (freeBank) LockStakes(context.Context, string, string, int, int64, []string) error {
	return nil
}

func (freeBank) Settle(_ context.Context, _ string, stake int64, r economy.Result) (economy.Settlement, error) {
	return economy.Settle(stake, r), nil
}

func bankContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), bankTimeout)
}

func (freeBank) RefundGame(context.Context, string) (bool, error) {
	return true, nil
}
