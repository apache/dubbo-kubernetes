<template>
  <main class="product-detail-page" v-if="product">
    <div class="container">
      <!-- Breadcrumbs -->
      <nav class="breadcrumbs">
        <router-link to="/">Collection</router-link>
        <span>&rsaquo;</span>
        <span v-if="product.categories && product.categories.length">{{ product.categories[0] }}</span>
        <span>&rsaquo;</span>
        <span>{{ product.name }}</span>
      </nav>

      <!-- Toast Feedback -->
      <div v-if="showToast" class="cart-toast" role="status">
        <span><strong>Added to bag!</strong> &ldquo;{{ product.name }}&rdquo; (&times;{{ quantity }}) is now in your shopping bag.</span>
        <div>
          <router-link to="/cart">View Bag &rarr;</router-link>
        </div>
      </div>

      <div class="product-layout">
        <!-- Gallery Frame -->
        <div class="product-gallery-frame">
          <img :src="product.picture" :alt="product.name" />
        </div>

        <!-- Details Panel -->
        <div class="product-info-panel">
          <div class="product-meta-header">
            <span class="product-category-tag" v-if="product.categories && product.categories.length">
              {{ product.categories[0] }}
            </span>
            <div class="product-sku">SKU #{{ product.id }}</div>
          </div>

          <h1 class="product-detail-title">{{ product.name }}</h1>
          <div class="product-detail-price">{{ formatPrice(product.priceUsd) }}</div>

          <p class="product-detail-description">
            {{ product.description }}
          </p>

          <div class="purchase-actions-box">
            <div class="quantity-control-wrap">
              <span class="quantity-label">Quantity</span>
              <div class="quantity-stepper">
                <button
                  type="button"
                  class="stepper-btn"
                  @click="decrementQty"
                  :disabled="quantity <= 1"
                  aria-label="Decrease quantity"
                >
                  &minus;
                </button>
                <span class="stepper-val">{{ quantity }}</span>
                <button
                  type="button"
                  class="stepper-btn"
                  @click="incrementQty"
                  :disabled="quantity >= 10"
                  aria-label="Increase quantity"
                >
                  &plus;
                </button>
              </div>
            </div>

            <button
              type="button"
              class="btn-primary"
              style="width: 100%;"
              @click="handleAddToCart"
              :disabled="adding"
            >
              {{ adding ? 'Adding to Bag...' : 'Add to Shopping Bag' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Editorial Ad Banner -->
      <div v-if="ad" class="ad-editorial-card">
        <div>
          <span class="ad-pill">Curator's Special</span>
          <div class="ad-text">{{ ad.text }}</div>
        </div>
        <router-link :to="ad.redirectUrl" class="btn-secondary">
          View Promotion &rarr;
        </router-link>
      </div>

      <!-- Recommendations -->
      <Recommendations :recommendations="recommendations" />
    </div>
  </main>

  <div v-else-if="loadingProduct" class="container" style="padding: 100px 0; text-align: center; color: var(--text-muted);">
    Loading item details...
  </div>
  <div v-else class="container" style="padding: 100px 0; text-align: center;">
    <h2>Item Not Found</h2>
    <router-link to="/" class="btn-primary" style="margin-top: 20px;">Return to Collection</router-link>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import Recommendations from '../components/Recommendations.vue';
import { useCart } from '../composables/useCart.js';
import { useCurrency } from '../composables/useCurrency.js';

const route = useRoute();
const { addToCart } = useCart();
const { formatPrice } = useCurrency();

const product = ref(null);
const quantity = ref(1);
const recommendations = ref([]);
const ad = ref(null);
const loadingProduct = ref(true);
const adding = ref(false);
const showToast = ref(false);

function incrementQty() {
  if (quantity.value < 10) quantity.value += 1;
}

function decrementQty() {
  if (quantity.value > 1) quantity.value -= 1;
}

async function handleAddToCart() {
  if (!product.value) return;
  adding.value = true;
  try {
    await addToCart(product.value.id, quantity.value);
    showToast.value = true;
    setTimeout(() => {
      showToast.value = false;
    }, 4500);
  } catch (err) {
    console.error('Failed to add to cart:', err);
    alert('Failed to add to shopping bag: ' + (err.message || 'Network error'));
  } finally {
    adding.value = false;
  }
}

async function loadData(id) {
  loadingProduct.value = true;
  showToast.value = false;
  try {
    const res = await fetch(`/api/products/${id}`);
    if (res.ok) {
      product.value = await res.json();
    } else {
      product.value = null;
      return;
    }

    // Parallel fetch for recommendations and ad
    const recPromise = fetch('/api/recommendations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: '1', productIds: [id] })
    }).then(async (recRes) => {
      if (recRes.ok) {
        const recData = await recRes.json();
        const pids = recData.productIds || [];
        const prods = await Promise.all(
          pids.map((pid) => fetch(`/api/products/${pid}`).then((r) => r.ok ? r.json() : null))
        );
        recommendations.value = prods.filter(Boolean);
      }
    });

    // Fix: pass product.categories to ad contextKeys so relevant category ads are matched
    const contextKeys = product.value.categories || [];
    const adPromise = fetch('/api/ads', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ contextKeys })
    }).then(async (adRes) => {
      if (adRes.ok) {
        const adData = await adRes.json();
        if (adData.ads && adData.ads.length > 0) {
          ad.value = adData.ads[0];
        }
      }
    });

    await Promise.all([recPromise, adPromise]);
  } catch (err) {
    console.error('Failed to load product details', err);
  } finally {
    loadingProduct.value = false;
  }
}

onMounted(() => {
  loadData(route.params.id);
});

watch(() => route.params.id, (newId) => {
  if (newId) {
    quantity.value = 1;
    loadData(newId);
  }
});
</script>
