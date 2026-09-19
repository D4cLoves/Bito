package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"bito/internal/game"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("==================================================")
	fmt.Println("             🃏 BITO: ДУРАК ОНЛАЙН 🃏             ")
	fmt.Println("==================================================")
	fmt.Println("Выберите режим игры:")
	fmt.Println("1. Подкидной дурак")
	fmt.Println("2. Переводной дурак")
	fmt.Print("Ваш выбор (1 или 2) [по умолчанию 1]: ")

	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)

	mode := game.ModePodkidnoy
	if modeInput == "2" {
		mode = game.ModePerevodnoy
		fmt.Println("Режим: ПЕРЕВОДНОЙ ДУРАК")
	} else {
		fmt.Println("Режим: ПОДКИДНОЙ ДУРАК")
	}

	pUser := game.NewPlayer("user", "Вы")
	pBot := game.NewPlayer("bot", "Компьютер")

	g, err := game.NewGame(mode, game.Deck36, []*game.Player{pUser, pBot})
	if err != nil {
		fmt.Printf("Ошибка запуска игры: %v\n", err)
		return
	}

	trump := g.Deck().Trump()
	fmt.Printf("\nКозырь партии: %v (%v)\n", trump, trump.Suit)
	fmt.Printf("Первым ходит: %s (наименьший козырь)\n", g.Attacker().Name())

	for g.Phase() != game.PhaseFinished {
		printGameState(g, pUser, pBot)

		isUserAttacker := g.Attacker().ID() == pUser.ID()
		isUserDefender := g.Defender().ID() == pUser.ID()

		if g.Phase() == game.PhaseGivingMore {
			if isUserDefender {
				fmt.Println("\n⏳ Вы взяли карты. Компьютер решает, подкидывать ли вдогонку...")
				handleBotGivingMore(g, pBot, reader)
			} else {
				fmt.Println("\n👉 Компьютер взял карты! Вы можете подкинуть вдогонку.")
				handleUserGivingMore(g, pUser, reader)
			}
			continue
		}

		if isUserAttacker {
			handleUserAttackTurn(g, pUser, pBot, reader)
		} else if isUserDefender {
			handleUserDefendTurn(g, pUser, pBot, reader)
		}
	}

	printGameOver(g, pUser, pBot)
}

func printGameState(g *game.Game, pUser, pBot *game.Player) {
	fmt.Println("\n--------------------------------------------------")
	fmt.Printf("Колода: %d карт | Бита: %d карт | Козырь: %v\n",
		g.Deck().CardsLeft(), len(g.Discard()), g.Deck().Trump())
	fmt.Printf("Карт у Компьютера: %d\n", pBot.HandCount())

	fmt.Println("\n--- СТОЛ ---")
	pairs := g.Table().Pairs()
	if len(pairs) == 0 {
		fmt.Println("  (стол пуст)")
	} else {
		for i, p := range pairs {
			if p.Defend != nil {
				fmt.Printf("  [%d] %v  ->  %v (побита)\n", i+1, p.Attack, *p.Defend)
			} else {
				fmt.Printf("  [%d] %v  ->  [ ? ] (НЕ ПОБИТА)\n", i+1, p.Attack)
			}
		}
	}

	pUser.SortHand(g.Deck().Trump().Suit)
	fmt.Println("\n--- ВАША РУКА ---")
	for i, c := range pUser.Hand() {
		fmt.Printf("[%d] %v   ", i+1, c)
	}
	fmt.Println()

	if g.Attacker().ID() == pUser.ID() {
		fmt.Println("👉 ВАШ ХОД (Вы атакуете)")
	} else {
		fmt.Println("🛡️ ВАША ЗАЩИТА (Компьютер атакует)")
	}
}

