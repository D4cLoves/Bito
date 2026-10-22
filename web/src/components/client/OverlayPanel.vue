<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { motion } from 'motion-v'
import { PhX } from '@phosphor-icons/vue'

const props = withDefaults(defineProps<{ title: string; side?: 'left' | 'right'; width?: number }>(), {
  side: 'left',
  width: 720,
})

const emit = defineEmits<{ (e: 'close'): void }>()

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))

const offset = props.side === 'left' ? -60 : 60
</script>

<template>
  <div class="overlay" :class="side">
    <motion.div
      class="scrim"
      :initial="{ opacity: 0 }"
      :animate="{ opacity: 1 }"
      :exit="{ opacity: 0 }"
      :transition="{ duration: 0.2 }"
      @click="emit('close')"
    />
    <motion.section
      class="sheet"
      role="dialog"
      aria-modal="true"
      :aria-label="title"
      :style="{ '--w': `${width}px` }"
      :initial="{ opacity: 0, x: offset }"
      :animate="{ opacity: 1, x: 0 }"
      :exit="{ opacity: 0, x: offset }"
      :transition="{ type: 'spring', stiffness: 380, damping: 36 }"
    >
      <header class="sheet-head">
        <h2>{{ title }}</h2>
        <slot name="head" />
        <button type="button" class="close" aria-label="Закрыть" @click="emit('close')">
          <PhX :size="20" />
        </button>
      </header>
      <div class="sheet-body">
        <slot />
      </div>
    </motion.section>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
}

.overlay.left {
  left: var(--rail-w, 0px);
}

.overlay.right {
  justify-content: flex-end;
}

.scrim {
  position: absolute;
  inset: 0;
  background: var(--color-scrim);
  backdrop-filter: blur(3px);
}

.sheet {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(var(--w), 100%);
  height: 100%;
  background: var(--color-paper);
  box-shadow: var(--shadow-panel);
}

.sheet-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 22px 28px 14px;
}

h2 {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  margin: 0;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-family: var(--font-card);
  font-size: 30px;
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
}

.close {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  margin-left: auto;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.close:hover {
  color: var(--color-ink);
}

.sheet-body {
  flex: 1;
  min-height: 0;
  padding: 8px 28px 32px;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 767px) {
  .overlay.left,
  .overlay.right {
    left: 0;
    bottom: var(--rail-h, 0px);
  }

  .sheet-head {
    gap: 10px;
    padding: 14px 16px 10px;
  }

  h2 {
    font-size: 24px;
  }

  .sheet-body {
    padding: 4px 16px 24px;
  }
}
</style>
