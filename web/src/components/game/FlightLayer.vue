<script setup lang="ts">
import FlyingCard from './FlyingCard.vue'
import type { Flight, FlightKind } from '../../composables/useCardFlights'

defineProps<{ flights: Flight[] }>()
defineEmits<{
  (e: 'land', kind: FlightKind): void
  (e: 'reveal', id: number): void
  (e: 'done', id: number): void
}>()
</script>

<template>
  <div class="flight-layer" aria-hidden="true">
    <FlyingCard
      v-for="f in flights"
      :key="f.id"
      :flight="f"
      @land="(kind) => $emit('land', kind)"
      @reveal="(id) => $emit('reveal', id)"
      @done="(id) => $emit('done', id)"
    />
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
