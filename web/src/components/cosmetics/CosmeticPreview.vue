<script setup lang="ts">
import CardBack from '../client/CardBack.vue'
import PaperCard from '../landing/PaperCard.vue'
import type { CardBackId, CosmeticKind } from '../../constants/cosmetics'

defineProps<{ kind: CosmeticKind; id: string }>()
</script>

<template>
  <div class="preview" :class="`is-${kind}`" :data-theme="kind === 'theme' ? id : undefined" aria-hidden="true">
    <div v-if="kind === 'back'" class="fan">
      <span v-for="i in 3" :key="i" class="fan-card" :class="`c${i}`"><CardBack :variant="id as CardBackId" /></span>
    </div>

    <div v-else-if="kind === 'felt'" class="table" :data-felt="id">
      <span class="rim"></span>
      <span class="mini m1"><PaperCard rank="Q" suit="♥" /></span>
      <span class="mini m2"><PaperCard rank="A" suit="♠" /></span>
    </div>

    <div v-else class="ui">
      <span class="ui-rail"><i></i><i></i><i></i></span>
      <span class="ui-main">
        <span class="ui-hero"><i class="ui-btn"></i></span>
        <span class="ui-tiles"><i></i><i></i><i></i></span>
      </span>
      <span class="ui-side"><i></i><i></i><i></i></span>
    </div>
  </div>
</template>

<style scoped>
.preview {
  position: relative;
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.fan {
  position: relative;
  width: 38%;
  aspect-ratio: 5 / 7;
}

.fan-card {
  position: absolute;
  inset: 0;
  transform-origin: 50% 110%;
  transition: transform 0.45s cubic-bezier(0.22, 1, 0.36, 1);
}

.c1 {
  transform: rotate(-14deg) translateX(-18%);
}

.c2 {
  transform: translateY(-4%);
}

.c3 {
  transform: rotate(14deg) translateX(18%);
}

.table {
  position: relative;
  width: 82%;
  aspect-ratio: 2 / 1;
  border-radius: 999px;
  background:
    radial-gradient(ellipse 70% 60% at 50% 35%, rgba(255, 255, 255, 0.14), transparent 70%),
    radial-gradient(ellipse at 50% 50%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    0 0 0 5px #fbfaf6,
    0 0 0 6px rgba(0, 0, 0, 0.12),
    inset 0 0 22px color-mix(in srgb, var(--color-felt-deep) 70%, transparent),
    0 18px 26px -16px rgba(0, 0, 0, 0.55);
}

.rim {
  position: absolute;
  inset: 9%;
  border-radius: 999px;
  border: 1px dashed var(--color-felt-line);
}

.mini {
  position: absolute;
  top: 26%;
  width: 17%;
  aspect-ratio: 5 / 7;
  font-size: 7px;
  transition: transform 0.45s cubic-bezier(0.22, 1, 0.36, 1);
}

.mini :deep(.index) {
  font-size: 8px;
  left: 3px;
  top: 2px;
}

.mini :deep(.index--bottom) {
  display: none;
}

.mini :deep(.pip) {
  font-size: 16px;
}

.m1 {
  left: 37%;
  transform: rotate(-8deg);
}

.m2 {
  left: 47%;
  transform: rotate(9deg) translateY(6%);
}

.ui {
  display: flex;
  gap: 5%;
  width: 84%;
  height: 74%;
  padding: 5%;
  box-sizing: border-box;
  border-radius: 12px;
  background: var(--color-paper);
  box-shadow:
    inset 0 0 0 1px var(--color-line-strong),
    0 18px 26px -18px rgba(0, 0, 0, 0.55);
  transition: transform 0.45s cubic-bezier(0.22, 1, 0.36, 1);
}

.ui-rail,
.ui-side {
  display: flex;
  flex-direction: column;
  gap: 8%;
}

.ui-rail {
  width: 9%;
  padding-top: 2%;
}

.ui-rail i {
  aspect-ratio: 1;
  border-radius: 30%;
  background: var(--color-paper-hover);
}

.ui-rail i:first-child {
  background: var(--color-gold);
}

.ui-main {
  display: flex;
  flex-direction: column;
  gap: 8%;
  flex: 1;
}

.ui-hero {
  position: relative;
  flex: 1.4;
  border-radius: 8px;
  background: radial-gradient(ellipse at 50% 30%, var(--color-gold-tint), var(--color-paper-sunken));
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.ui-btn {
  position: absolute;
  right: 10%;
  bottom: 16%;
  width: 34%;
  height: 22%;
  border-radius: 999px;
  background: var(--color-gold);
}

.ui-tiles {
  display: flex;
  gap: 6%;
  flex: 1;
}

.ui-tiles i,
.ui-side i {
  flex: 1;
  border-radius: 6px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.ui-side {
  width: 24%;
}
</style>
