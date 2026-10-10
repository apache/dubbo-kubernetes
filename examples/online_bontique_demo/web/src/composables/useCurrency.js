import { ref, computed } from 'vue';

const currentCurrency = ref(localStorage.getItem('ob_currency') || 'USD');
const supportedCurrencies = ref(['USD', 'EUR', 'JPY', 'GBP', 'CNY']);
const rates = ref({ USD: 1.0 });
const loadingRate = ref(false);

const CURRENCY_SYMBOLS = {
  USD: '$',
  EUR: '€',
  GBP: '£',
  JPY: '¥',
  CNY: '¥',
  CAD: 'CA$',
  AUD: 'AU$',
  CHF: 'CHF',
  HKD: 'HK$',
  SGD: 'SG$'
};

export function useCurrency() {
  const activeSymbol = computed(() => CURRENCY_SYMBOLS[currentCurrency.value] || currentCurrency.value + ' ');

  async function initCurrency() {
    try {
      const res = await fetch('/api/currencies');
      if (res.ok) {
        const data = await res.json();
        if (Array.isArray(data.currencyCodes) && data.currencyCodes.length > 0) {
          supportedCurrencies.value = data.currencyCodes;
        }
      }
    } catch (err) {
      console.warn('Could not load currencies list, using defaults:', err);
    }

    if (currentCurrency.value !== 'USD') {
      await fetchRate(currentCurrency.value);
    }
  }

  async function fetchRate(code) {
    if (code === 'USD') {
      rates.value[code] = 1.0;
      return 1.0;
    }
    if (rates.value[code] !== undefined) {
      return rates.value[code];
    }
    loadingRate.value = true;
    try {
      const res = await fetch('/api/currencies/convert', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          from: { currencyCode: 'USD', units: 1, nanos: 0 },
          toCode: code
        })
      });
      if (res.ok) {
        const data = await res.json();
        const rate = (data.units || 0) + ((data.nanos || 0) / 1000000000.0);
        rates.value[code] = rate > 0 ? rate : 1.0;
        return rates.value[code];
      }
    } catch (err) {
      console.warn(`Failed to convert currency to ${code}:`, err);
    } finally {
      loadingRate.value = false;
    }
    rates.value[code] = 1.0;
    return 1.0;
  }

  async function setCurrency(code) {
    currentCurrency.value = code;
    try {
      localStorage.setItem('ob_currency', code);
    } catch {
      // ignore
    }
    await fetchRate(code);
  }

  function formatPrice(price) {
    if (!price && price !== 0) return `${activeSymbol.value}0.00`;
    let usd = 0;
    if (typeof price === 'number') {
      usd = price;
    } else if (typeof price === 'object') {
      const units = price.units !== undefined ? price.units : 0;
      const nanos = price.nanos !== undefined ? price.nanos : 0;
      usd = units + (nanos / 1000000000.0);
    }
    const rate = rates.value[currentCurrency.value] || 1.0;
    const converted = usd * rate;

    const symbol = activeSymbol.value;
    if (currentCurrency.value === 'JPY' || currentCurrency.value === 'KRW') {
      return `${symbol} ${Math.round(converted).toLocaleString()}`;
    }
    return `${symbol} ${converted.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
  }

  function formatRawNumber(usd) {
    const rate = rates.value[currentCurrency.value] || 1.0;
    const converted = usd * rate;
    if (currentCurrency.value === 'JPY' || currentCurrency.value === 'KRW') {
      return Math.round(converted);
    }
    return Number(converted.toFixed(2));
  }

  return {
    currentCurrency,
    supportedCurrencies,
    activeSymbol,
    loadingRate,
    initCurrency,
    setCurrency,
    formatPrice,
    formatRawNumber
  };
}
