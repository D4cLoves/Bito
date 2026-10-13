<script setup lang="ts">
import FlyingCard from './FlyingCard.vue'
import type { Flight } from '../../composables/useCardFlights'

defineProps<{ flights: Flight[] }>()
defineEmits<{ (e: 'done', id: number): void }>()
</script>

<template>
  <div class="flight-layer" aria-hidden="true">
    <FlyingCard v-for="f in flights" :key="f.id" :flight="f" @done="(id) => $emit('done', id)" />
  </div>
</template>

<style scoped>
.flight-layer {
  position: fixed;
  inset: 0;
  z-index: 90;
  pointer-events: none;
  overflow: hidden;
}
</style>
