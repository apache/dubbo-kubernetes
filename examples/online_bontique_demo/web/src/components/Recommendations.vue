<template>
  <section v-if="recommendations && recommendations.length > 0" class="recommendations-section">
    <div class="container">
      <div class="recommendations-header">
        <h3>Curator's Suggestions</h3>
      </div>
      <div class="products-grid">
        <div v-for="prod in recommendations" :key="prod.id" class="product-card">
          <router-link :to="`/product/${prod.id}`" class="product-card-img-wrap">
            <img :src="prod.picture" :alt="prod.name" loading="lazy" />
          </router-link>
          <div class="product-card-body">
            <span class="product-category-tag" v-if="prod.categories && prod.categories.length">
              {{ prod.categories[0] }}
            </span>
            <router-link :to="`/product/${prod.id}`" class="product-card-title">
              {{ prod.name }}
            </router-link>
            <div class="product-card-footer">
              <span class="product-card-price">{{ formatPrice(prod.priceUsd) }}</span>
              <router-link :to="`/product/${prod.id}`" class="product-view-link">
                View &rarr;
              </router-link>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { useCurrency } from '../composables/useCurrency.js';

const { formatPrice } = useCurrency();

defineProps({
  recommendations: {
    type: Array,
    default: () => []
  }
});
</script>
