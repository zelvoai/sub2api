<template>
  <router-link
    v-if="enabled"
    to="/chatgpt-images-2"
    class="images2-promo-banner"
  >
    <div class="images2-promo-banner__copy">
      <span class="images2-promo-banner__badge">NEW</span>
      <div>
        <h3 class="images2-promo-banner__title">{{ title }}</h3>
        <p class="images2-promo-banner__text">{{ text }}</p>
      </div>
    </div>
    <span class="images2-promo-banner__cta">{{ cta }}</span>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()
const settings = computed(() => appStore.cachedPublicSettings)

const enabled = computed(() => settings.value?.images2_promo_banner_enabled === true)
const title = computed(() => settings.value?.images2_promo_banner_title || 'ChatGPT Images 2 一键生图上新')
const text = computed(() => settings.value?.images2_promo_banner_text || '点击立即体验更丝滑的高质量生图。')
const cta = computed(() => settings.value?.images2_promo_banner_cta || '点击体验')
</script>

<style>
.images2-promo-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 22px;
  border: 1px solid rgba(59, 130, 246, 0.18);
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(14, 165, 233, 0.12));
  padding: 1rem 1.15rem;
  text-decoration: none;
  color: inherit;
  box-shadow: 0 18px 40px rgba(37, 99, 235, 0.08);
  transition: transform 180ms ease, box-shadow 180ms ease, border-color 180ms ease;
}

.images2-promo-banner:hover {
  transform: translateY(-1px);
  border-color: rgba(37, 99, 235, 0.28);
  box-shadow: 0 22px 48px rgba(37, 99, 235, 0.12);
}

.images2-promo-banner__copy {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  min-width: 0;
}

.images2-promo-banner__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: #ef4444;
  color: white;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  padding: 0.25rem 0.55rem;
}

.images2-promo-banner__title {
  margin: 0;
  font-size: 1rem;
  font-weight: 700;
  color: #0f172a;
}

.images2-promo-banner__text {
  margin: 0.2rem 0 0;
  color: #475569;
  font-size: 0.92rem;
}

.images2-promo-banner__cta {
  white-space: nowrap;
  border-radius: 9999px;
  background: #0f172a;
  color: #f8fafc;
  padding: 0.7rem 0.95rem;
  font-size: 0.88rem;
  font-weight: 600;
}

.dark .images2-promo-banner {
  border-color: rgba(96, 165, 250, 0.2);
  background: linear-gradient(135deg, rgba(30, 41, 59, 0.92), rgba(15, 23, 42, 0.9));
  box-shadow: 0 20px 48px rgba(2, 6, 23, 0.22);
}

.dark .images2-promo-banner__title {
  color: #f8fafc;
}

.dark .images2-promo-banner__text {
  color: #94a3b8;
}

.dark .images2-promo-banner__cta {
  background: #f8fafc;
  color: #020617;
}

@media (max-width: 768px) {
  .images2-promo-banner {
    flex-direction: column;
    align-items: stretch;
  }

  .images2-promo-banner__copy {
    align-items: flex-start;
  }

  .images2-promo-banner__cta {
    text-align: center;
  }
}
</style>
