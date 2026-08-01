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
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

async function load() {
  const params = { page: page.value, page_size: pageSize }
  if (typeFilter.value) params.type = typeFilter.value
  const res = await api.listTransactions(params)
  rows.value = res.transactions
  total.value = res.total
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
  return locale.value === 'zh-CN' ? tx.name_zh : tx.name_en
}
</script>

<template>
  <div class="card table-wrap">
    <h3 style="margin-top: 0">{{ t('nav.history') }}</h3>
    <div style="display:flex;gap:10px;align-items:center;margin-bottom:10px">
      <select v-model="typeFilter" @change="onFilter" style="padding:6px 10px">
        <option value="">{{ locale === 'zh-CN' ? '全部类型' : 'All types' }}</option>
        <option value="IN">IN</option>
        <option value="OUT">OUT</option>
        <option value="RETURN">RETURN</option>
        <option value="ADJUST">ADJUST</option>
      </select>
    </div>
    <table>
      <thead>
        <tr>
          <th>Time</th>
          <th>Type</th>
          <th>{{ t('item.barcode') }}</th>
          <th>Name</th>
          <th>Qty</th>
          <th>{{ t('scan.operator') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="tx in rows" :key="tx.id">
          <td>{{ new Date(tx.created_at).toLocaleString() }}</td>
          <td>{{ tx.type }}</td>
          <td>{{ tx.barcode }}</td>
          <td>{{ name(tx) }}</td>
          <td>{{ tx.quantity }}</td>
          <td>{{ tx.operator_name || '—' }}</td>
        </tr>
      </tbody>
    </table>
    <div class="pager" style="display:flex;align-items:center;gap:10px;margin-top:10px;color:var(--muted)">
      <button type="button" class="secondary" style="padding:2px 10px;font-size:0.8rem" :disabled="page <= 1" @click="prevPage">‹</button>
      <span>{{ page }} / {{ totalPages }} · {{ total }} {{ t('nav.history') }}</span>
      <button type="button" class="secondary" style="padding:2px 10px;font-size:0.8rem" :disabled="page >= totalPages" @click="nextPage">›</button>
    </div>
  </div>
</template>
