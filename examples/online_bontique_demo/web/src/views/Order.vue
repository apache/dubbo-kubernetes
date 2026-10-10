<template>
  <div class="order-page-section">
    <div class="container">
      <!-- Order Found & Valid -->
      <section v-if="order" class="order-complete-card">
        <div class="order-hero-badge">&check;</div>

        <div class="order-header-text">
          <h2>Order Confirmed</h2>
          <p>Thank you for supporting our artisans. Your order has been placed and is being prepared.</p>
        </div>

        <div class="order-info-grid">
          <div class="order-info-box">
            <div class="info-box-label">Order Confirmation #</div>
            <div class="info-box-val">{{ order.orderId }}</div>
          </div>

          <div class="order-info-box">
            <div class="info-box-label">Shipping Tracking #</div>
            <div class="info-box-val">{{ order.shippingTrackingId }}</div>
          </div>
        </div>

        <!-- Shipping Destination -->
        <div v-if="order.shippingAddress" class="order-info-box" style="margin-bottom: 28px;">
          <div class="info-box-label">Delivery Destination</div>
          <div style="font-size: 14px; color: var(--text-primary); margin-top: 4px;">
            {{ order.shippingAddress.streetAddress }}, {{ order.shippingAddress.city }},
            {{ order.shippingAddress.state }} {{ order.shippingAddress.zipCode }}, {{ order.shippingAddress.country }}
          </div>
        </div>

        <!-- Itemized Receipt -->
        <table class="receipt-table" v-if="order.items && order.items.length">
          <thead>
            <tr>
              <th>Item / SKU</th>
              <th style="text-align: center;">Qty</th>
              <th style="text-align: right;">Amount</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, idx) in order.items" :key="idx">
              <td>
                <strong>{{ getItemName(it.item.productId) }}</strong>
                <div style="font-size: 11px; color: var(--text-muted);">SKU #{{ it.item.productId }}</div>
              </td>
              <td style="text-align: center;">&times; {{ it.item.quantity }}</td>
              <td style="text-align: right;">
                <strong>{{ formatMoney(it.cost) }}</strong>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Summary Breakdown -->
        <div class="cart-summary-totals">
          <div class="summary-row" v-if="order.shippingCost">
            <span>Shipping &amp; Handling</span>
            <span>{{ formatMoney(order.shippingCost) }}</span>
          </div>
          <div class="summary-row grand-total">
            <span>Total Paid</span>
            <span>{{ computedTotal }}</span>
          </div>
        </div>

        <div style="display: flex; gap: 16px; justify-content: center; margin-top: 36px;">
          <router-link to="/" class="btn-primary">
            Continue Shopping &rarr;
          </router-link>
          <button type="button" class="btn-secondary" @click="printReceipt">
            Print Receipt
          </button>
        </div>
      </section>

      <!-- Empty Order State -->
      <section v-else class="cart-empty-state">
        <div class="empty-icon-wrap">
          <img src="/icons/Hipster_CheckOutIcon.svg" alt="" />
        </div>
        <h3>No Recent Order Found</h3>
        <p>No active order receipt was found in this session. You can view our collection and place a new order at any time.</p>
        <router-link to="/" class="btn-primary">
          Explore Collection &rarr;
        </router-link>
      </section>

      <!-- Recommendations -->
      <Recommendations :recommendations="recommendations" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import Recommendations from '../components/Recommendations.vue';

const order = ref(null);
const userCurrency = ref('USD');
const recommendations = ref([]);
const productNames = ref({});

function getItemName(productId) {
  return productNames.value[productId] || `Artisan Piece (${productId})`;
}

const SYMBOL_MAP = {
  USD: '$',
  EUR: '€',
  GBP: '£',
  JPY: '¥',
  CNY: '¥',
  CAD: 'CA$',
  AUD: 'AU$'
};

function formatMoney(m) {
  if (!m) return '$ 0.00';
  const units = m.units || 0;
  const nanos = m.nanos || 0;
  const code = m.currencyCode || userCurrency.value || 'USD';
  const symbol = SYMBOL_MAP[code] || code + ' ';
  const val = units + (nanos / 1000000000.0);
  if (code === 'JPY' || code === 'KRW') {
    return `${symbol} ${Math.round(val).toLocaleString()}`;
  }
  return `${symbol} ${val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

const computedTotal = computed(() => {
  if (!order.value) return '$ 0.00';
  let totalUnits = 0;
  let totalNanos = 0;
  let currencyCode = userCurrency.value;

  if (order.value.shippingCost) {
    totalUnits += order.value.shippingCost.units || 0;
    totalNanos += order.value.shippingCost.nanos || 0;
    currencyCode = order.value.shippingCost.currencyCode || currencyCode;
  }

  for (const it of order.value.items || []) {
    if (it.cost) {
      totalUnits += it.cost.units || 0;
      totalNanos += it.cost.nanos || 0;
      currencyCode = it.cost.currencyCode || currencyCode;
    }
  }

  return formatMoney({ units: totalUnits, nanos: totalNanos, currencyCode });
});

function printReceipt() {
  window.print();
}

onMounted(async () => {
  const cached = sessionStorage.getItem('last_order');
  if (cached) {
    try {
      const parsed = JSON.parse(cached);
      order.value = parsed.order;
      userCurrency.value = parsed.userCurrency || 'USD';

      // Load item details for product names
      if (order.value && order.value.items) {
        for (const it of order.value.items) {
          const pid = it.item.productId;
          try {
            const pRes = await fetch(`/api/products/${pid}`);
            if (pRes.ok) {
              const pData = await pRes.json();
              productNames.value[pid] = pData.name;
            }
          } catch {
            // ignore
          }
        }
      }
    } catch (e) {
      console.error('Failed to parse cached order', e);
    }
  }

  // Recommendations
  try {
    const recRes = await fetch('/api/recommendations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: '1', productIds: [] })
    });
    if (recRes.ok) {
      const recData = await recRes.json();
      const pids = recData.productIds || [];
      const prods = await Promise.all(
        pids.map((pid) => fetch(`/api/products/${pid}`).then((r) => r.ok ? r.json() : null))
      );
      recommendations.value = prods.filter(Boolean);
    }
  } catch (err) {
    console.warn('Failed to load recommendations', err);
  }
});
</script>
