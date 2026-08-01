<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, playBeep } from '../api/client'
import { useOperators } from '../composables/useOperators'
import GuideSidebar from '../components/GuideSidebar.vue'

const { t, locale } = useI18n()
const { operators, loadOperators } = useOperators()
const scanRef = ref(null)
const code = ref('')
const operatorId = ref(null)
const note = ref('')
const message = ref('')
const messageOk = ref(true)
const loading = ref(false)
const submitting = ref(false)

// Cart lines: each has barcode, item info, type, quantity
const lines = ref([])
const nextId = ref(1)
const selectedIds = ref(new Set())

function toggleSelect(id) {
  const s = new Set(selectedIds.value)
  s.has(id) ? s.delete(id) : s.add(id)
  selectedIds.value = s
}

function toggleSelectAll() {
  if (selectedIds.value.size === lines.value.length) {
    selectedIds.value = new Set()
  } else {
    selectedIds.value = new Set(lines.value.map(l => l.id))
  }
}

function batchSetType(type) {
  const count = selectedIds.value.size
  for (const id of selectedIds.value) {
    setLineType(id, type)
  }
  selectedIds.value = new Set()
  message.value = `${count} ${locale.value === 'zh-CN' ? '种' : 'items'} → ${typeLabel(type)}`
  messageOk.value = true
}

function displayName(item) {
  if (!item) return ''
  return locale.value === 'zh-CN'
    ? item.name_zh || item.name_en
    : item.name_en || item.name_zh
}

function typeLabel(type) {
  if (type === 'OUT') return t('batch.out')
  if (type === 'IN') return t('batch.in')
  if (type === 'RETURN') return t('batch.return')
  return type
}

const summary = computed(() => {
  const s = { out: 0, in: 0, return: 0, outCount: 0, inCount: 0, returnCount: 0 }
  for (const line of lines.value) {
    if (line.type === 'OUT') { s.out += line.quantity; s.outCount++ }
    if (line.type === 'IN') { s.in += line.quantity; s.inCount++ }
    if (line.type === 'RETURN') { s.return += line.quantity; s.returnCount++ }
  }
  return s
})

onMounted(async () => {
  await loadOperators()
  const saved = localStorage.getItem('operatorId')
  if (saved) {
    const id = Number(saved)
    if (operators.value.some((o) => o.id === id)) {
      operatorId.value = id
    }
  }
  focusScan()
})

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
      message.value = t('scan.unknown')
      messageOk.value = false
      playBeep(false)
      code.value = ''
      focusScan()
      return
    }
    const item = res.item

    // Check if already in cart
    const existing = lines.value.find(l => l.barcode === item.barcode)
    if (existing) {
      existing.quantity++
      message.value = `${displayName(item)} +1 → ${existing.quantity}`
      messageOk.value = true
      playBeep(true)
    } else {
      // Default type: if item is checked out to someone, default to RETURN; otherwise OUT
      const defaultType = 'OUT'
      lines.value.push({
        id: nextId.value++,
        barcode: item.barcode,
        item: item,
        type: defaultType,
        quantity: 1,
      })
      message.value = `${t('batch.added')}: ${displayName(item)}`
      messageOk.value = true
      playBeep(true)
    }
    code.value = ''
    focusScan()
  } catch (e) {
    message.value = e.message
    messageOk.value = false
    playBeep(false)
  } finally {
    loading.value = false
  }
}

function removeLine(id) {
  lines.value = lines.value.filter(l => l.id !== id)
}

function setLineType(id, type) {
  const line = lines.value.find(l => l.id === id)
  if (line) line.type = type
}

function setLineQty(id, qty) {
  const line = lines.value.find(l => l.id === id)
  if (!line) return
  if (line.item.track_mode === 'SNP') {
    line.quantity = 1
    return
  }
  const v = parseInt(qty, 10)
  if (!isNaN(v) && v >= 1) {
    line.quantity = v
  }
}

// Validation per line
function lineValid(line) {
  if (line.type === 'OUT' && line.item.quantity < line.quantity) return false
  return true
}

function lineError(line) {
  if (line.type === 'OUT' && line.item.quantity < line.quantity) {
    return t('batch.insufficient', { have: line.item.quantity, need: line.quantity })
  }
  return ''
}

const allValid = computed(() => lines.value.length > 0 && lines.value.every(lineValid))

async function confirmAll() {
  if (lines.value.length === 0) return
  if (!operatorId.value) {
    message.value = t('scan.selectOperator')
    messageOk.value = false
    return
  }
  if (!allValid.value) return

  submitting.value = true
  message.value = ''
  try {
    const result = await api.batchCreateTransactions({
      operator_id: operatorId.value,
      note: note.value,
      lines: lines.value.map(l => ({
        barcode: l.barcode,
        type: l.type,
        quantity: l.quantity,
      })),
    })

    const ok = result.transactions?.length || 0
    const err = result.errors?.length || 0

    if (err > 0 && ok === 0) {
      message.value = t('batch.allFailed')
      messageOk.value = false
      playBeep(false)
    } else if (err > 0) {
      message.value = t('batch.partialSuccess', { ok, err })
      messageOk.value = false
      playBeep(false)
      // Remove succeeded lines from cart, keep errors
      const errorBarcodes = new Set(result.errors.map(e => e.barcode))
      lines.value = lines.value.filter(l => errorBarcodes.has(l.barcode))
    } else {
      message.value = t('batch.allSuccess', { count: ok })
      messageOk.value = true
      playBeep(true)
      lines.value = []
      note.value = ''
      selectedIds.value = new Set()
    }
    focusScan()
  } catch (e) {
    message.value = e.message
    messageOk.value = false
    playBeep(false)
  } finally {
    submitting.value = false
  }
}

