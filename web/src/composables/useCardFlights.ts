import { nextTick, onUnmounted, ref, watch, type Ref, type ShallowRef } from 'vue'
import { cardKey, type Card, type GameState } from '../types/game'

export type FlightKind = 'throw' | 'deal' | 'sweep' | 'take'

export interface Box {
  x: number
  y: number
  w: number
  angle: number
}

export interface Flight {
  id: number
  kind: FlightKind
  card: Card | null
  faceDown: boolean
  flip: boolean
  fade: boolean
  from: Box
  to: Box
  delay: number
}

type PlannedFlight = Omit<Flight, 'id'> & { target?: HTMLElement | null }

const PILE_CARD_W = 58
const SEAT_CARD_W = 26
const MAX_FLIGHTS = 40
const THROW_STAGGER = 150
const SWEEP_STAGGER = 40
const TAKE_STAGGER = 35
const SWEEP_MS = 660
const DEAL_STAGGER_MAX = 95
const DEAL_WINDOW = 1300
const SAFETY_MS = 2600

function angleOf(el: HTMLElement) {
  let angle = 0
  let node: HTMLElement | null = el
  for (let depth = 0; node && depth < 3; depth++, node = node.parentElement) {
    const style = getComputedStyle(node)
    if (style.transform && style.transform !== 'none') {
      const m = new DOMMatrix(style.transform)
      angle += (Math.atan2(m.b, m.a) * 180) / Math.PI
    }
    if (style.rotate && style.rotate !== 'none') angle += parseFloat(style.rotate)
  }
  return angle
}

function boxOf(el: HTMLElement, w = el.offsetWidth, angle = angleOf(el)): Box {
  const r = el.getBoundingClientRect()
  return { x: r.left + r.width / 2, y: r.top + r.height / 2, w, angle }
}

function tableCards(state: GameState | null | undefined): Card[] {
  return (state?.table ?? []).flatMap((p) => (p.defend ? [p.attack, p.defend] : [p.attack]))
}

