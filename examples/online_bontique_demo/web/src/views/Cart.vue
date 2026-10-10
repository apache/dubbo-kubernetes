<template>
  <div class="cart-page-section">
    <div class="container">
      <!-- Empty Cart State -->
      <div v-if="cartItems.length === 0" class="cart-empty-state">
        <div class="empty-icon-wrap">
          <img src="/icons/Hipster_CartIcon.svg" alt="" />
        </div>
        <h3>Your Shopping Bag is Empty</h3>
        <p>Your bag is currently empty. Explore our latest arrivals to find pieces for your daily ritual.</p>
        <router-link to="/" class="btn-primary">
          Explore Collection &rarr;
        </router-link>
      </div>

      <!-- Non-Empty Cart Content -->
      <div v-else class="cart-content-grid">
        <!-- Items & Breakdown Column -->
        <div class="cart-items-card">
          <div class="cart-items-header">
            <h3>Items in Bag ({{ cartCount }})</h3>
            <button type="button" class="btn-ghost" @click="handleEmptyCart" :disabled="cartLoading">
              Empty Bag
            </button>
          </div>

          <div class="cart-items-list">
            <div v-for="item in cartItems" :key="item.product.id" class="cart-item-row">
              <div class="cart-item-thumb">
                <router-link :to="`/product/${item.product.id}`">
                  <img :src="item.product.picture" :alt="item.product.name" />
                </router-link>
              </div>

              <div class="cart-item-details">
                <router-link :to="`/product/${item.product.id}`">
                  <h4>{{ item.product.name }}</h4>
                </router-link>
                <div class="cart-item-sku">SKU #{{ item.product.id }}</div>
                <span class="cart-item-qty-tag">Quantity: {{ item.quantity }}</span>
              </div>

              <div class="cart-item-price-col">
                <div class="cart-item-total">
                  {{ formatItemTotal(item) }}
                </div>
              </div>
            </div>
          </div>

          <!-- Totals Breakdown -->
          <div class="cart-summary-totals">
            <div class="summary-row">
              <span>Subtotal</span>
              <span>{{ formatPrice(subtotalUsd) }}</span>
            </div>
            <div class="summary-row">
              <span>Estimated Shipping</span>
              <span>{{ shippingQuoteLoading ? 'Calculating...' : formatPrice(shippingCostUsd) }}</span>
            </div>
            <div class="summary-row grand-total">
              <span>Estimated Total ({{ currentCurrency }})</span>
              <span>{{ formatPrice(grandTotalUsd) }}</span>
            </div>
          </div>
        </div>

        <!-- Checkout Form Column -->
        <div class="checkout-card">
          <form @submit.prevent="submitCheckout">
            <!-- Step 1: Address -->
            <div class="checkout-section-title">
              <span class="step-num">1</span>
              <span>Shipping Address</span>
            </div>

            <div class="form-grid">
              <div class="form-group form-col-full">
                <label for="email" class="form-label">Email Address</label>
                <input type="email" id="email" class="form-input" v-model="form.email" required />
              </div>

              <div class="form-group form-col-full">
                <label for="street" class="form-label">Street Address</label>
                <input type="text" id="street" class="form-input" v-model="form.streetAddress" @change="fetchShippingQuote" required />
              </div>

              <div class="form-group">
                <label for="city" class="form-label">City</label>
                <input type="text" id="city" class="form-input" v-model="form.city" @change="fetchShippingQuote" required />
              </div>

              <div class="form-group">
                <label for="state" class="form-label">State / Province</label>
                <input type="text" id="state" class="form-input" v-model="form.state" @change="fetchShippingQuote" required />
              </div>

              <div class="form-group">
                <label for="zip" class="form-label">Zip / Postal Code</label>
                <input type="text" id="zip" class="form-input" v-model="form.zipCode" @change="fetchShippingQuote" required />
              </div>

              <div class="form-group">
                <label for="country" class="form-label">Country</label>
                <input type="text" id="country" class="form-input" v-model="form.country" @change="fetchShippingQuote" required />
              </div>
            </div>

            <!-- Step 2: Payment -->
            <div class="checkout-section-title">
              <span class="step-num">2</span>
              <span>Payment Details</span>
            </div>

            <div class="form-grid">
              <div class="form-group form-col-full">
                <label for="card" class="form-label">Card Number</label>
                <input type="text" id="card" class="form-input" v-model="form.creditCardNumber" placeholder="4432-8015-6152-0454" required />
              </div>

              <div class="form-group">
                <label for="month" class="form-label">Expiry Month</label>
                <select id="month" class="form-select" v-model.number="form.creditCardExpirationMonth">
                  <option v-for="m in 12" :key="m" :value="m">{{ String(m).padStart(2, '0') }}</option>
                </select>
              </div>

              <div class="form-group">
                <label for="year" class="form-label">Expiry Year</label>
                <select id="year" class="form-select" v-model.number="form.creditCardExpirationYear">
                  <option :value="2026">2026</option>
                  <option :value="2027">2027</option>
                  <option :value="2028">2028</option>
                  <option :value="2029">2029</option>
                </select>
              </div>

              <div class="form-group form-col-full">
                <label for="cvv" class="form-label">Security Code (CVV)</label>
                <input type="text" id="cvv" class="form-input" v-model="form.creditCardCvv" maxlength="4" placeholder="123" required />
              </div>
            </div>

            <button type="submit" class="btn-primary" style="width: 100%; padding: 14px;" :disabled="loading || cartLoading">
              {{ loading ? 'Securing Order...' : `Place Order &bull; ${formatPrice(grandTotalUsd)}` }}
            </button>
          </form>
        </div>
      </div>

      <!-- Recommendations -->
      <Recommendations :recommendations="recommendations" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import Recommendations from '../components/Recommendations.vue';
