<script setup lang="ts">
import { computed } from 'vue'

// Светлая «бумажная» карта для демо на лендинге. Лицо и рубашка лежат друг на друге,
// поворот делается снаружи (rotateY у родителя) — backface скрывает лишнюю сторону.
const props = defineProps<{
  rank?: string
  suit?: '♠' | '♣' | '♥' | '♦'
}>()

const red = computed(() => props.suit === '♥' || props.suit === '♦')
</script>

<template>
  <div class="paper-card">
    <div class="face face--front" :class="{ red }">
      <span class="index index--top">{{ rank }}<i>{{ suit }}</i></span>
      <span class="pip">{{ suit }}</span>
      <span class="index index--bottom">{{ rank }}<i>{{ suit }}</i></span>
    </div>
    <div class="face face--back">
      <span class="back-inner"><span class="back-mark">B</span></span>
    </div>
  </div>
</template>

<style scoped>
.paper-card {
  position: relative;
  width: 100%;
  height: 100%;
  transform-style: preserve-3d;
}

.face {
  position: absolute;
  inset: 0;
  box-sizing: border-box;
  border-radius: 9px;
  backface-visibility: hidden;
  -webkit-backface-visibility: hidden;
  /* тень лёгкая: в колоде карты лежат стопкой и тени складываются */
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.9) inset,
    0 8px 16px -10px rgba(70, 62, 48, 0.28),
    0 0 0 1px rgba(160, 150, 132, 0.35);
}

.face--front {
  background: linear-gradient(160deg, #fdfcf8, #f2efe7);
  color: #2b2824;
  font-family: Georgia, 'Times New Roman', serif;
}

.face--front.red {
  color: #a1473f;
}

.face--back {
  transform: rotateY(180deg);
  padding: 6px;
  background: #ebe6db;
}

.back-inner {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  border: 1px solid #cfc6b4;
  border-radius: 5px;
  background:
    repeating-linear-gradient(45deg, transparent 0 6px, rgba(160, 148, 124, 0.18) 6px 7px),
    repeating-linear-gradient(-45deg, transparent 0 6px, rgba(160, 148, 124, 0.18) 6px 7px),
    #e7e1d4;
}

.back-mark {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: 1px solid #b9ae98;
  border-radius: 50%;
  background: #efeadf;
  font-family: Georgia, serif;
  font-size: 13px;
  color: #8a7f68;
}

.index {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-size: 17px;
  line-height: 1;
}

.index i {
  margin-top: 2px;
  font-style: normal;
  font-size: 14px;
}

.index--top {
  left: 8px;
  top: 7px;
}

.index--bottom {
  right: 8px;
  bottom: 7px;
  transform: rotate(180deg);
}

.pip {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  font-size: 40px;
  line-height: 1;
}
</style>
