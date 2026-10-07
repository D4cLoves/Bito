<script setup lang="ts">
import { ref } from 'vue'
import { LANDING_SECTIONS, requestJoin, scrollToSection } from '../../composables/useLanding'
import { useReveal } from '../../composables/useReveal'

const rootEl = ref<HTMLElement | null>(null)
useReveal(rootEl)
</script>

<template>
  <footer ref="rootEl" class="end">
    <div class="cta" data-reveal>
      <h2 class="cta-title">Take a seat.</h2>
      <p class="cta-text">Six cards, one trump, no luck you didn’t earn.</p>
      <button type="button" class="pill-btn" @click="requestJoin">Join Game</button>
    </div>

    <div class="bar">
      <a href="#" class="brand" @click.prevent="requestJoin">
        <svg viewBox="0 0 40 38" aria-hidden="true">
          <rect x="17.5" y="4.5" width="18" height="26" rx="3.2" transform="rotate(12 26.5 17.5)" fill="none" stroke="currentColor" stroke-width="2" />
          <rect x="4" y="5.5" width="18" height="26" rx="3.2" transform="rotate(-14 13 18.5)" fill="#f5f4ef" stroke="currentColor" stroke-width="2" />
        </svg>
        Bito
      </a>
      <nav class="links">
        <a v-for="s in LANDING_SECTIONS" :key="s.id" :href="`#${s.id}`" @click.prevent="scrollToSection(s.id)">{{ s.label }}</a>
      </nav>
      <span class="copy">© {{ new Date().getFullYear() }} Bito</span>
    </div>
  </footer>
</template>

<style scoped>
.end {
  max-width: 1280px;
  margin: 0 auto;
  padding: 60px 48px 36px;
}

.cta {
  padding: 90px 32px;
  border-radius: 36px;
  text-align: center;
  background:
    radial-gradient(ellipse at 50% 0%, rgba(246, 235, 211, 0.9), rgba(246, 235, 211, 0) 60%),
    rgba(250, 249, 245, 0.6);
  box-shadow: 0 0 0 8px #d3d0c8 inset;
}

.cta-title {
  margin: 0 0 14px;
  font-size: 72px;
  font-weight: 400;
  line-height: 1;
  letter-spacing: -0.035em;
  color: #36322c;
}

.cta-text {
  margin: 0 0 32px;
  font-size: 19px;
  color: #57534b;
}

.pill-btn {
  height: 50px;
  padding: 0 34px;
  border: 0;
  border-radius: 999px;
  background: #d7d3c8;
  color: #1b1916;
  font: inherit;
  font-size: 16px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.pill-btn:hover {
  background: #cdc8bb;
}

.pill-btn:active {
  transform: scale(0.985);
}

.bar {
  display: flex;
  align-items: center;
  gap: 32px;
  padding: 36px 8px 0;
  font-size: 14.5px;
  color: #6b675e;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 20px;
  color: #1b1916;
  text-decoration: none;
}

.brand svg {
  width: 28px;
  height: 27px;
  color: #26231e;
}

.links {
  display: flex;
  gap: 24px;
  margin-left: auto;
}

.links a {
  color: inherit;
  text-decoration: none;
  transition: color 0.2s;
}

.links a:hover {
  color: #1b1916;
}

@media (max-width: 760px) {
  .end {
    padding: 40px 20px 28px;
  }

  .cta {
    padding: 64px 20px;
    border-radius: 28px;
  }

  .cta-title {
    font-size: 48px;
  }

  .bar {
    flex-wrap: wrap;
    gap: 16px 24px;
  }

  .links {
    order: 3;
    width: 100%;
    margin-left: 0;
    flex-wrap: wrap;
    gap: 12px 20px;
  }

  .copy {
    margin-left: auto;
  }
}
</style>
