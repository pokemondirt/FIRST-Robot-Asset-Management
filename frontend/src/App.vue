<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from './api/client'

const { t, locale } = useI18n()
const route = useRoute()
const isFullscreen = computed(() => route.meta.fullscreen === true)

function toggleLocale() {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  localStorage.setItem('locale', locale.value)
}

// ---- alert sidebar ----
const alerts = ref([])
const alertCount = ref(0)
const sidebarOpen = ref(false)
let timer = null

async function loadAlerts() {
  try {
    const data = await api.getDashboard()
    alerts.value = data.low_stock_alerts || []
    alertCount.value = data.alert_count || 0
  } catch (_) {}
}

onMounted(() => {
  loadAlerts()
  timer = setInterval(loadAlerts, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function alertName(a) {
  return locale.value === 'zh-CN'
    ? a.name_zh || a.name_en
    : a.name_en || a.name_zh
}
</script>

<template>
  <div class="layout" :class="{ 'layout-board': isFullscreen }">
    <header v-if="!isFullscreen" class="header">
      <div>
        <h1>{{ t('app.title') }}</h1>
        <small style="color: var(--muted)">{{ t('app.subtitle') }}</small>
      </div>
      <button class="secondary" type="button" @click="toggleLocale">
        {{ locale === 'zh-CN' ? 'EN' : '中文' }}
      </button>
    </header>
    <nav v-if="!isFullscreen" class="nav">
      <router-link to="/checkout">{{ t('nav.checkout') }}</router-link>
      <router-link to="/items">{{ t('nav.items') }}</router-link>
      <router-link to="/history">{{ t('nav.history') }}</router-link>
      <router-link to="/board">{{ t('nav.board') }}</router-link>
      <router-link to="/settings">{{ t('nav.settings') }}</router-link>
    </nav>
    <div class="body-row">
      <main class="main" :class="{ 'main-board': isFullscreen }">
        <router-view />
      </main>

      <!-- alert sidebar -->
      <aside class="alert-sidebar" :class="{ open: sidebarOpen }">
        <button
          class="sidebar-toggle"
          @click="sidebarOpen = !sidebarOpen"
          :title="sidebarOpen ? '收起' : '库存预警'"
        >
          <span v-if="!sidebarOpen" class="toggle-icon">!</span>
          <span v-else class="toggle-icon">×</span>
          <span v-if="!sidebarOpen && alertCount > 0" class="toggle-badge">{{ alertCount }}</span>
          <span v-if="!sidebarOpen" class="toggle-label">预警</span>
        </button>

        <div v-if="sidebarOpen" class="sidebar-body">
          <h3 class="sidebar-title">
            库存预警
            <span class="badge" :class="{ hot: alertCount > 0 }">{{ alertCount }}</span>
          </h3>

          <ul v-if="alerts.length" class="sidebar-list">
            <li
              v-for="a in alerts"
              :key="a.id"
              :class="a.level"
            >
              <div class="si-name">{{ alertName(a) }}</div>
              <div class="si-code">{{ a.barcode }}</div>
              <div class="si-row">
                <span class="si-stock">
                  库存 <b :class="a.level">{{ a.quantity }}</b>
                  <template v-if="a.min_stock > 0"> / {{ a.min_stock }}</template>
                  {{ a.unit }}
                </span>
                <span class="si-tag" :class="a.level">
                  {{ a.level === 'critical' ? '缺货' : '偏低' }}
                </span>
              </div>
            </li>
          </ul>
          <p v-else class="si-ok">库存正常 ✓</p>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.body-row {
  display: flex;
  flex: 1;
  min-height: 0;
}

/* ---- sidebar ---- */
.alert-sidebar {
  position: relative;
  flex-shrink: 0;
  transition: width 0.2s ease;
  width: 38px;
  border-left: 1px solid var(--border);
  background: var(--surface);
}

.alert-sidebar.open {
  width: 280px;
}

.sidebar-toggle {
  position: absolute;
  top: 50%;
  left: 0;
  transform: translateY(-50%);
  width: 38px;
  height: 80px;
  border-radius: 8px 0 0 8px;
  background: var(--warn);
  border: none;
  color: #111;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 4px;
  font-weight: 700;
  z-index: 2;
}

.sidebar-toggle:hover {
  opacity: 0.9;
}

.alert-sidebar.open .sidebar-toggle {
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--muted);
}

.toggle-icon {
  font-size: 1rem;
  line-height: 1;
}

.toggle-badge {
  background: var(--danger);
  color: #fff;
  border-radius: 10px;
  padding: 0 6px;
  font-size: 0.7rem;
  line-height: 1.4;
}

.toggle-label {
  font-size: 0.65rem;
  writing-mode: vertical-rl;
  letter-spacing: 0.1em;
}

.sidebar-body {
  padding: 14px 12px;
  height: 100%;
  overflow-y: auto;
  margin-left: 38px;
}

.sidebar-title {
  margin: 0 0 12px;
  font-size: 0.95rem;
  display: flex;
  align-items: center;
  gap: 8px;
}

.badge {
  background: var(--border);
  color: var(--text);
  padding: 1px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
}

.badge.hot {
  background: var(--danger);
  color: #fff;
}

.sidebar-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.sidebar-list li {
  padding: 10px;
  margin-bottom: 8px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--bg);
}

.sidebar-list li.critical {
  border-color: var(--danger);
  border-left: 3px solid var(--danger);
}

.si-name {
  font-weight: 600;
  font-size: 0.85rem;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.si-code {
  font-family: monospace;
  font-size: 0.7rem;
  color: var(--muted);
  margin-bottom: 6px;
}

.si-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.si-stock {
  font-size: 0.8rem;
  color: var(--muted);
}

.si-stock b {
  color: var(--text);
}

.si-stock b.critical {
  color: var(--danger);
}

.si-tag {
  font-size: 0.65rem;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 3px;
}

.si-tag.critical {
  background: var(--danger);
  color: #fff;
}

.si-tag.warning {
  background: var(--warn);
  color: #111;
}

.si-ok {
  text-align: center;
  color: var(--success);
  font-size: 0.9rem;
  padding: 20px 0;
}

@media (max-width: 600px) {
  .alert-sidebar.open {
    width: 240px;
  }
}
</style>