import { useCart } from '../composables/useCart.js';
import { useCurrency } from '../composables/useCurrency.js';

const router = useRouter();
const { cartItems, cartCount, cartLoading, refreshCart, emptyCart } = useCart();
const { currentCurrency, formatPrice } = useCurrency();

const recommendations = ref([]);
const loading = ref(false);
const shippingCostUsd = ref(8.99);
const shippingQuoteLoading = ref(false);

const form = ref({
  email: 'someone@example.com',
  streetAddress: '1600 Amphitheatre Parkway',
  zipCode: '94043',
  city: 'Mountain View',
  state: 'CA',
  country: 'United States',
  creditCardNumber: '4432-8015-6152-0454',
  creditCardExpirationMonth: 12,
  creditCardExpirationYear: 2026,
  creditCardCvv: '123'
});

const subtotalUsd = computed(() => {
  let sum = 0;
  for (const it of cartItems.value) {
    if (it.product && it.product.priceUsd) {
      const price = (it.product.priceUsd.units || 0) + ((it.product.priceUsd.nanos || 0) / 1000000000.0);
      sum += price * (it.quantity || 1);
    }
  }
  return sum;
});

const grandTotalUsd = computed(() => {
  return subtotalUsd.value + shippingCostUsd.value;
});

function formatItemTotal(item) {
  if (!item.product || !item.product.priceUsd) return formatPrice(0);
  const unitUsd = (item.product.priceUsd.units || 0) + ((item.product.priceUsd.nanos || 0) / 1000000000.0);
  return formatPrice(unitUsd * (item.quantity || 1));
}

async function handleEmptyCart() {
  await emptyCart('1');
}

async function fetchShippingQuote() {
  if (cartItems.value.length === 0) return;
  shippingQuoteLoading.value = true;
  try {
    const quotePayload = {
      address: {
        streetAddress: form.value.streetAddress,
        city: form.value.city,
        state: form.value.state,
        country: form.value.country,
        zipCode: parseInt(form.value.zipCode, 10) || 94043
      },
      items: cartItems.value.map((it) => ({
        productId: it.product.id,
        quantity: it.quantity
      }))
    };
    const res = await fetch('/api/shipping/quote', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(quotePayload)
    });
    if (res.ok) {
      const data = await res.json();
      if (data.costUsd) {
        shippingCostUsd.value = (data.costUsd.units || 0) + ((data.costUsd.nanos || 0) / 1000000000.0);
      }
    }
  } catch (err) {
    console.warn('Could not fetch dynamic shipping quote, using default:', err);
  } finally {
    shippingQuoteLoading.value = false;
  }
}

async function loadRecommendations() {
  try {
    const productIds = cartItems.value.map(it => it.product.id);
    const recRes = await fetch('/api/recommendations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: '1', productIds })
    });
    if (recRes.ok) {
      const recData = await recRes.json();
      const pids = recData.productIds || [];
      const prods = await Promise.all(
        pids.map(pid => fetch(`/api/products/${pid}`).then(r => r.ok ? r.json() : null))
      );
      recommendations.value = prods.filter(Boolean);
    }
  } catch (err) {
    console.warn('Recommendations fetch failed:', err);
  }
}

async function submitCheckout() {
  loading.value = true;
  try {
    const payload = {
      userId: '1',
      userCurrency: currentCurrency.value,
      address: {
        streetAddress: form.value.streetAddress,
        city: form.value.city,
        state: form.value.state,
        country: form.value.country,
        zipCode: parseInt(form.value.zipCode, 10) || 94043
      },
      email: form.value.email,
      creditCard: {
        creditCardNumber: form.value.creditCardNumber,
        creditCardCvv: parseInt(form.value.creditCardCvv, 10) || 123,
        creditCardExpirationYear: form.value.creditCardExpirationYear,
        creditCardExpirationMonth: form.value.creditCardExpirationMonth
      }
    };

    const res = await fetch('/api/checkout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const data = await res.json();
      sessionStorage.setItem('last_order', JSON.stringify({
        order: data.order,
        userCurrency: currentCurrency.value
      }));
      await refreshCart('1');
      router.push('/order');
    } else {
      const err = await res.json().catch(() => ({}));
      alert('Checkout failed: ' + (err.error || res.statusText));
    }
  } catch (err) {
    console.error('Checkout error:', err);
    alert('Checkout network error');
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await refreshCart('1');
  await fetchShippingQuote();
  await loadRecommendations();
});
</script>
