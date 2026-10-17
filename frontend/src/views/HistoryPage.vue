<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'

const { t, locale } = useI18n()
const rows = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const typeFilter = ref('')
const error = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

// Labels for every type that can appear in the data, including RETURN and
// ADJUST rows written by older versions or by direct API calls.
const TYPE_LABEL = {
  IN: { cls: 'in', key: 'batch.in' },
  OUT: { cls: 'out', key: 'batch.out' },
  RETURN: { cls: 'return', key: 'batch.return' },
  ADJUST: { cls: 'adjust', key: 'history.adjust' },
}

// The UI only ever creates IN and OUT now (returning an item is just checking
// it back in), so the filter only offers those two.
const FILTERS = ['IN', 'OUT']

async function load() {
  const params = { page: page.value, page_size: pageSize }
  if (typeFilter.value) params.type = typeFilter.value
  try {
    const res = await api.listTransactions(params)
    rows.value = res.transactions
    total.value = res.total
    error.value = ''
  } catch (e) {
    rows.value = []
    total.value = 0
    error.value = e.message
  }
}

onMounted(load)

function prevPage() {
  if (page.value > 1) {
    page.value--
    load()
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    load()
  }
}

function onFilter() {
  page.value = 1
  load()
}

function name(tx) {
  return locale.value === 'zh-CN'
    ? tx.name_zh || tx.name_en
    : tx.name_en || tx.name_zh
}

function formatTime(iso) {
  return new Date(iso).toLocaleString(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="history">
    <div class="toolbar card">
      <div class="segmented">
        <button
          type="button"
          :class="{ active: typeFilter === '' }"
          @click="typeFilter = ''; onFilter()"
        >{{ t('history.all') }}</button>
        <button
          v-for="ft in FILTERS"
          :key="ft"
          type="button"
          :class="{ active: typeFilter === ft }"
          @click="typeFilter = ft; onFilter()"
        >{{ t(TYPE_LABEL[ft].key) }}</button>
      </div>
      <span class="count">{{ t('history.count', { n: total }) }}</span>
      <button type="button" class="secondary" @click="load">{{ t('history.refresh') }}</button>
    </div>

    <p v-if="error" class="msg error">{{ error }}</p>

    <div class="table-wrap card table-card">
      <table>
        <thead>
          <tr>
            <th>{{ t('history.time') }}</th>
            <th>{{ t('history.type') }}</th>
            <th>{{ t('item.barcode') }}</th>
            <th>{{ t('history.item') }}</th>
            <th class="num">{{ t('scan.quantity') }}</th>
            <th>{{ t('scan.operator') }}</th>
            <th>{{ t('scan.note') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tx in rows" :key="tx.id">
            <td class="mono nowrap">{{ formatTime(tx.created_at) }}</td>
            <td>
              <span class="type" :class="TYPE_LABEL[tx.type]?.cls">
                {{ TYPE_LABEL[tx.type] ? t(TYPE_LABEL[tx.type].key) : tx.type }}
              </span>
            </td>
            <td class="mono">{{ tx.barcode }}</td>
            <td>{{ name(tx) }}</td>
            <td class="num qty">{{ tx.quantity }}</td>
            <td class="muted">{{ tx.operator_name || '—' }}</td>
            <td class="muted note">{{ tx.note || '—' }}</td>
          </tr>
          <tr v-if="!rows.length">
            <td colspan="7" class="empty-hint">{{ t('history.empty') }}</td>
          </tr>
        </tbody>
      </table>

      <div class="pager">
        <button type="button" class="secondary btn-sm" :disabled="page <= 1" @click="prevPage">‹</button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button type="button" class="secondary btn-sm" :disabled="page >= totalPages" @click="nextPage">›</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.history {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
}

.count {
  color: var(--muted);
  font-size: 0.8125rem;
}

.toolbar > button {
  margin-left: auto;
}

.table-card {
  padding: 6px 14px 14px;
}

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.nowrap {
  white-space: nowrap;
}

.muted {
  color: var(--muted);
}

.qty {
  font-weight: 700;
}

.note {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type {
  display: inline-block;
  padding: 1px 7px;
  border-radius: var(--radius-sm);
  font-size: 0.6875rem;
  font-weight: 700;
}

.type.in { background: var(--success-soft); color: var(--success); }
.type.out { background: var(--danger-soft); color: var(--danger); }
.type.return { background: var(--accent-soft); color: var(--accent); }
.type.adjust { background: var(--warn-soft); color: var(--warn); }

.btn-sm {
  padding: 3px 9px;
  font-size: 0.75rem;
}
</style>
