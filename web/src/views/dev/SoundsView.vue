<script setup lang="ts">
import { useGameSound } from '../../composables/useGameSound'
import { useSettingsStore } from '../../stores/settings'

const sound = useGameSound()
const settings = useSettingsStore()

const groups = [
  {
    title: 'Карты',
    items: [
      { label: 'Карта на стол', where: 'атака, защита, подкидывание', play: sound.playCardDrop },
      { label: 'Раздача', where: 'карта из колоды в руку', play: sound.playCardDeal },
      { label: 'Бито', where: 'карты уходят в отбой', play: sound.playCardSweep },
      { label: 'Взять', where: 'карты со стола в руку', play: sound.playCardTake },
      { label: 'Выбор карты', where: 'клик по карте в руке', play: sound.playCardPick },
    ],
  },
  {
    title: 'Партия',
    items: [
      { label: 'Ваш ход', where: 'вкладка свёрнута', play: sound.playYourTurn },
      { label: 'Победа', where: 'конец партии, вы не дурак', play: sound.playChipsWin },
      { label: 'Проигрыш', where: 'конец партии, вы дурак', play: sound.playLose },
      { label: 'Сообщение', where: 'чат за столом', play: sound.playMessage },
    ],
  },
  {
    title: 'Интерфейс',
    items: [
      { label: 'Клик', where: 'кнопки, переключатели', play: sound.playClick },
      { label: 'Успех', where: 'сохранено, вход, ничья', play: sound.playSuccess },
      { label: 'Ошибка', where: 'неверный ход, ошибка формы', play: sound.playError },
    ],
  },
]

function burst(fn: () => void, count: number, gap: number) {
  for (let i = 0; i < count; i++) window.setTimeout(fn, i * gap)
}
</script>

<template>
  <main class="board" :data-theme="settings.theme">
    <header>
      <h1>Звуки</h1>
      <p v-if="!settings.soundOn" class="warn">Звук выключен в настройках, кнопки молчат.</p>
    </header>

    <section v-for="g in groups" :key="g.title">
      <h2>{{ g.title }}</h2>
      <div class="grid">
        <button v-for="s in g.items" :key="s.label" type="button" @click="s.play()">
          <b>{{ s.label }}</b>
          <span>{{ s.where }}</span>
        </button>
      </div>
    </section>

    <section>
      <h2>Серии</h2>
      <div class="grid">
        <button type="button" @click="burst(sound.playCardDeal, 6, 70)">
          <b>Раздача 6 карт</b>
          <span>как в начале партии</span>
        </button>
        <button type="button" @click="burst(sound.playCardSweep, 4, 55)">
          <b>Бито, 4 карты</b>
          <span>отбой после хода</span>
        </button>
        <button type="button" @click="burst(sound.playCardTake, 4, 60)">
          <b>Взять 4 карты</b>
          <span>защитник берёт</span>
        </button>
        <button type="button" @click="burst(sound.playCardDrop, 3, 90)">
          <b>Три карты подряд</b>
          <span>подкидывание</span>
        </button>
      </div>
    </section>
  </main>
</template>

<style scoped>
.board {
  width: 100%;
  min-height: 100dvh;
  padding: 32px max(16px, calc((100% - 920px) / 2)) 64px;
  background: var(--color-paper);
  color: var(--color-ink);
  display: flex;
  flex-direction: column;
  gap: 28px;
}

h1 {
  font-size: 28px;
  font-weight: 700;
}

h2 {
  font-size: 13px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-ink-muted);
  margin-bottom: 10px;
}

.warn {
  margin-top: 6px;
  color: var(--color-red);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
}

button {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid var(--color-line);
  background: var(--color-paper-raised);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
}

button:hover {
  background: var(--color-paper-hover);
}

button:active {
  transform: scale(0.98);
}

button span {
  font-size: 13px;
  color: var(--color-ink-muted);
}
</style>
