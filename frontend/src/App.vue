<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from './api/client'
import { useTheme } from './composables/useTheme'
import { useUi } from './composables/useUi'
import AppDrawer from './components/AppDrawer.vue'
import CategoryGuide from './components/CategoryGuide.vue'

const { t, locale } = useI18n()
const route = useRoute()
const { preference: themePref, cycleTheme } = useTheme()
const { guideOpen, openGuide } = useUi()

const isFullscreen = computed(() => route.meta.fullscreen === true)

function toggleLocale() {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN'
  localStorage.setItem('locale', locale.value)
}

const themeTitle = computed(() => {
  if (themePref.value === 'auto') return t('theme.auto')
  return themePref.value === 'light' ? t('theme.light') : t('theme.dark')
})

// ---- drawers ----
const showAlerts = ref(false)

// ---- alert data ----
const alerts = ref([])
const alertCount = ref(0)
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
  <div class="app">
    <header v-if="!isFullscreen" class="topbar">
      <div class="brand">
        <strong>{{ t('app.title') }}</strong>
        <span>{{ t('app.subtitle') }}</span>
      </div>

      <nav class="nav">
        <router-link to="/checkout" class="nav-link">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M4 8V5a1 1 0 0 1 1-1h3M20 8V5a1 1 0 0 0-1-1h-3M4 16v3a1 1 0 0 0 1 1h3M20 16v3a1 1 0 0 1-1 1h-3" stroke-linecap="round" />
            <path d="M7 12h10" stroke-linecap="round" />
          </svg>
          <span>{{ t('nav.checkout') }}</span>
        </router-link>
        <router-link to="/items" class="nav-link">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M3.5 7.5 12 3l8.5 4.5v9L12 21l-8.5-4.5z" stroke-linejoin="round" />
            <path d="M3.5 7.5 12 12l8.5-4.5M12 12v9" stroke-linejoin="round" />
          </svg>
          <span>{{ t('nav.items') }}</span>
        </router-link>
        <router-link to="/history" class="nav-link">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M4 6h16M4 12h16M4 18h10" stroke-linecap="round" />
          </svg>
          <span>{{ t('nav.history') }}</span>
        </router-link>
      </nav>

      <div class="topbar-actions">
        <button
          type="button"
          class="icon-btn"
          :title="t('alerts.title')"
          @click="showAlerts = true"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M12 3a6 6 0 0 0-6 6v4l-1.5 3h15L18 13V9a6 6 0 0 0-6-6Z" stroke-linejoin="round" />
            <path d="M10 19a2 2 0 0 0 4 0" stroke-linecap="round" />
          </svg>
          <span v-if="alertCount > 0" class="icon-badge">{{ alertCount }}</span>
        </button>

        <button
          type="button"
          class="nav-link guide-btn"
          :title="t('guide.title')"
          @click="openGuide"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M4.5 5.5A1.5 1.5 0 0 1 6 4h5v16H6a1.5 1.5 0 0 1-1.5-1.5z" stroke-linejoin="round" />
            <path d="M19.5 5.5A1.5 1.5 0 0 0 18 4h-5v16h5a1.5 1.5 0 0 0 1.5-1.5z" stroke-linejoin="round" />
          </svg>
          <span>{{ t('nav.guide') }}</span>
        </button>

        <span class="divider" aria-hidden="true"></span>

        <router-link to="/board" class="nav-link board-link">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <rect x="3" y="4" width="18" height="12" rx="1.5" />
            <path d="M9 20h6M12 16v4" stroke-linecap="round" />
          </svg>
          <span>{{ t('nav.board') }}</span>
        </router-link>

        <router-link to="/settings" class="icon-btn" :title="t('nav.settings')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-2.9 1.2 2 2 0 1 1-4 0 1.7 1.7 0 0 0-2.9-1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1A1.7 1.7 0 0 0 3.6 15a2 2 0 1 1 0-4 1.7 1.7 0 0 0 1.2-2.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1A1.7 1.7 0 0 0 10.5 4a2 2 0 1 1 4 0 1.7 1.7 0 0 0 2.9 1.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0 1.2 2.9 2 2 0 1 1 0 4Z" stroke-linejoin="round" />
          </svg>
        </router-link>

        <button type="button" class="ghost-lang" @click="toggleLocale">
          {{ locale === 'zh-CN' ? 'EN' : '中文' }}
        </button>

        <button type="button" class="icon-btn" :title="themeTitle" @click="cycleTheme">
          <svg v-if="themePref === 'auto'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <circle cx="12" cy="12" r="7.5" />
            <path d="M12 4.5v15" />
            <path d="M12 4.5a7.5 7.5 0 0 1 0 15z" fill="currentColor" stroke="none" />
          </svg>
          <svg v-else-if="themePref === 'light'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <circle cx="12" cy="12" r="4" />
            <path d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.2 5.2l1.4 1.4M17.4 17.4l1.4 1.4M18.8 5.2l-1.4 1.4M6.6 17.4l-1.4 1.4" stroke-linecap="round" />
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
            <path d="M20 13.5A8 8 0 0 1 10.5 4a8 8 0 1 0 9.5 9.5Z" stroke-linejoin="round" />
          </svg>
        </button>
      </div>
    </header>

    <main class="main" :class="{ 'main-board': isFullscreen }">
      <router-view />
    </main>

    <!-- 库存预警 -->
    <AppDrawer v-model="showAlerts" :title="t('alerts.title')">
      <p v-if="!alerts.length" class="empty-hint">{{ t('alerts.none') }}</p>
      <ul v-else class="alert-list">
        <li v-for="a in alerts" :key="a.id" :class="a.level">
          <div class="alert-main">
            <strong>{{ alertName(a) }}</strong>
            <span class="tag" :class="a.level === 'critical' ? 'critical' : 'warning'">
              {{ a.level === 'critical' ? t('alerts.critical') : t('alerts.warning') }}
            </span>
          </div>
          <div class="alert-meta">
            <code class="mono">{{ a.barcode }}</code>
            <span>
              {{ t('alerts.stock') }}
              <b>{{ a.quantity }}</b>
              <template v-if="a.min_stock > 0"> / {{ a.min_stock }}</template>
              {{ a.unit }}
            </span>
          </div>
        </li>
      </ul>
    </AppDrawer>

    <!-- 分类说明 -->
    <AppDrawer v-model="guideOpen" :title="t('guide.title')" width="460px">
      <CategoryGuide />
    </AppDrawer>
  </div>
