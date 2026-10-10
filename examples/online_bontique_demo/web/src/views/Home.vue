<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="home-hero-section">
      <div class="container">
        <div class="hero-grid">
          <div class="hero-content">
            <h1 class="hero-title">
              Curated Essentials for Considered Living
            </h1>
            <p class="hero-subtitle">
              A meticulously designed collection of homeware, accessories, and apparel.
              Each piece balances functional utility with understated Scandinavian and Japanese aesthetics.
            </p>
            <div class="hero-actions">
              <a href="#catalog" class="btn-primary">
                Explore Collection &darr;
              </a>
              <router-link to="/ad" class="btn-secondary">
                Curator's Specials
              </router-link>
            </div>
          </div>

          <div class="hero-media">
            <div class="hero-image-card">
              <img
                src="/images/folded-clothes-on-white-chair.jpg"
                alt="Curated lifestyle showcase"
              />
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Catalog Section -->
    <section id="catalog" class="catalog-section">
      <div class="container">
        <div class="section-header-row">
          <div class="section-title-wrap">
            <h2>The Catalog</h2>
          </div>

          <div class="category-filter-pills">
            <button
              v-for="cat in availableCategories"
              :key="cat.key"
              class="filter-pill"
              :class="{ active: selectedCategory === cat.key }"
              @click="selectedCategory = cat.key"
            >
              {{ cat.label }}
            </button>
          </div>
        </div>

        <div v-if="filteredProducts.length > 0" class="products-grid">
          <div v-for="product in filteredProducts" :key="product.id" class="product-card">
            <router-link :to="`/product/${product.id}`" class="product-card-img-wrap">
              <img :src="product.picture" :alt="product.name" loading="lazy" />
            </router-link>
            <div class="product-card-body">
              <span class="product-category-tag" v-if="product.categories && product.categories.length">
                {{ product.categories[0] }}
              </span>
              <router-link :to="`/product/${product.id}`" class="product-card-title">
                {{ product.name }}
              </router-link>
              <div class="product-card-footer">
                <span class="product-card-price">{{ formatPrice(product.priceUsd) }}</span>
                <router-link :to="`/product/${product.id}`" class="product-view-link">
                  View Piece &rarr;
                </router-link>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="loading" style="text-align: center; padding: 60px; color: var(--text-muted);">
          Loading collection...
        </div>
        <div v-else style="text-align: center; padding: 60px; color: var(--text-muted);">
          No items found for this category.
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useCurrency } from '../composables/useCurrency.js';

const { formatPrice } = useCurrency();

const products = ref([]);
const loading = ref(true);
const selectedCategory = ref('all');

const availableCategories = [
  { key: 'all', label: 'All Pieces' },
  { key: 'kitchen', label: 'Kitchen & Dining' },
  { key: 'accessories', label: 'Accessories' },
  { key: 'clothing', label: 'Apparel' },
  { key: 'decor', label: 'Decor' }
];

const filteredProducts = computed(() => {
  if (selectedCategory.value === 'all') {
    return products.value;
  }
  return products.value.filter(p => {
    return (p.categories || []).includes(selectedCategory.value);
  });
});

onMounted(async () => {
  try {
    const res = await fetch('/api/products');
    if (res.ok) {
      const data = await res.json();
      products.value = data.products || [];
    }
  } catch (err) {
    console.error('Failed to load products', err);
  } finally {
    loading.value = false;
  }
});
</script>
