<template>
  <header class="site-header">
    <div class="top-announcement">
      <span>Complimentary shipping on orders over $150</span>
    </div>

    <div class="container">
      <div class="navbar-inner">
        <router-link to="/" class="brand-link" title="Online Boutique Home">
          <img src="/icons/Hipster_NavLogo.svg" class="brand-logo-img" alt="Online Boutique Logo" />
        </router-link>

        <div class="header-controls">
          <!-- Currency Selector -->
          <div class="currency-selector-wrapper" ref="dropdownRef">
            <button
              type="button"
              class="currency-trigger"
              @click="toggleCurrencyDropdown"
              :aria-expanded="isCurrencyOpen"
              title="Change Currency"
            >
              <img src="/icons/Hipster_CurrencyIcon.svg" class="currency-icon" alt="" />
              <span>{{ currentCurrency }}</span>
              <img
                src="/icons/Hipster_DownArrow.svg"
                class="currency-chevron"
                :class="{ open: isCurrencyOpen }"
                alt=""
              />
            </button>

            <div v-if="isCurrencyOpen" class="currency-dropdown">
              <div
                v-for="code in supportedCurrencies"
                :key="code"
                class="currency-option"
                :class="{ active: code === currentCurrency }"
                @click="selectCurrency(code)"
              >
                <span>{{ code }}</span>
                <span class="option-symbol">{{ getSymbol(code) }}</span>
              </div>
            </div>
          </div>

          <!-- Cart Link -->
          <router-link to="/cart" class="header-cart-btn" title="View Shopping Cart">
            <img src="/icons/Hipster_CartIcon.svg" class="cart-icon-img" alt="Cart" />
            <span>Bag</span>
            <span v-if="cartCount > 0" class="cart-count-badge">{{ cartCount }}</span>
          </router-link>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import { useCart } from '../composables/useCart.js';
import { useCurrency } from '../composables/useCurrency.js';

const { cartCount, refreshCart } = useCart();
const { currentCurrency, supportedCurrencies, setCurrency, initCurrency } = useCurrency();

const isCurrencyOpen = ref(false);
const dropdownRef = ref(null);

const SYMBOL_MAP = {
  USD: '$',
  EUR: '€',
  GBP: '£',
  JPY: '¥',
  CNY: '¥',
  CAD: 'CA$',
  AUD: 'AU$',
  CHF: 'CHF'
};

function getSymbol(code) {
  return SYMBOL_MAP[code] || '';
}

function toggleCurrencyDropdown() {
  isCurrencyOpen.value = !isCurrencyOpen.value;
}

async function selectCurrency(code) {
  await setCurrency(code);
  isCurrencyOpen.value = false;
}

function handleClickOutside(event) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target)) {
    isCurrencyOpen.value = false;
  }
}

onMounted(() => {
  initCurrency();
  refreshCart();
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});
</script>