</template>

<style scoped>
.divider {
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: var(--border);
}

/* The guide entry uses .nav-link styling but is a <button>, so the global
   button background has to be reset. */
.guide-btn {
  background: transparent;
  border-color: transparent;
  font-weight: 600;
}

.guide-btn:hover:not(:disabled) {
  background: var(--surface-3);
  color: var(--text);
}

.guide-btn svg {
  width: 17px;
  height: 17px;
}

.board-link {
  border: 1px solid var(--border);
  color: var(--text-soft);
}

.ghost-lang {
  min-width: 44px;
  height: 34px;
  padding: 0 8px;
  background: transparent;
  border-color: transparent;
  color: var(--text-soft);
  font-size: 0.8125rem;
}

.ghost-lang:hover:not(:disabled) {
  background: var(--surface-3);
  color: var(--text);
}

.alert-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.alert-list li {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-left: 3px solid var(--warn);
  border-radius: var(--radius);
  background: var(--surface-2);
}

.alert-list li.critical {
  border-left-color: var(--danger);
}

.alert-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.alert-main strong {
  font-size: 0.875rem;
}

.alert-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
  font-size: 0.8125rem;
  color: var(--muted);
}

.alert-meta b {
  color: var(--text);
}

@media (max-width: 900px) {
  .board-link span {
    display: none;
  }
  .board-link {
    padding: 8px;
  }
  .divider {
    display: none;
  }
}
</style>