func handleUserAttackTurn(g *game.Game, pUser, pBot *game.Player, reader *bufio.Reader) {
	if g.Table().PairsCount() == 0 {
		fmt.Print("Выберите номер карты для первого хода: ")
		cardIdx := readNumber(reader)
		if cardIdx < 1 || cardIdx > pUser.HandCount() {
			fmt.Println("❌ Неверный номер карты!")
			return
		}
		card := pUser.Hand()[cardIdx-1]
		if err := g.Attack(pUser.ID(), card); err != nil {
			fmt.Printf("❌ Ошибка хода: %v\n", err)
			return
		}
		fmt.Printf("Вы походили: %v\n", card)
		botDefendOrTransfer(g, pBot)
		return
	}

	if g.Table().IsAllBeaten() {
		fmt.Print("Подкинуть карту (номер) или нажать БИТА ('b'/'pass'): ")
		input := readInput(reader)
		if input == "b" || input == "pass" || input == "p" {
			if err := g.Pass(pUser.ID()); err != nil {
				fmt.Printf("❌ Ошибка паса: %v\n", err)
				return
			}
			fmt.Println("✅ Бита! Карты уходят в сброс.")
			return
		}

		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > pUser.HandCount() {
			fmt.Println("❌ Введите корректный номер карты или 'b' для биты.")
			return
		}

		card := pUser.Hand()[num-1]
		if err := g.Attack(pUser.ID(), card); err != nil {
			fmt.Printf("❌ Нельзя подкинуть: %v\n", err)
			return
		}
		fmt.Printf("Вы подкинули: %v\n", card)
		botDefendOrTransfer(g, pBot)
	} else {
		fmt.Println("Ждем ответа Компьютера...")
		botDefendOrTransfer(g, pBot)
	}
}

func handleUserDefendTurn(g *game.Game, pUser, pBot *game.Player, reader *bufio.Reader) {
	if g.Table().PairsCount() == 0 {
		botAttack(g, pBot)
		return
	}

	canTransfer := g.Mode() == game.ModePerevodnoy && g.Table().UnbeatenCount() == g.Table().PairsCount()

	if canTransfer {
		fmt.Println("Действия: побить ('номер_атаки номер_карты'), перевести ('t номер_карты') или взять ('take'):")
	} else {
		fmt.Println("Действия: побить ('номер_атаки номер_карты') или взять ('take'):")
	}
	fmt.Print("> ")

	input := readInput(reader)

	if input == "take" || input == "t" && !canTransfer {
		if err := g.Take(pUser.ID()); err != nil {
			fmt.Printf("❌ Ошибка: %v\n", err)
			return
		}
		fmt.Println("Вы решили взять карты!")
		return
	}

	if canTransfer && strings.HasPrefix(input, "t ") {
		parts := strings.Fields(input)
		if len(parts) == 2 {
			cardIdx, err := strconv.Atoi(parts[1])
			if err == nil && cardIdx >= 1 && cardIdx <= pUser.HandCount() {
				card := pUser.Hand()[cardIdx-1]
				if err := g.Transfer(pUser.ID(), card); err != nil {
					fmt.Printf("❌ Нельзя перевести: %v\n", err)
					return
				}
				fmt.Printf("🔄 Вы перевели ход картой: %v!\n", card)
				return
			}
		}
		fmt.Println("❌ Неверный формат перевода. Пример: 't 1'")
		return
	}

	parts := strings.Fields(input)
	if len(parts) == 2 {
		attackIdx, err1 := strconv.Atoi(parts[0])
		defendIdx, err2 := strconv.Atoi(parts[1])

		if err1 == nil && err2 == nil &&
			attackIdx >= 1 && attackIdx <= g.Table().PairsCount() &&
			defendIdx >= 1 && defendIdx <= pUser.HandCount() {

			attackPair := g.Table().Pairs()[attackIdx-1]
			defendCard := pUser.Hand()[defendIdx-1]

			if err := g.Defend(pUser.ID(), attackPair.Attack, defendCard); err != nil {
				fmt.Printf("❌ Нельзя побить: %v\n", err)
				return
			}
			fmt.Printf("Вы покрыли %v картой %v!\n", attackPair.Attack, defendCard)

			botDecideTossOrPass(g, pBot)
			return
		}
	}

	fmt.Println("❌ Неверный ввод! Пример: '1 2' (покрыть атаку №1 вашей картой №2) или 'take'.")
}

func handleUserGivingMore(g *game.Game, pUser *game.Player, reader *bufio.Reader) {
	fmt.Println("Введите номер карты, чтобы подкинуть вдогонку, или 'done'/'p' чтобы закончить:")
	fmt.Print("> ")
	input := readInput(reader)

	if input == "done" || input == "p" || input == "pass" {
		_ = g.Pass(pUser.ID())
		fmt.Println("Раунд окончен. Компьютер забрал карты со стола.")
		return
	}

	cardIdx, err := strconv.Atoi(input)
	if err == nil && cardIdx >= 1 && cardIdx <= pUser.HandCount() {
		card := pUser.Hand()[cardIdx-1]
		if err := g.Attack(pUser.ID(), card); err != nil {
			fmt.Printf("❌ Нельзя подкинуть: %v\n", err)
		} else {
			fmt.Printf("Вы докинули: %v\n", card)
		}
	} else {
		fmt.Println("❌ Неверный ввод.")
	}
}

