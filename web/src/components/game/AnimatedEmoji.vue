<script setup lang="ts">
import { computed, ref } from 'vue'
import { animatedEmojiUrl, staticEmojiUrl, type EmojiEntry } from '../../constants/emoji'

const props = withDefaults(defineProps<{ emoji: EmojiEntry; size?: number; animated?: boolean }>(), {
  size: 32,
  animated: true,
})

const failed = ref(false)
const src = computed(() => (props.animated ? animatedEmojiUrl(props.emoji.code) : staticEmojiUrl(props.emoji.code)))
</script>

<template>
  <span class="emoji" :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${size * 0.8}px` }">
    <span v-if="failed" aria-hidden="true">{{ emoji.char }}</span>
    <img v-else :src="src" :alt="emoji.char" :width="size" :height="size" decoding="async" @error="failed = true" />
  </span>
</template>

<style scoped>
.emoji {
  display: inline-grid;
  place-items: center;
  flex: none;
  line-height: 1;
}

img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
