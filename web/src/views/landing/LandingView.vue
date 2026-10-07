<script setup lang="ts">
import HeroZenShowcase from '../../components/landing/HeroZenShowcase.vue'
import SiteHeader from '../../components/landing/SiteHeader.vue'
import HowToPlay from '../../components/landing/HowToPlay.vue'
import GameModes from '../../components/landing/GameModes.vue'
import FairPlay from '../../components/landing/FairPlay.vue'
import FaqSection from '../../components/landing/FaqSection.vue'
import SiteFooter from '../../components/landing/SiteFooter.vue'
</script>

<template>
  <div class="landing">
    <SiteHeader />
    <HeroZenShowcase />
    <main>
      <HowToPlay />
      <GameModes />
      <FairPlay />
      <FaqSection />
    </main>
    <SiteFooter />
  </div>
</template>

<style scoped>
.landing {
  min-height: 100vh;
  background: #f5f4ef;
  color: #1b1916;
  font-family: 'Inter', -apple-system, 'Helvetica Neue', Arial, sans-serif;
}
</style>
