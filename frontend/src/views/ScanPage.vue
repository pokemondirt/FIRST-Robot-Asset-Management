<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api, playBeep } from '../api/client'
import { useOperators } from '../composables/useOperators'

const props = defineProps({
  mode: { type: String, required: true },
})

const { t, locale } = useI18n()
const scanRef = ref(null)
const code = ref('')
const item = ref(null)
const quantity = ref(1)
const operatorId = ref(null)
const route = useRoute()
const { operators, loadOperators } = useOperators()
const note = ref('')
const message = ref('')
const messageOk = ref(true)
const flashClass = ref('')
const loading = ref(false)

const isQuery = computed(() => props.mode === 'QUERY')
const stockClass = computed(() => {
  if (!item.value) return ''
  const q = item.value.quantity
  if (q <= 0) return 'empty'
  if (item.value.min_stock > 0 && q < item.value.min_stock) return 'low'
  return ''
})

const displayName = computed(() => {
  if (!item.value) return ''
  return locale.value === 'zh-CN'
    ? item.value.name_zh || item.value.name_en
    : item.value.name_en || item.value.name_zh
})

async function initOperators() {
  await loadOperators()
  const saved = localStorage.getItem('operatorId')
  if (saved) {
    const id = Number(saved)
    if (operators.value.some((o) => o.id === id)) {
      operatorId.value = id
    }
  }
}

onMounted(async () => {
  await initOperators()
  focusScan()
})

watch(
  () => route.path,
  async () => {
    await initOperators()
  }
)

watch(operatorId, (v) => {
  if (v) localStorage.setItem('operatorId', String(v))
})

function focusScan() {
  setTimeout(() => scanRef.value?.focus(), 50)
}

async function onScanEnter() {
  const raw = code.value.trim()
  if (!raw) return
  message.value = ''
  loading.value = true
  try {
    const res = await api.lookup(raw)
    if (res.kind !== 'item' || !res.item) {
      item.value = null
      message.value = t('scan.unknown')
      messageOk.value = false
      flashClass.value = 'flash-err'
      playBeep(false)
      code.value = ''
      focusScan()
      return
    }
    item.value = res.item
    quantity.value = res.item.track_mode === 'SNP' ? 1 : (quantity.value || 1)
    flashClass.value = ''
    playBeep(true)
    if (isQuery.value) {
      code.value = ''
      focusScan()
      return
    }
    code.value = ''
    if (!operatorId.value) {
      message.value = t('scan.selectOperator')
      messageOk.value = false
      return
    }
  } catch (e) {
    message.value = e.message
    messageOk.value = false
    playBeep(false)
  } finally {
    loading.value = false
  }
}

async function confirm() {
  if (!item.value || isQuery.value) return
  if (!operatorId.value) {
    message.value = t('scan.selectOperator')
    messageOk.value = false
    return
  }
  loading.value = true
  message.value = ''
  try {
    const result = await api.createTransaction({
      barcode: item.value.barcode,
      type: props.mode,
      quantity: Number(quantity.value) || 1,
      operator_id: operatorId.value,
      note: note.value,
    })
    item.value = {
      ...item.value,
      quantity:
        props.mode === 'OUT'
          ? item.value.quantity - result.quantity
          : item.value.quantity + result.quantity,
    }
    message.value = `${t('scan.success')} — ${t('scan.remaining')}: ${item.value.quantity}`
    messageOk.value = true
    flashClass.value = 'flash-ok'
    playBeep(true)
    code.value = ''
    note.value = ''
    focusScan()
  } catch (e) {
    message.value = e.message
    messageOk.value = false
    flashClass.value = 'flash-err'
    playBeep(false)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div :class="flashClass">
    <input
      ref="scanRef"
      v-model="code"
      class="scan-input"
      type="text"
      autocomplete="off"
      :placeholder="t('scan.placeholder')"
      @keydown.enter.prevent="onScanEnter"
    />

    <div class="card" style="margin-top: 16px">
      <label>{{ t('scan.operator') }}</label>
      <select v-model="operatorId">
        <option :value="null">—</option>
        <option v-for="op in operators" :key="op.id" :value="op.id">
          {{ op.display_name }}
        </option>
      </select>
    </div>

    <div
      v-if="item"
      class="item-display card"
      :class="stockClass"
      style="margin-top: 16px"
    >
      <div style="color: var(--muted); font-family: monospace">{{ item.barcode }}</div>
      <h2>{{ displayName }}</h2>
      <p style="color: var(--muted); margin: 0">
        {{ item.program }} · {{ item.track_mode }} · {{ item.category }}
      </p>
      <div class="stock">{{ item.quantity }} {{ item.unit }}</div>
      <p v-if="item.location_code" style="color: var(--muted)">
        {{ t('item.location') }}: {{ item.location_code }}
      </p>
    </div>

    <div v-if="item && !isQuery" class="card" style="margin-top: 16px">
      <label v-show="item.track_mode === 'BLK'">{{ t('scan.quantity') }}</label>
      <input
        v-show="item.track_mode === 'BLK'"
        v-model="quantity"
        class="qty-input"
      />
      <label>{{ t('scan.note') }}</label>
      <input v-model="note" type="text" />
      <div class="row-actions">
        <button type="button" :disabled="loading" @click="confirm">
          {{ t('scan.confirm') }}
        </button>
      </div>
    </div>

    <p v-if="message" class="msg" :class="messageOk ? 'ok' : 'error'">{{ message }}</p>
  </div>
</template>

<style scoped>
.qty-input {
  font-size: 1.5rem !important;
  text-align: center;
  font-weight: 700;
  margin-bottom: 12px;
}
</style>