function clearAll() {
  lines.value = []
  note.value = ''
  message.value = ''
  selectedIds.value = new Set()
}
</script>

<template>
  <div class="batch-layout">
    <GuideSidebar />

    <div class="batch-main">
    <!-- Scan bar -->
    <div class="scan-bar">
      <input
        ref="scanRef"
        v-model="code"
        class="scan-input"
        type="text"
        autocomplete="off"
        :placeholder="t('batch.scanPlaceholder')"
        @keydown.enter.prevent="onScanEnter"
      />
    </div>

    <!-- Operator & note -->
    <div class="meta-row">
      <div class="meta-item">
        <label>{{ t('scan.operator') }}</label>
        <select v-model="operatorId">
          <option :value="null">—</option>
          <option v-for="op in operators" :key="op.id" :value="op.id">
            {{ op.display_name }}
          </option>
        </select>
      </div>
      <div class="meta-item" style="flex:1">
        <label>{{ t('scan.note') }}</label>
        <input v-model="note" type="text" />
      </div>
    </div>

    <!-- Cart -->
    <div v-if="lines.length" class="cart card">
      <table>
        <thead>
          <tr>
            <th style="width:30px">
              <input type="checkbox" :checked="selectedIds.size === lines.length && lines.length > 0" @change="toggleSelectAll" />
            </th>
            <th>{{ t('item.barcode') }}</th>
            <th>{{ t('item.nameZh') }}</th>
            <th style="width:80px">{{ t('item.stock') }}</th>
            <th style="width:100px">{{ t('batch.operation') }}</th>
            <th style="width:80px">{{ t('scan.quantity') }}</th>
            <th style="width:40px"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="line in lines"
            :key="line.id"
            :class="{ 'row-err': !lineValid(line) }"
          >
            <td>
              <input type="checkbox" :checked="selectedIds.has(line.id)" @change="toggleSelect(line.id)" />
            </td>
            <td style="font-family:monospace;font-size:0.8rem">{{ line.barcode }}</td>
            <td>
              <div class="line-name">{{ displayName(line.item) }}</div>
              <div v-if="lineError(line)" class="line-err-msg">{{ lineError(line) }}</div>
            </td>
            <td>
              <span class="stock-num" :class="{
                low: line.item.quantity > 0 && line.item.min_stock > 0 && line.item.quantity < line.item.min_stock,
                zero: line.item.quantity <= 0
              }">{{ line.item.quantity }}</span>
            </td>
            <td>
              <select
                :value="line.type"
                @change="setLineType(line.id, $event.target.value)"
                class="type-select"
              >
                <option value="OUT">{{ t('batch.out') }}</option>
                <option value="IN">{{ t('batch.in') }}</option>
                <option value="RETURN">{{ t('batch.return') }}</option>
              </select>
            </td>
            <td>
              <input
                :value="line.quantity"
                @change="setLineQty(line.id, $event.target.value)"
                type="number"
                min="1"
                class="qty-input"
                :disabled="line.item.track_mode === 'SNP'"
                :title="line.item.track_mode === 'SNP' ? 'SNP 单品模式，数量固定为 1' : ''"
              />
            </td>
            <td>
              <button
                type="button"
                class="btn-remove"
                @click="removeLine(line.id)"
                :title="locale === 'zh-CN' ? '移除' : 'Remove'"
              >✕</button>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Summary -->
      <div class="cart-summary">
        <span v-if="summary.outCount" class="sum-tag out">{{ t('batch.out') }}: {{ summary.outCount }}种 / {{ summary.out }}件</span>
        <span v-if="summary.inCount" class="sum-tag in">{{ t('batch.in') }}: {{ summary.inCount }}种 / {{ summary.in }}件</span>
        <span v-if="summary.returnCount" class="sum-tag return">{{ t('batch.return') }}: {{ summary.returnCount }}种 / {{ summary.return }}件</span>
        <span class="sum-total">{{ lines.length }} {{ locale === 'zh-CN' ? '种' : 'items' }}</span>
      </div>
    </div>

    <!-- Batch type switcher -->
    <div v-if="selectedIds.size > 0" class="batch-bar">
      <span class="batch-bar-label">{{ selectedIds.size }} {{ locale === 'zh-CN' ? '种已选 → 改为' : ' selected → set to' }}</span>
      <button type="button" class="btn-batch out" @click="batchSetType('OUT')">{{ t('batch.out') }}</button>
      <button type="button" class="btn-batch in" @click="batchSetType('IN')">{{ t('batch.in') }}</button>
      <button type="button" class="btn-batch return" @click="batchSetType('RETURN')">{{ t('batch.return') }}</button>
    </div>

    <div v-else class="empty-hint card">
      {{ t('batch.emptyHint') }}
    </div>

    <!-- Action bar -->
    <div class="action-bar" v-if="lines.length">
      <button
        type="button"
        class="btn-clear secondary"
        @click="clearAll"
      >{{ t('batch.clearAll') }}</button>
      <button
        type="button"
        class="btn-confirm"
        :disabled="!allValid || submitting || !operatorId"
        @click="confirmAll"
      >
        {{ submitting ? t('common.loading') : t('batch.confirmAll') }}
      </button>
    </div>

    <p v-if="message" class="msg" :class="messageOk ? 'ok' : 'error'">{{ message }}</p>
    </div><!-- /batch-main -->
  </div><!-- /batch-layout -->
