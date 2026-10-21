<script setup lang="ts">
import { PhCheck, PhInfo, PhWarningCircle } from '@phosphor-icons/vue'
import { useToast } from '../../composables/useToast'

const { toasts, dismiss } = useToast()
</script>

<template>
  <div class="toast-stack" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <button v-for="t in toasts" :key="t.id" type="button" class="toast" :class="t.tone" @click="dismiss(t.id)">
        <PhCheck v-if="t.tone === 'success'" :size="17" weight="bold" aria-hidden="true" />
        <PhWarningCircle v-else-if="t.tone === 'error'" :size="17" weight="bold" aria-hidden="true" />
        <PhInfo v-else :size="17" weight="bold" aria-hidden="true" />
        <span>{{ t.text }}</span>
      </button>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-stack {
  position: fixed;
  z-index: 400;
  left: 50%;
  bottom: calc(var(--rail-h, 0px) + 24px);
  translate: -50% 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  width: max-content;
  max-width: calc(100% - 32px);
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink);
  color: var(--color-paper);
  font: inherit;
  font-size: 15px;
  text-align: left;
  box-shadow: 0 18px 40px -16px rgba(27, 25, 22, 0.6);
  pointer-events: auto;
  cursor: pointer;
}

.toast.success svg {
  color: var(--color-green-soft);
}

.toast.error svg {
  color: var(--color-red-soft);
}

.toast-enter-active,
.toast-leave-active {
  transition:
    opacity 0.3s,
    transform 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.96);
}

.toast-move {
  transition: transform 0.3s;
}
</style>