func handleBotGivingMore(g *game.Game, pBot *game.Player, reader *bufio.Reader) {
	tossed := false
	for _, card := range pBot.Hand() {
		if game.CanToss(g.Table(), card) && game.CanAddAttack(g.Table(), 6, g.IsFirstRound()) {
			if err := g.Attack(pBot.ID(), card); err == nil {
				fmt.Printf("Компьютер докинул вам: %v\n", card)
				tossed = true
				break
			}
		}
	}
	if !tossed {
		fmt.Println("Компьютер решил больше не докидывать.")
		_ = g.Pass(pBot.ID())
		fmt.Println("Вы забрали карты со стола.")
	}
}

func botAttack(g *game.Game, pBot *game.Player) {
	if pBot.HandCount() == 0 {
		return
	}
	trumpSuit := g.Deck().Trump().Suit
	pBot.SortHand(trumpSuit)

	cardToPlay := pBot.Hand()[0]
	if err := g.Attack(pBot.ID(), cardToPlay); err == nil {
		fmt.Printf("Компьютер походил: %v\n", cardToPlay)
	}
}

func botDefendOrTransfer(g *game.Game, pBot *game.Player) {
	trumpSuit := g.Deck().Trump().Suit

	if g.Mode() == game.ModePerevodnoy && g.Table().UnbeatenCount() == g.Table().PairsCount() {
		firstAttack, _ := g.Table().FirstAttack()
		for _, card := range pBot.Hand() {
			if card.Rank == firstAttack.Rank {
				if err := g.Transfer(pBot.ID(), card); err == nil {
					fmt.Printf("🔄 Компьютер перевел ход картой: %v!\n", card)
					return
				}
			}
		}
	}

	for _, pair := range g.Table().Pairs() {
		if pair.Defend == nil {
			beaten := false
			pBot.SortHand(trumpSuit)
			for _, defCard := range pBot.Hand() {
				if game.CanBeat(pair.Attack, defCard, trumpSuit) {
					if err := g.Defend(pBot.ID(), pair.Attack, defCard); err == nil {
						fmt.Printf("Компьютер побил %v картой %v\n", pair.Attack, defCard)
						beaten = true
						break
					}
				}
			}
			if !beaten {
				fmt.Println("Компьютер не может отбиться и берет карты!")
				_ = g.Take(pBot.ID())
				return
			}
		}
	}

	fmt.Println("Компьютер успешно отбил все карты!")
}

func botDecideTossOrPass(g *game.Game, pBot *game.Player) {
	if pBot.HandCount() == 0 {
		_ = g.Pass(pBot.ID())
		return
	}

	for _, card := range pBot.Hand() {
		if card.Suit != g.Deck().Trump().Suit {
			if err := g.Attack(pBot.ID(), card); err == nil {
				fmt.Printf("Компьютер подкинул: %v\n", card)
				return
			}
		}
	}

	fmt.Println("Компьютер пасует (Бита).")
	_ = g.Pass(pBot.ID())
}

func printGameOver(g *game.Game, pUser, pBot *game.Player) {
	fmt.Println("\n==================================================")
	fmt.Println("               ИГРА ОКОНЧЕНА!                     ")
	fmt.Println("==================================================")

	if g.IsDraw() {
		fmt.Println("🤝 НИЧЬЯ! Оба игрока одновременно избавились от карт.")
	} else if g.WinnerID() == pUser.ID() {
		fmt.Println("🏆 ПОЗДРАВЛЯЕМ! ВЫ ПОБЕДИЛИ!")
		fmt.Printf("Компьютер остался в дураках с картами: %v\n", pBot.Hand())
	} else {
		fmt.Println("🃏 ВЫ ОСТАЛИСЬ В ДУРАКАХ!")
		fmt.Printf("Ваши оставшиеся карты: %v\n", pUser.Hand())
	}
}

func readInput(reader *bufio.Reader) string {
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func readNumber(reader *bufio.Reader) int {
	input := readInput(reader)
	num, err := strconv.Atoi(input)
	if err != nil {
		return -1
	}
	return num
}