</template>

<style scoped>
.batch-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.batch-main {
  flex: 1;
  min-width: 0;
  max-width: 960px;
}

.scan-bar { margin-bottom: 12px; }

.scan-input {
  width: 100%;
  padding: 14px 16px;
  font-size: 1.15rem;
  border: 2px solid var(--accent);
  border-radius: 12px;
  background: var(--bg);
  color: var(--text);
  outline: none;
}

.scan-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(99,102,241,0.15);
}

.meta-row {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 140px;
}

.meta-item label {
  font-size: 0.8rem;
  color: var(--muted);
}

.meta-item select,
.meta-item input {
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text);
  font-size: 0.95rem;
}

/* Cart table */
.cart {
  padding: 0;
  overflow: hidden;
  overflow-x: auto;
  margin-bottom: 12px;
}

.cart table {
  width: 100%;
  border-collapse: collapse;
}

.cart th {
  text-align: left;
  padding: 10px 12px;
  font-size: 0.8rem;
  color: var(--muted);
  border-bottom: 1px solid var(--border);
  background: var(--bg);
}

.cart td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

.cart tbody tr:last-child td {
  border-bottom: none;
}

.cart tbody tr:hover {
  background: rgba(99,102,241,0.04);
}

.line-name {
  font-weight: 600;
  font-size: 0.95rem;
}

.line-err-msg {
  color: var(--danger);
  font-size: 0.75rem;
  margin-top: 2px;
}

.row-err {
  background: rgba(246,109,109,0.08);
}

.stock-num {
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--success);
}

.stock-num.low { color: var(--warn); }
.stock-num.zero { color: var(--danger); }

.type-select {
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  color: var(--text);
  font-size: 0.85rem;
  width: 100%;
}

.qty-input {
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  color: var(--text);
  font-size: 0.9rem;
  width: 70px;
  text-align: center;
}

.qty-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: var(--surface);
}

.btn-remove {
  border: none;
  background: none;
  color: var(--muted);
  cursor: pointer;
  font-size: 1rem;
  padding: 4px 8px;
  border-radius: 4px;
}

.btn-remove:hover {
  color: var(--danger);
  background: rgba(246,109,109,0.1);
}

/* Summary */
.cart-summary {
  padding: 12px 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  border-top: 1px solid var(--border);
  background: var(--bg);
}

.sum-tag {
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 0.8rem;
  font-weight: 600;
}

.sum-tag.out { background: rgba(246,109,109,0.12); color: var(--danger); }
.sum-tag.in { background: rgba(34,197,94,0.12); color: var(--success); }
.sum-tag.return { background: rgba(99,102,241,0.12); color: var(--accent); }

.sum-total {
  margin-left: auto;
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--muted);
}

/* Batch type bar */
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  margin-bottom: 12px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 10px;
}

.batch-bar-label {
  font-size: 0.85rem;
  color: var(--muted);
  margin-right: 4px;
}

.btn-batch {
  padding: 6px 16px;
  border-radius: 8px;
  border: none;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  color: #fff;
}

.btn-batch.out { background: var(--danger); }
.btn-batch.in { background: var(--success); }
.btn-batch.return { background: var(--accent); }

.btn-batch:hover { filter: brightness(1.15); }

/* Empty */
.empty-hint {
  text-align: center;
  padding: 48px 16px;
  color: var(--muted);
  font-size: 1rem;
}

/* Action bar */
.action-bar {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-bottom: 12px;
}

.btn-clear {
  padding: 10px 20px;
  border-radius: 10px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--muted);
  font-size: 0.95rem;
  cursor: pointer;
}

.btn-clear:hover {
  color: var(--danger);
  border-color: var(--danger);
}

.btn-confirm {
  padding: 10px 32px;
  border-radius: 10px;
  border: none;
  background: var(--accent);
  color: #fff;
  font-size: 1.05rem;
  font-weight: 700;
  cursor: pointer;
}

.btn-confirm:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.btn-confirm:not(:disabled):hover {
  filter: brightness(1.1);
}
</style>
