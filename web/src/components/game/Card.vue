<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { type Card, getRankLabel, getSuitSymbol, isRedSuit } from '../../types/game'
import { useGameSound } from '../../composables/useGameSound'

interface Props {
  card: Card
  isTrump?: boolean
  isFaceDown?: boolean
  disabled?: boolean
  selected?: boolean
  size?: 'sm' | 'md' | 'lg' | 'hero'
  tiltEnabled?: boolean
  customRotation?: number
}

const props = withDefaults(defineProps<Props>(), {
  isTrump: false,
  isFaceDown: false,
  disabled: false,
  selected: false,
  size: 'md',
  tiltEnabled: true,
  customRotation: 0,
})

const emit = defineEmits<{
  (e: 'click', card: Card): void
  (e: 'hover', card: Card): void
}>()

const sound = useGameSound()
const cardEl = ref<HTMLElement | null>(null)

// 3D Tilt State
const tiltX = ref(0)
const tiltY = ref(0)
const glareX = ref(50)
const glareY = ref(50)
const isHovered = ref(false)

const rankLabel = computed(() => getRankLabel(props.card.rank))
const suitSymbol = computed(() => getSuitSymbol(props.card.suit))
const isRed = computed(() => isRedSuit(props.card.suit))

// Aspect ratio 1 : 1.45 (Classic Casino Poker card)
const sizeClasses = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'w-16 h-[96px] text-xs rounded-lg'
    case 'lg':
      return 'w-32 h-[192px] text-base rounded-2xl'
    case 'hero':
      return 'w-44 h-[260px] sm:w-48 sm:h-[285px] text-lg rounded-2xl'
    case 'md':
    default:
      return 'w-24 h-[142px] sm:w-28 sm:h-[165px] text-sm rounded-xl'
  }
})

function handleMouseMove(e: MouseEvent) {
  if (!props.tiltEnabled || props.disabled || !cardEl.value) return
  const rect = cardEl.value.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top

  const centerX = rect.width / 2
  const centerY = rect.height / 2

  const maxTilt = props.size === 'hero' ? 16 : 12
  tiltY.value = ((x - centerX) / centerX) * maxTilt
  tiltX.value = -((y - centerY) / centerY) * maxTilt

  glareX.value = Math.max(0, Math.min(100, (x / rect.width) * 100))
  glareY.value = Math.max(0, Math.min(100, (y / rect.height) * 100))
}

function handleMouseEnter() {
  if (props.disabled) return
  isHovered.value = true
  emit('hover', props.card)
}

function handleMouseLeave() {
  isHovered.value = false
  tiltX.value = 0
  tiltY.value = 0
  glareX.value = 50
  glareY.value = 50
}

function handleClick() {
  if (props.disabled) return
  sound.playCardDeal()
  emit('click', props.card)
}

const transformStyle = computed(() => {
  const baseRot = props.customRotation
  if (!isHovered.value || !props.tiltEnabled) {
    return props.selected
      ? `translate3d(0, -18px, 20px) rotate(${baseRot}deg) scale3d(1.04, 1.04, 1)`
      : `translate3d(0, 0, 0) rotate(${baseRot}deg) scale3d(1, 1, 1)`
  }

  const elevation = props.selected ? -22 : -10
  return `perspective(900px) translate3d(0, ${elevation}px, 30px) rotate(${baseRot}deg) rotateX(${tiltX.value}deg) rotateY(${tiltY.value}deg) scale3d(1.05, 1.05, 1)`
})

onUnmounted(() => {
  if (cardEl.value) {
    cardEl.value.style.transform = ''
  }
})
</script>

