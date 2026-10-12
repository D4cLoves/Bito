<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), { size: 20 })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" class="chip" aria-hidden="true">
    <circle cx="12" cy="12" r="10.5" fill="currentColor" />
    <circle cx="12" cy="12" r="10.5" fill="none" stroke="var(--color-gold-tint)" stroke-width="2.6" stroke-dasharray="3.4 4.85" />
    <circle cx="12" cy="12" r="6.2" fill="none" stroke="var(--color-gold-tint)" stroke-width="1.2" />
    <path d="M12 8.6l1.2 2.2 2.2 1.2-2.2 1.2-1.2 2.2-1.2-2.2-2.2-1.2 2.2-1.2z" fill="var(--color-gold-tint)" />
  </svg>
</template>

<style scoped>
.chip {
  flex: none;
  color: var(--color-gold);
}
</style>
