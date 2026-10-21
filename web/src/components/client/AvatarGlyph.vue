<script setup lang="ts">
import { computed, watchEffect, type Component } from 'vue'
import {
  PhAlien,
  PhAnchor,
  PhBird,
  PhCactus,
  PhCat,
  PhClover,
  PhCrown,
  PhDetective,
  PhDog,
  PhFire,
  PhFish,
  PhGhost,
  PhHorse,
  PhLightning,
  PhRabbit,
  PhRobot,
  PhRocket,
  PhSkull,
  PhSunglasses,
  PhSword,
  PhTrophy,
} from '@phosphor-icons/vue'
import { avatarInitial } from '../../composables/avatar'
import { useAuthStore } from '../../stores/auth'
import { useAvatarsStore } from '../../stores/avatars'
import { useSettingsStore } from '../../stores/settings'

const ICONS: Record<string, Component> = {
  cat: PhCat,
  dog: PhDog,
  rabbit: PhRabbit,
  fish: PhFish,
  bird: PhBird,
  cactus: PhCactus,
  ghost: PhGhost,
  robot: PhRobot,
  alien: PhAlien,
  sunglasses: PhSunglasses,
  clover: PhClover,
  lightning: PhLightning,
  fire: PhFire,
  anchor: PhAnchor,
  rocket: PhRocket,
  horse: PhHorse,
  sword: PhSword,
  detective: PhDetective,
  skull: PhSkull,
  trophy: PhTrophy,
  crown: PhCrown,
}

const SUITS: Record<string, string> = { spade: '♠', heart: '♥', diamond: '♦', club: '♣' }

const props = defineProps<{ name: string; userId?: string; avatar?: string; self?: boolean }>()

const auth = useAuthStore()
const avatars = useAvatarsStore()
const settings = useSettingsStore()

const isMe = computed(() => props.self || (!!props.userId && props.userId === auth.user?.id))

watchEffect(() => {
  if (!props.avatar && !isMe.value) avatars.want(props.userId)
})

const id = computed(() => props.avatar ?? (isMe.value ? settings.avatar : avatars.of(props.userId)) ?? 'initial')
const icon = computed(() => ICONS[id.value])
const suit = computed(() => SUITS[id.value])
</script>

<template>
  <component :is="icon" v-if="icon" class="glyph icon" weight="fill" aria-hidden="true" />
  <span v-else-if="suit" class="glyph suit" aria-hidden="true">{{ suit }}</span>
  <template v-else>{{ avatarInitial(name) }}</template>
</template>

<style scoped>
.icon {
  width: 58%;
  height: 58%;
}

.suit {
  font-family: var(--font-card);
  font-size: 1.25em;
  line-height: 1;
}
</style>