<template>
  <div
    ref="cardEl"
    :class="[
      'relative select-none transition-transform duration-200 ease-out will-change-transform cursor-pointer',
      sizeClasses,
      disabled ? 'opacity-30 cursor-not-allowed pointer-events-none' : 'hover:z-40',
      selected ? 'z-30' : '',
    ]"
    :style="{
      transform: transformStyle,
    }"
    @mousemove="handleMouseMove"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
    @click="handleClick"
  >
    <!-- FRONT OF CARD (QUIET LUXURY CASINO FINISH) -->
    <div
      v-if="!isFaceDown"
      :class="[
        'w-full h-full relative overflow-hidden rounded-[inherit] flex flex-col justify-between p-2.5 sm:p-3',
        'bg-[#0b0c10] border transition-colors duration-300',
        isTrump
          ? 'border-amber-400/60 shadow-[0_12px_36px_rgba(0,0,0,0.85),0_0_24px_rgba(251,191,36,0.15)]'
          : selected
            ? 'border-zinc-400 shadow-[0_12px_36px_rgba(0,0,0,0.9),0_0_20px_rgba(255,255,255,0.12)]'
            : 'border-white/[0.08] shadow-[0_10px_30px_rgba(0,0,0,0.85)] hover:border-white/[0.22]',
      ]"
    >
      <!-- Double Hairline Filigree Frame -->
      <div
        :class="[
          'absolute inset-1.5 rounded-[inherit] border pointer-events-none transition-opacity duration-300',
          isTrump ? 'border-amber-400/25 opacity-100' : 'border-white/[0.05] opacity-70',
        ]"
      >
        <!-- Fine Corner Florets -->
        <div class="absolute top-0.5 left-0.5 w-1 h-1 rounded-full bg-white/20"></div>
        <div class="absolute top-0.5 right-0.5 w-1 h-1 rounded-full bg-white/20"></div>
        <div class="absolute bottom-0.5 left-0.5 w-1 h-1 rounded-full bg-white/20"></div>
        <div class="absolute bottom-0.5 right-0.5 w-1 h-1 rounded-full bg-white/20"></div>
      </div>

      <!-- Trump Gold Seal (Refined, Discreet) -->
      <div
        v-if="isTrump"
        class="absolute top-2 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-amber-400/10 border border-amber-400/30 text-amber-300 text-[9px] font-mono tracking-widest uppercase font-semibold"
      >
        <span class="w-1 h-1 rounded-full bg-amber-400 animate-pulse"></span>
        <span>TRUMP</span>
      </div>

      <!-- Top-Left Corner Index -->
      <div class="flex flex-col items-center leading-none z-10 select-none">
        <span
          :class="[
            'font-serif font-bold tracking-tight text-sm sm:text-base',
            isRed ? 'text-rose-500' : 'text-zinc-100',
          ]"
        >
          {{ rankLabel }}
        </span>
        <span
          :class="[
            'text-xs sm:text-sm mt-0.5',
            isRed ? 'text-rose-500' : 'text-zinc-300',
          ]"
        >
          {{ suitSymbol }}
        </span>
      </div>

      <!-- Center Heraldic Gravure / Royal Emblem -->
      <div class="absolute inset-0 flex items-center justify-center pointer-events-none select-none">
        <!-- Subtle Radial Ambient Foil -->
        <div
          :class="[
            'w-20 h-20 rounded-full blur-2xl transition-opacity duration-300',
            isRed ? 'bg-rose-500/[0.08]' : 'bg-white/[0.04]',
          ]"
        ></div>

        <!-- Central Heraldic Emblem Layout -->
        <div class="relative flex flex-col items-center justify-center">
          <!-- Fine Geometric Ring -->
          <div
            :class="[
              'w-16 h-16 sm:w-20 sm:h-20 rounded-full border border-dashed flex items-center justify-center opacity-30 transition-transform duration-300',
              isTrump ? 'border-amber-400/40' : 'border-white/20',
              isHovered ? 'scale-110 rotate-45' : 'scale-100',
            ]"
          ></div>

          <!-- Main Suit Glyph -->
          <div
            :class="[
              'absolute font-serif select-none drop-shadow-lg transition-transform duration-200',
              size === 'hero' ? 'text-6xl sm:text-7xl' : size === 'lg' ? 'text-4xl' : 'text-3xl',
              isRed ? 'text-rose-500' : 'text-zinc-200',
              isHovered ? 'scale-105' : 'scale-100',
            ]"
          >
            {{ suitSymbol }}
          </div>

          <!-- Luxury Roman Numerals or Monogram for Court Cards -->
          <span
            v-if="card.rank >= 11"
            class="absolute -bottom-5 text-[8px] tracking-[0.3em] font-mono uppercase text-zinc-500"
          >
            {{ card.rank === 14 ? 'ACE' : card.rank === 13 ? 'KING' : card.rank === 12 ? 'QUEEN' : 'JACK' }}
          </span>
        </div>
      </div>

      <!-- Bottom-Right Corner Inverted Index -->
      <div class="flex flex-col items-center leading-none self-end rotate-180 z-10 select-none">
        <span
          :class="[
            'font-serif font-bold tracking-tight text-sm sm:text-base',
            isRed ? 'text-rose-500' : 'text-zinc-100',
          ]"
        >
          {{ rankLabel }}
        </span>
        <span
          :class="[
            'text-xs sm:text-sm mt-0.5',
            isRed ? 'text-rose-500' : 'text-zinc-300',
          ]"
        >
          {{ suitSymbol }}
        </span>
      </div>

      <!-- Dynamic Specular Foil Reflection -->
      <div
        v-if="tiltEnabled && isHovered"
        class="absolute inset-0 pointer-events-none rounded-[inherit] mix-blend-color-dodge transition-opacity duration-150"
        :style="{
          background: `radial-gradient(circle 140px at ${glareX}% ${glareY}%, rgba(255, 255, 255, 0.18), transparent 70%)`,
        }"
      ></div>
    </div>

    <!-- BACK OF CARD (THEORY11 BESPOKE CASINO PATTERN) -->
    <div
      v-else
      :class="[
        'w-full h-full relative overflow-hidden rounded-[inherit] p-2 flex items-center justify-center',
        'bg-[#08090c] border border-amber-400/30 shadow-[0_12px_36px_rgba(0,0,0,0.9)]',
      ]"
    >
      <!-- Double Inner Gold Border -->
      <div class="absolute inset-1.5 rounded-[inherit] border border-amber-400/20 bg-[#090b0e] overflow-hidden flex items-center justify-center">
        <!-- Guilloché Lattice Mesh -->
        <div class="absolute inset-0 opacity-15 bg-[radial-gradient(#fbbf24_1px,transparent_1px)] [background-size:10px_10px]"></div>

        <!-- Central Diamond Emblem -->
        <div class="relative w-10 h-10 sm:w-12 sm:h-12 border border-amber-400/40 rotate-45 flex items-center justify-center bg-zinc-950/80">
          <div class="w-7 h-7 border border-amber-400/20 -rotate-45 flex items-center justify-center">
            <span class="font-mono font-bold text-[10px] text-amber-400 tracking-wider">BITO</span>
          </div>
        </div>
      </div>

      <!-- Glare on Back -->
      <div
        v-if="tiltEnabled && isHovered"
        class="absolute inset-0 pointer-events-none rounded-[inherit] mix-blend-overlay transition-opacity duration-150"
        :style="{
          background: `radial-gradient(circle 120px at ${glareX}% ${glareY}%, rgba(251, 191, 36, 0.25), transparent 70%)`,
        }"
      ></div>
    </div>
  </div>
</template>
