<script setup lang="ts">
import { ref } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { PhSmiley } from '@phosphor-icons/vue'
import AnimatedEmoji from './AnimatedEmoji.vue'
import { EMOJI, QUICK_PHRASES } from '../../constants/emoji'

defineProps<{ disabled?: boolean }>()
const emit = defineEmits<{ (e: 'send', text: string): void }>()

const open = ref(false)
const hovered = ref<string | null>(null)

function pick(text: string) {
  emit('send', text)
  open.value = false
}
</script>

<template>
  <div class="quick">
    <AnimatePresence>
      <motion.div
        v-if="open"
        class="palette"
        role="listbox"
        aria-label="Эмодзи"
        :initial="{ opacity: 0, y: 8, scale: 0.97 }"
        :animate="{ opacity: 1, y: 0, scale: 1 }"
        :exit="{ opacity: 0, y: 8, scale: 0.97 }"
        :transition="{ duration: 0.16 }"
      >
        <button
          v-for="e in EMOJI"
          :key="e.code"
          type="button"
          class="emoji-btn"
          :title="e.label"
          :aria-label="e.label"
          :disabled="disabled"
          @mouseenter="hovered = e.code"
          @mouseleave="hovered = null"
          @focus="hovered = e.code"
          @blur="hovered = null"
          @click="pick(e.char)"
        >
          <AnimatedEmoji :emoji="e" :size="34" :animated="hovered === e.code" />
        </button>
      </motion.div>
    </AnimatePresence>

    <div class="row">
      <button
        type="button"
        class="toggle"
        :class="{ on: open }"
        :aria-expanded="open"
        aria-label="Эмодзи"
        title="Эмодзи"
        @click="open = !open"
      >
        <PhSmiley :size="20" :weight="open ? 'fill' : 'regular'" />
      </button>
      <div class="phrases">
        <button v-for="p in QUICK_PHRASES" :key="p" type="button" class="phrase" :disabled="disabled" @click="pick(p)">
          {{ p }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.quick {
  position: relative;
  padding-top: 8px;
}

.row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.toggle {
  display: grid;
  place-items: center;
  flex: none;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.toggle.on,
.toggle:hover {
  color: var(--color-gold-ink);
}

.toggle.on {
  background: var(--color-gold-tint);
}

.phrases {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}

.phrases::-webkit-scrollbar {
  display: none;
}

.phrase {
  flex: none;
  height: 32px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
  cursor: pointer;
}

.phrase:hover:not(:disabled) {
  color: var(--color-ink);
  box-shadow: inset 0 0 0 1px var(--color-line-heavy);
}

.palette {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(100% + 6px);
  z-index: 5;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 4px;
  padding: 8px;
  border-radius: 18px;
  background: var(--color-paper-raised);
  box-shadow: var(--shadow-panel);
}

.emoji-btn {
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  border: 0;
  border-radius: 12px;
  background: transparent;
  cursor: pointer;
  transition:
    background-color 0.15s,
    transform 0.15s;
}

.emoji-btn:hover:not(:disabled) {
  background: var(--color-paper-hover);
  transform: scale(1.12);
}
</style>
