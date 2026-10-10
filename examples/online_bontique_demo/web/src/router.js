import { createRouter, createWebHistory } from 'vue-router';
import Home from './views/Home.vue';
import Product from './views/Product.vue';
import Cart from './views/Cart.vue';
import Order from './views/Order.vue';
import AdView from './views/AdView.vue';

const routes = [
  { path: '/', name: 'Home', component: Home },
  { path: '/product/:id', name: 'Product', component: Product },
  { path: '/cart', name: 'Cart', component: Cart },
  { path: '/order', name: 'Order', component: Order },
  { path: '/ad', name: 'Ad', component: AdView }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

export default router;