export function useCardFlights(root: Ref<HTMLElement | null>, state: ShallowRef<GameState | null>, me: Ref<string>) {
  const flights = ref<Flight[]>([])
  const targets = new Map<number, HTMLElement>()
  const safety = new Map<number, number>()
  let seq = 0

  const enabled = !window.matchMedia('(prefers-reduced-motion: reduce)').matches

  function find(selector: string) {
    return root.value?.querySelector<HTMLElement>(selector) ?? null
  }

  function cardEl(key: string) {
    return find(`[data-fly="${CSS.escape(key)}"]`)
  }

  function seatBox(id: string | undefined, w: number): Box | null {
    if (!id) return null
    const seat = find(`[data-fly-seat="${CSS.escape(id)}"]`)
    if (!seat) return null
    const backs = seat.querySelector<HTMLElement>('.backs')
    const anchor = backs && backs.offsetWidth ? backs : (seat.querySelector<HTMLElement>('.avatar-wrap') ?? seat)
    return boxOf(anchor, w, 0)
  }

  function pileBox(kind: 'deck' | 'discard'): Box | null {
    const pile = find(`[data-fly-${kind}]`)
    const backs = kind === 'deck' ? pile?.querySelectorAll<HTMLElement>('.deck-back') : null
    const el = backs?.length ? backs[backs.length - 1] : pile
    return el ? boxOf(el, PILE_CARD_W, kind === 'discard' ? 14 : 0) : null
  }

  function launch(planned: PlannedFlight[]) {
    for (const { target, ...flight } of planned.slice(0, MAX_FLIGHTS)) {
      const id = ++seq
      if (target) {
        target.style.visibility = 'hidden'
        targets.set(id, target)
      }
      safety.set(id, window.setTimeout(() => done(id), flight.delay + SAFETY_MS))
      flights.value.push({ ...flight, id })
    }
  }

  function reveal(id: number) {
    targets.get(id)?.style.removeProperty('visibility')
    targets.delete(id)
  }

  function done(id: number) {
    reveal(id)
    clearTimeout(safety.get(id))
    safety.delete(id)
    flights.value = flights.value.filter((f) => f.id !== id)
  }

  function plan(
    next: GameState,
    prev: GameState | null,
    prevHand: Set<string>,
    prevTable: Card[],
    handBoxes: Map<string, Box>,
    tableBoxes: Map<string, Box>,
  ) {
    const planned: PlannedFlight[] = []
    const prevTableKeys = new Set(prevTable.map(cardKey))
    const nextTable = next.table ?? []

    const decreased: string[] = []
    if (prev) {
      for (const p of next.players) {
        if (p.id === me.value) continue
        const before = prev.players.find((x) => x.id === p.id)?.handCount ?? p.handCount
        for (let i = 0; i < before - p.handCount; i++) decreased.push(p.id)
      }
    }

    let throws = 0
    for (const pair of nextTable) {
      const placed: [Card | undefined, boolean][] = [
        [pair.attack, false],
        [pair.defend, true],
      ]
      for (const [card, isDefend] of placed) {
        if (!card) continue
        const key = cardKey(card)
        if (prevTableKeys.has(key)) continue
        const target = cardEl(key)
        if (!target) continue
        const to = boxOf(target)
        let from: Box | null = handBoxes.get(key) ?? null
        let faceDown = false
        if (!from) {
          let who: string | undefined
          if (isDefend) {
            who = next.defenderId
            const i = decreased.indexOf(who)
            if (i >= 0) decreased.splice(i, 1)
          } else {
            const i = decreased.findIndex((id) => id !== next.defenderId)
            who = i >= 0 ? decreased.splice(i, 1)[0] : next.attackerId
          }
          from = seatBox(who, to.w * 0.45)
          faceDown = true
        }
        if (!from) continue
        planned.push({ kind: 'throw', card, faceDown, flip: faceDown, fade: false, from, to, delay: throws++ * THROW_STAGGER + (key.charCodeAt(key.length - 1) % 5) * 8, target })
      }
    }

    const cleared = !!prev && prevTable.length > 0 && nextTable.length === 0
    const swept = cleared && next.discardSize > (prev?.discardSize ?? 0)
    const takenBy = cleared && !swept ? prev!.defenderId : null

    if (cleared) {
      const discard = swept ? pileBox('discard') : null
      prevTable.forEach((card, i) => {
        const from = tableBoxes.get(cardKey(card))
        if (!from) return
        if (swept && discard) {
          const to = { ...discard, angle: discard.angle + (i % 2 ? -7 : 7) }
          planned.push({ kind: 'sweep', card, faceDown: false, flip: true, fade: true, from, to, delay: i * SWEEP_STAGGER })
        } else if (takenBy && takenBy !== me.value) {
          const to = seatBox(takenBy, SEAT_CARD_W)
          if (to) planned.push({ kind: 'take', card, faceDown: false, flip: true, fade: true, from, to, delay: i * TAKE_STAGGER })
        }
      })
    }

    const queues = new Map<string, PlannedFlight[]>()
    const deck = pileBox('deck')
    const myQueue: PlannedFlight[] = []
    let takenIntoHand = 0
    for (const card of next.myHand ?? []) {
      const key = cardKey(card)
      if (prevHand.has(key)) continue
      const target = cardEl(key)
      if (!target) continue
      const to = boxOf(target)
      const fromTable = tableBoxes.get(key)
      if (fromTable) {
        planned.push({ kind: 'take', card, faceDown: false, flip: false, fade: false, from: fromTable, to, delay: takenIntoHand++ * TAKE_STAGGER, target })
      } else if (deck) {
        myQueue.push({ kind: 'deal', card, faceDown: true, flip: true, fade: false, from: deck, to, delay: 0, target })
      }
    }
    myQueue.sort((a, b) => a.to.x - b.to.x)
    if (myQueue.length) queues.set(me.value, myQueue)

    if (deck) {
      for (const p of next.players) {
        if (p.id === me.value) continue
        const before = prev?.players.find((x) => x.id === p.id)?.handCount ?? 0
        let gained = p.handCount - before
        if (takenBy === p.id) gained -= prevTable.length
        const to = seatBox(p.id, SEAT_CARD_W)
        if (!to || gained <= 0) continue
        queues.set(
          p.id,
          Array.from({ length: Math.min(gained, 6) }, () => ({
            kind: 'deal' as const,
            card: null,
            faceDown: true,
            flip: false,
            fade: true,
            from: deck,
            to,
            delay: 0,
          })),
        )
      }
    }

    const sweepEnd = swept ? (prevTable.length - 1) * SWEEP_STAGGER + SWEEP_MS * 0.75 + 120 : 0
    const takeEnd = cleared && !swept ? (prevTable.length - 1) * TAKE_STAGGER + 420 : 0
    const base = Math.max(sweepEnd, takeEnd, throws * THROW_STAGGER)
    const dealCount = [...queues.values()].reduce((sum, q) => sum + q.length, 0)
    const dealStagger = Math.min(DEAL_STAGGER_MAX, DEAL_WINDOW / Math.max(1, dealCount))
    let dealt = 0
    for (let round = 0; [...queues.values()].some((q) => q.length > round); round++) {
      for (const queue of queues.values()) {
        const flight = queue[round]
        if (flight) planned.push({ ...flight, delay: base + dealt++ * dealStagger })
      }
    }

    launch(planned)
  }

  function locations(s: GameState | null | undefined) {
    const map = new Map<string, string>()
    for (const c of s?.myHand ?? []) map.set(cardKey(c), 'hand')
    for (const c of tableCards(s)) map.set(cardKey(c), 'table')
    return map
  }

  function settleMoving(next: GameState, prev: GameState | null | undefined) {
    if (!flights.value.length) return
    const before = locations(prev)
    const after = locations(next)
    for (const f of [...flights.value]) {
      if (!f.card) continue
      const key = cardKey(f.card)
      if (before.has(key) && before.get(key) !== after.get(key)) done(f.id)
    }
  }

  watch(state, (next, prev) => {
    if (!next || !root.value || !enabled) return
    settleMoving(next, prev)
    const prevHand = new Set((prev?.myHand ?? []).map(cardKey))
    const prevTable = tableCards(prev)
    const handBoxes = new Map<string, Box>()
    const tableBoxes = new Map<string, Box>()
    for (const key of prevHand) {
      const el = cardEl(key)
      if (el) handBoxes.set(key, boxOf(el))
    }
    for (const card of prevTable) {
      const el = cardEl(cardKey(card))
      if (el) tableBoxes.set(cardKey(card), boxOf(el))
    }
    nextTick(() => {
      targets.forEach((el, id) => {
        if (!el.isConnected) done(id)
      })
      plan(next, prev ?? null, prevHand, prevTable, handBoxes, tableBoxes)
    })
  })

  onUnmounted(() => {
    safety.forEach((t) => clearTimeout(t))
    targets.forEach((el) => el.style.removeProperty('visibility'))
  })

  return { flights, reveal, done, enabled }
}
