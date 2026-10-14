<script setup lang="ts" generic="T extends string | number">
import { computed, ref, useId } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { PhCaretDown, PhCheck, PhLockSimple } from '@phosphor-icons/vue'
import StakeChip from './StakeChip.vue'
import type { PickOption } from '../../types/ui'

const props = withDefaults(defineProps<{ label: string; options: PickOption<T>[]; align?: 'start' | 'end'; tone?: 'felt' | 'paper' }>(), {
  align: 'start',
  tone: 'felt',
})
const model = defineModel<T>({ required: true })
const emit = defineEmits<{ (e: 'change', value: T): void }>()

const open = ref(false)
const active = ref(-1)
const root = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const listId = `pick-${useId()}`

const current = computed(() => props.options.find((o) => o.value === model.value))

onClickOutside(root, () => close())

function show() {
  open.value = true
  active.value = Math.max(0, props.options.findIndex((o) => o.value === model.value))
}

function close(refocus = false) {
  open.value = false
  if (refocus) trigger.value?.focus()
}

function toggle() {
  if (open.value) close()
  else show()
}

function choose(i: number) {
  const o = props.options[i]
  if (!o || o.disabled) return
  const changed = o.value !== model.value
  model.value = o.value
  close(true)
  if (changed) emit('change', o.value)
}

function move(step: number) {
  const n = props.options.length
  let i = active.value
  for (let k = 0; k < n; k++) {
    i = (i + step + n) % n
    if (!props.options[i].disabled) {
      active.value = i
      return
    }
  }
}

function onKey(e: KeyboardEvent) {
  if (!open.value) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      show()
    }
    return
  }
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      move(1)
      break
    case 'ArrowUp':
      e.preventDefault()
      move(-1)
      break
    case 'Enter':
    case ' ':
      e.preventDefault()
      choose(active.value)
      break
    case 'Escape':
      e.preventDefault()
      e.stopPropagation()
      close(true)
      break
    case 'Tab':
      close()
      break
  }
}
</script>

<template>
  <div ref="root" class="pick" :class="[tone, { open }]">
    <button
      ref="trigger"
      type="button"
      class="trigger"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-controls="listId"
      :aria-activedescendant="open && active >= 0 ? `${listId}-${active}` : undefined"
      @click="toggle"
      @keydown="onKey"
    >
      <span class="label">{{ label }}</span>
      <span class="value">
        <StakeChip v-if="current?.chip" :value="current.chip" :size="22" :label="false" />
        <span v-else-if="current?.suit" class="suit" :class="{ red: current.red }" aria-hidden="true">{{ current.suit }}</span>
        <span class="text">{{ current?.short ?? current?.label }}</span>
        <PhCaretDown class="caret" :size="14" weight="bold" aria-hidden="true" />
      </span>
    </button>

    <Transition name="menu">
      <ul v-if="open" :id="listId" class="menu" :class="align" role="listbox" :aria-label="label">
        <li
          v-for="(o, i) in options"
          :id="`${listId}-${i}`"
          :key="String(o.value)"
          role="option"
          class="option"
          :class="{ active: i === active, selected: o.value === model, disabled: o.disabled }"
          :aria-selected="o.value === model"
          :aria-disabled="o.disabled || undefined"
          @mouseenter="!o.disabled && (active = i)"
          @click="choose(i)"
        >
          <StakeChip v-if="o.chip" :value="o.chip" :size="30" />
          <span v-else-if="o.suit" class="suit" :class="{ red: o.red }" aria-hidden="true">{{ o.suit }}</span>
          <span v-else class="suit blank" aria-hidden="true"></span>
          <span class="opt-text">
            <b>{{ o.label }}</b>
            <small v-if="o.hint">{{ o.hint }}</small>
          </span>
          <PhLockSimple v-if="o.disabled" class="mark" :size="15" weight="bold" aria-hidden="true" />
          <PhCheck v-else-if="o.value === model" class="mark ok" :size="16" weight="bold" aria-hidden="true" />
        </li>
      </ul>
    </Transition>
  </div>
</template>

<style scoped>
.pick {
  position: relative;
  min-width: 0;
}

.trigger {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  width: 100%;
  min-height: 60px;
  padding: 9px 12px 10px;
  box-sizing: border-box;
  border: 0;
  border-radius: 18px;
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
  color: var(--color-on-felt);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: box-shadow 0.2s;
}

.trigger:hover,
.open .trigger {
  box-shadow: inset 0 0 0 1.5px var(--color-on-felt-faint);
}

.trigger:focus-visible {
  outline: 2px solid var(--color-felt-target);
  outline-offset: 2px;
}

.paper .trigger {
  min-height: 0;
  padding: 7px 14px 8px;
  border-radius: 14px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink);
}

.paper .trigger:hover,
.paper.open .trigger {
  box-shadow: inset 0 0 0 1.5px var(--color-ink-ghost);
}

.paper .trigger:focus-visible {
  outline-color: var(--color-gold);
}

.paper .label {
  color: var(--color-ink-faint);
}

.paper .value {
  font-size: 14px;
}

.label {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-on-felt-faint);
}

.value {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
  font-size: 16px;
  font-weight: 500;
}

.text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.caret {
  flex: none;
  opacity: 0.6;
  transition: transform 0.2s;
}

.open .caret {
  transform: rotate(180deg);
}

.suit {
  flex: none;
  font-family: var(--font-card);
  font-size: 18px;
  line-height: 1;
}

.value .suit.red {
  color: var(--color-felt-minus);
}

.menu {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 40;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 100%;
  width: max-content;
  max-width: 300px;
  max-height: 340px;
  margin: 0;
  padding: 6px;
  box-sizing: border-box;
  overflow-y: auto;
  border-radius: 18px;
  background: var(--color-paper-raised);
  box-shadow: var(--shadow-panel);
  color: var(--color-ink);
  list-style: none;
}

.menu.end {
  left: auto;
  right: 0;
}

.option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 12px;
  cursor: pointer;
}

.option.active {
  background: var(--color-paper-hover);
}

.option.disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.option .suit {
  display: grid;
  place-items: center;
  width: 30px;
  font-size: 20px;
}

.option .suit.red {
  color: var(--color-red);
}

.opt-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.opt-text b {
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
}

.opt-text small {
  font-size: 12px;
  color: var(--color-ink-faint);
  white-space: nowrap;
}

.mark {
  flex: none;
  color: var(--color-ink-faint);
}

.mark.ok {
  color: var(--color-gold-ink);
}

.menu-enter-active,
.menu-leave-active {
  transition:
    opacity 0.16s,
    transform 0.16s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}
</style>
