<template>
  <div class="catalog-section">
    <div class="container">
      <div class="section-title-wrap" style="margin-bottom: 36px;">
        <h2>Curator's Seasonal Offers</h2>
        <p>Exclusive promotional partnerships and limited seasonal deals</p>
      </div>

      <div v-if="ads.length > 0" class="products-grid">
        <div v-for="(ad, index) in ads" :key="index" class="product-card" style="padding: 24px;">
          <span class="ad-pill">Special Partnership</span>
          <h3 class="product-card-title" style="margin-top: 12px; margin-bottom: 16px;">
            {{ ad.text }}
          </h3>
          <div style="margin-top: auto; padding-top: 16px;">
            <router-link :to="ad.redirectUrl" class="btn-primary" style="width: 100%;">
              Explore Offer &rarr;
            </router-link>
          </div>
        </div>
      </div>

      <div v-else style="text-align: center; padding: 60px; color: var(--text-muted);">
        No special promotions active at this time.
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';

const ads = ref([]);

onMounted(async () => {
  try {
    const res = await fetch('/api/ads', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ contextKeys: [] })
    });
    if (res.ok) {
      const data = await res.json();
      ads.value = data.ads || [];
    }
  } catch (err) {
    console.error('Failed to load promotional offers', err);
  }
});
</script>
