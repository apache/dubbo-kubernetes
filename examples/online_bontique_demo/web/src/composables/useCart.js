import { ref } from 'vue';

const cartCount = ref(0);
const cartItems = ref([]);
const cartLoading = ref(false);

export function useCart() {
  async function refreshCart(userId = '1') {
    cartLoading.value = true;
    try {
      const res = await fetch(`/api/cart?userId=${userId}`);
      if (!res.ok) throw new Error('Failed to fetch cart');
      const data = await res.json();
      const rawItems = data.items || [];
      cartCount.value = rawItems.reduce((acc, it) => acc + (it.quantity || 0), 0);

      // Concurrent fetch for item product details
      const detailPromises = rawItems.map(async (item) => {
        try {
          const pRes = await fetch(`/api/products/${item.productId}`);
          if (pRes.ok) {
            const product = await pRes.json();
            return { product, quantity: item.quantity };
          }
        } catch {
          // ignore individual item error
        }
        return {
          product: { id: item.productId, name: 'Product ' + item.productId, priceUsd: { units: 0, nanos: 0, currencyCode: 'USD' } },
          quantity: item.quantity
        };
      });

      cartItems.value = await Promise.all(detailPromises);
    } catch (err) {
      console.error('refreshCart error:', err);
    } finally {
      cartLoading.value = false;
    }
  }

  async function addToCart(productId, quantity = 1, userId = '1') {
    cartLoading.value = true;
    try {
      const res = await fetch('/api/cart/add', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId,
          item: { productId, quantity }
        })
      });
      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `Failed to add to cart (HTTP ${res.status})`);
      }
      cartCount.value += quantity;
      await refreshCart(userId);
      return { success: true };
    } finally {
      cartLoading.value = false;
    }
  }

  async function emptyCart(userId = '1') {
    cartLoading.value = true;
    try {
      const res = await fetch('/api/cart/empty', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId })
      });
      if (!res.ok) throw new Error('Failed to empty cart');
      cartItems.value = [];
      cartCount.value = 0;
      return { success: true };
    } finally {
      cartLoading.value = false;
    }
  }

  return {
    cartCount,
    cartItems,
    cartLoading,
    refreshCart,
    addToCart,
    emptyCart
  };
}
