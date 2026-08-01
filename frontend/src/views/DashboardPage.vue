<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'

const { t, locale } = useI18n()
const data = ref(null)
const error = ref('')
let timer = null

async function load() {
  try {
    data.value = await api.getDashboard()
    error.value = ''
  } catch (e) {
    error.value = e.message
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const updatedAt = computed(() => {
  if (!data.value?.generated_at) return ''
  return new Date(data.value.generated_at).toLocaleString()
})

function itemName(row) {
  return locale.value === 'zh-CN'
    ? row.name_zh || row.name_en
    : row.name_en || row.name_zh
}

function formatTime(iso) {
  return new Date(iso).toLocaleString(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="board">
    <header class="board-header">
      <div>
        <h1>{{ t('board.title') }}</h1>
        <p class="board-sub">{{ t('app.subtitle') }} · {{ updatedAt }}</p>
      </div>
      <router-link to="/checkout" class="board-exit">{{ t('board.exit') }}</router-link>
    </header>

    <p v-if="error" class="msg error">{{ error }}</p>

    <div v-if="data" class="board-grid">
      <section class="board-panel board-in">
        <h2>{{ t('board.inboundToday') }}</h2>
        <div class="board-stat-row">
          <div class="board-stat">
            <span class="board-num">{{ data.inbound.today_quantity }}</span>
            <span class="board-label">{{ t('board.todayQty') }}</span>
          </div>
          <div class="board-stat">
            <span class="board-num sub">{{ data.inbound.today_transactions }}</span>
            <span class="board-label">{{ t('board.todayCount') }}</span>
          </div>
          <div class="board-stat">
            <span class="board-num sub">{{ data.inbound.week_quantity }}</span>
            <span class="board-label">{{ t('board.weekQty') }}</span>
          </div>
        </div>

        <h3 class="board-h3">{{ t('board.recentIn') }}</h3>
        <ul class="board-feed">
          <li v-for="line in data.recent_inbound" :key="line.id">
            <time>{{ formatTime(line.created_at) }}</time>
            <span class="feed-name">{{ itemName(line) }}</span>
            <span class="feed-qty">+{{ line.quantity }}</span>
            <span v-if="line.operator_name" class="feed-op">{{ line.operator_name }}</span>
          </li>
          <li v-if="!data.recent_inbound.length" class="empty">{{ t('board.noRecent') }}</li>
        </ul>
      </section>

      <section class="board-panel board-alert">
        <h2>
          {{ t('board.lowStock') }}
          <span class="badge" :class="{ hot: data.alert_count > 0 }">{{ data.alert_count }}</span>
        </h2>
        <ul class="board-alerts">
          <li
            v-for="a in data.low_stock_alerts"
            :key="a.id"
            :class="a.level"
          >
            <div class="alert-main">
              <strong>{{ itemName(a) }}</strong>
              <code>{{ a.barcode }}</code>
            </div>
            <div class="alert-meta">
              <span>{{ a.program }} · {{ a.category }}</span>
              <span class="alert-stock">
                {{ t('board.stock') }}: <b>{{ a.quantity }}</b>
                <template v-if="a.min_stock > 0"> / {{ a.min_stock }} {{ a.unit }}</template>
                <template v-else> {{ a.unit }}</template>
              </span>
              <span v-if="a.level === 'critical'" class="tag critical">{{ t('board.critical') }}</span>
              <span v-else class="tag warning">{{ t('board.warning') }}</span>
            </div>
          </li>
          <li v-if="!data.low_stock_alerts.length" class="empty ok-text">{{ t('board.allOk') }}</li>
        </ul>
      </section>
    </div>

    <footer class="board-footer">{{ t('board.autoRefresh') }}</footer>
  </div>
</template>

<style scoped>
.board {
  max-width: 1400px;
  margin: 0 auto;
}

.board-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.board-header h1 {
  margin: 0;
  font-size: 1.75rem;
}

.board-sub {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 0.9rem;
}

.board-exit {
  padding: 8px 16px;
  border-radius: 8px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text);
  font-weight: 600;
}

.board-grid {
  display: grid;
  grid-template-columns: 1fr 1.1fr;
  gap: 20px;
}

@media (max-width: 900px) {
  .board-grid {
    grid-template-columns: 1fr;
  }
}

.board-panel {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 20px;
  min-height: 420px;
}

.board-panel h2 {
  margin: 0 0 16px;
  font-size: 1.25rem;
  display: flex;
  align-items: center;
  gap: 10px;
}

.board-h3 {
  margin: 20px 0 10px;
  font-size: 1rem;
  color: var(--muted);
}

.board-stat-row {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
}

.board-stat {
  display: flex;
  flex-direction: column;
}

.board-num {
  font-size: 3.5rem;
  font-weight: 800;
  line-height: 1;
  color: var(--success);
}

.board-num.sub {
  font-size: 2.25rem;
  color: var(--accent);
}

.board-label {
  margin-top: 6px;
  color: var(--muted);
  font-size: 0.95rem;
}

.board-in {
  border-top: 4px solid var(--success);
}

.board-alert {
  border-top: 4px solid var(--warn);
}

.badge {
  background: var(--border);
  color: var(--text);
  padding: 2px 10px;
  border-radius: 20px;
  font-size: 0.9rem;
}

.badge.hot {
  background: var(--danger);
  color: #fff;
}

.board-feed,
.board-alerts {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
}

.board-feed li,
.board-alerts li {
  padding: 10px 0;
  border-bottom: 1px solid var(--border);
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
}

.board-feed time {
  color: var(--muted);
  font-size: 0.85rem;
  min-width: 100px;
}

.feed-name {
  flex: 1;
  font-weight: 600;
}

.feed-qty {
  color: var(--success);
  font-weight: 700;
}

.feed-op {
  color: var(--muted);
  font-size: 0.85rem;
}

.board-alerts li.critical {
  background: rgba(246, 109, 109, 0.08);
  margin: 0 -12px;
  padding: 12px;
  border-radius: 8px;
}

.alert-main {
  width: 100%;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.alert-main code {
  font-size: 0.8rem;
  color: var(--muted);
}

.alert-meta {
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 0.9rem;
  color: var(--muted);
}

.alert-stock b {
  color: var(--text);
}

.tag {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 700;
}

.tag.critical {
  background: var(--danger);
  color: #fff;
}

.tag.warning {
  background: var(--warn);
  color: #111;
}

.empty {
  color: var(--muted);
  padding: 24px 0;
}

.ok-text {
  color: var(--success);
}

.board-footer {
  text-align: center;
  margin-top: 16px;
  color: var(--muted);
  font-size: 0.85rem;
}
</style>
