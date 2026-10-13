<script setup lang="ts">
import { PhCheck } from '@phosphor-icons/vue'
import CosmeticPreview from './CosmeticPreview.vue'
import type { CosmeticKind } from '../../constants/cosmetics'

const KIND_LABEL: Record<CosmeticKind, string> = { back: 'Рубашка', felt: 'Сукно', theme: 'Тема' }

withDefaults(defineProps<{ kind: CosmeticKind; id: string; name: string; note?: string; active?: boolean; clickable?: boolean }>(), {
  note: undefined,
  active: false,
  clickable: false,
})

defineEmits<{ (e: 'pick'): void }>()
</script>

<template>
  <article class="tile" :class="{ active, clickable }" @click="clickable && $emit('pick')">
    <div class="stage">
      <CosmeticPreview :kind="kind" :id="id" />
      <span v-if="active" class="badge"><PhCheck :size="13" weight="bold" />Выбрано</span>
    </div>
    <div class="foot">
      <div class="meta">
        <small>{{ note ?? KIND_LABEL[kind] }}</small>
        <b>{{ name }}</b>
      </div>
      <slot />
    </div>
  </article>
</template>

<style scoped>
.tile {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  transition:
    transform 0.35s cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 0.35s;
}

.tile:hover {
  transform: translateY(-3px);
  box-shadow:
    inset 0 0 0 1px var(--color-line-stronger),
    0 22px 36px -26px rgba(0, 0, 0, 0.55);
}

.tile.clickable {
  cursor: pointer;
}

.tile.active {
  box-shadow:
    inset 0 0 0 2px var(--color-gold),
    0 22px 36px -26px rgba(0, 0, 0, 0.45);
}

.stage {
  position: relative;
  aspect-ratio: 5 / 4;
  margin: 6px 6px 0;
  overflow: hidden;
  border-radius: 17px;
  background:
    radial-gradient(ellipse 80% 70% at 50% 30%, color-mix(in srgb, var(--color-glow) 60%, transparent), transparent 70%),
    var(--color-paper-sunken);
}

.tile:hover .stage :deep(.c1) {
  transform: rotate(-22deg) translateX(-30%);
}

.tile:hover .stage :deep(.c3) {
  transform: rotate(22deg) translateX(30%);
}

.tile:hover .stage :deep(.c2) {
  transform: translateY(-10%);
}

.tile:hover .stage :deep(.m1) {
  transform: rotate(-14deg) translateX(-12%);
}

.tile:hover .stage :deep(.m2) {
  transform: rotate(16deg) translate(12%, 4%);
}

.tile:hover .stage :deep(.ui) {
  transform: scale(1.04);
}

.badge {
  position: absolute;
  top: 10px;
  left: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 9px 4px 7px;
  border-radius: 999px;
  background: var(--color-gold);
  color: var(--color-paper);
  font-size: 11px;
  font-weight: 600;
}

.foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px 14px;
}

.meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.meta small {
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.meta b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 500;
}
</style>
