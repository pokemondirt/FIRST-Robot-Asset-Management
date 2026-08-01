import { createRouter, createWebHashHistory } from 'vue-router'
import BatchPage from '../views/BatchPage.vue'
import ItemsPage from '../views/ItemsPage.vue'
import HistoryPage from '../views/HistoryPage.vue'
import SettingsPage from '../views/SettingsPage.vue'
import DashboardPage from '../views/DashboardPage.vue'

const routes = [
  { path: '/', redirect: '/checkout' },
  { path: '/board', component: DashboardPage, meta: { fullscreen: true } },
  { path: '/checkout', component: BatchPage },
  { path: '/items', component: ItemsPage },
  { path: '/history', component: HistoryPage },
  { path: '/settings', component: SettingsPage },
]

export default createRouter({
  history: createWebHashHistory(),
  routes,
})
