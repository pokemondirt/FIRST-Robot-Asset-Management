<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, playBeep } from '../api/client'
import { useOperators } from '../composables/useOperators'

const { t, locale } = useI18n()
const { operators, loadOperators } = useOperators()

// ---------------------------------------------------------------- state

// Returning an item is just checking it back in, so the UI only offers these
// two operations. The API still accepts RETURN for rows created by older
// versions; they keep rendering in the history list.
const MODES = [
  { value: 'IN', labelKey: 'batch.in', cls: 'mode-in' },
  { value: 'OUT', labelKey: 'batch.out', cls: 'mode-out' },
]
// The working mode is the missing piece of the old flow: a warehouse decides
// "I am receiving" or "I am issuing" once, then every scan applies at once.
const mode = ref(localStorage.getItem('opMode') || 'OUT')
const operatorId = ref(null)
const quantity = ref(1)
const note = ref('')

const code = ref('')
const busy = ref(false)
const scanRef = ref(null)

// Barcode scanners behave like keyboards, so a Chinese IME swallows the scan
// into a pinyin composition and Enter never reaches us. A web page cannot
// change the OS input method, but it can throw the composed text away instead
// of turning it into a bogus lookup.
const imeState = ref('idle') // 'idle' | 'warn'

// A web page cannot change the operating system's input method. What it can do
// is publish "the barcode box is focused" through the window title, which the
// optional helper in scripts/ime/ watches to switch the IME to English. The
// marker is a zero-width space, so the visible tab title does not change.
const BASE_TITLE = typeof document !== 'undefined' ? document.title : ''
const IME_MARKER = '\u200B'

function setScanFocus(focused) {
  if (typeof document === 'undefined') return
  document.title = focused ? IME_MARKER + BASE_TITLE : BASE_TITLE
}

const current = ref(null) // last scanned item, refreshed after each movement
const ops = ref([]) // this session, newest first
const errorText = ref('')
const okText = ref('')

let refocusTimer = null

const modeLabel = computed(
  () => t(MODES.find((m) => m.value === mode.value)?.labelKey || 'batch.out'),
)
const modeClass = computed(
  () => MODES.find((m) => m.value === mode.value)?.cls || 'mode-out',
)

const totals = computed(() => {
  const s = { IN: 0, OUT: 0, count: 0 }
  for (const op of ops.value) {
    if (op.undone) continue
    s[op.type] = (s[op.type] || 0) + op.quantity
    s.count += 1
  }
  return s
})

// ---------------------------------------------------------------- helpers

function displayName(item) {
  if (!item) return ''
  return locale.value === 'zh-CN'
    ? item.name_zh || item.name_en
    : item.name_en || item.name_zh
}

function focusScan() {
  setTimeout(() => scanRef.value?.focus(), 30)
}

function applyError(e) {
  switch (e?.code) {
    case 'NOT_FOUND':
      return t('scan.unknown')
    case 'INSUFFICIENT_STOCK':
      return t('op.insufficient', {
        have: e.payload?.have ?? '?',
        need: e.payload?.need ?? '?',
      })
    case 'INVALID_QUANTITY':
      return t('op.invalidQty')
    case 'OPERATOR_INVALID':
      return t('op.operatorInvalid')
    case 'INVALID_TYPE':
      return t('op.invalidType')
    default:
      return e?.message || String(e)
  }
}

function setMode(next) {
  mode.value = next
  localStorage.setItem('opMode', next)
  focusScan()
}

// ---------------------------------------------------------------- IME guard

// Any character outside printable ASCII means the IME was active (CJK text or
// full-width letters/digits). A barcode never contains those.
const NON_ASCII = /[^\x20-\x7e]/
let composing = false

function warnIme() {
  imeState.value = 'warn'
  okText.value = ''
  code.value = ''
  playBeep(false)
}

function onCompositionStart() {
  // Fires as soon as the IME starts a composition: drop whatever it produces.
  composing = true
  imeState.value = 'warn'
}

function onCompositionEnd() {
  // The composition text is pinyin/Chinese; never let it reach the server.
  composing = false
  warnIme()
  focusScan()
}

function onCodeInput(e) {
  // Read the DOM value directly: it is correct regardless of whether v-model
  // has already written the new value into `code`.
  const value = e?.target?.value ?? code.value
  if (composing) return // pinyin is ASCII, so ignore it until the composition ends
  if (NON_ASCII.test(value)) {
    warnIme()
    return
  }
  // Plain ASCII after a warning means the operator switched back to English.
  if (value) imeState.value = 'idle'
}

// ---------------------------------------------------------------- actions

async function onScan() {
  const raw = code.value.trim()
  if (!raw || busy.value) return
  errorText.value = ''
  okText.value = ''

  if (NON_ASCII.test(raw)) {
    warnIme()
    return
  }

  if (!operatorId.value) {
    errorText.value = t('op.scanBlocked')
    playBeep(false)
    return
  }

  busy.value = true
  try {
    // Look the code up first: an unknown barcode and a rejected movement need
    // different messages, and the item tells us whether the quantity is fixed.
    const found = await api.lookup(raw)
    if (found.kind !== 'item' || !found.item) {
      const err = new Error(t('scan.unknown'))
      err.code = 'NOT_FOUND'
      throw err
    }
    const item = found.item
    const qty = item.track_mode === 'SNP' ? 1 : Math.max(1, Number(quantity.value) || 1)

    const tx = await api.createTransaction({
      barcode: item.barcode,
      type: mode.value,
      quantity: qty,
      operator_id: operatorId.value,
      note: note.value,
    })

    // Re-read the item so the stock shown is the post-movement value.
    let fresh = item
    try {
      const after = await api.lookup(item.barcode)
      if (after.kind === 'item' && after.item) fresh = after.item
    } catch (_) {}
    current.value = fresh

    ops.value.unshift({
      id: tx.id,
      type: tx.type,
      barcode: tx.barcode,
      name: displayName(fresh),
      unit: fresh.unit,
      quantity: tx.quantity,
      at: new Date(),
      undone: false,
    })
    if (ops.value.length > 60) ops.value.length = 60

    okText.value = `${t('op.done')} · ${displayName(fresh)} ${tx.quantity}${fresh.unit || ''}`
    playBeep(true)
    code.value = ''
    quantity.value = 1
    imeState.value = 'idle'
  } catch (e) {
    errorText.value = applyError(e)
    playBeep(false)
  } finally {
    busy.value = false
    focusScan()
  }
}

const INVERSE = { IN: 'OUT', OUT: 'IN' }

async function undo(op) {
  if (op.undone || !INVERSE[op.type]) return
  errorText.value = ''
  okText.value = ''
  try {
    await api.createTransaction({
      barcode: op.barcode,
      type: INVERSE[op.type],
      quantity: op.quantity,
      operator_id: operatorId.value,
      note: `撤销 #${op.id}`,
    })
    op.undone = true
    okText.value = t('op.undone')
    if (current.value?.barcode === op.barcode) {
      try {
        const after = await api.lookup(op.barcode)
        if (after.kind === 'item' && after.item) current.value = after.item
      } catch (_) {}
    }
  } catch (e) {
    errorText.value = applyError(e)
  }
  focusScan()
}

function clearSession() {
  ops.value = []
  okText.value = ''
  errorText.value = ''
  focusScan()
}

// ---------------------------------------------------------------- lifecycle

onMounted(async () => {
  await loadOperators()
  const saved = localStorage.getItem('operatorId')
  if (saved) {
    const id = Number(saved)
    if (operators.value.some((o) => o.id === id)) operatorId.value = id
  }
  focusScan()
  // A scanner gun types wherever focus happens to be. Recover an idle focus so
  // a scan is never silently lost after clicking a button or a table row.
  refocusTimer = setInterval(() => {
    if (document.activeElement === document.body) focusScan()
  }, 700)
})

onUnmounted(() => {
  if (refocusTimer) clearInterval(refocusTimer)
  if (typeof document !== 'undefined') document.title = BASE_TITLE
})

watch(operatorId, (v) => {
  if (v) localStorage.setItem('operatorId', String(v))
  focusScan()
})
</script>

<template>
  <div class="op">
    <!-- toolbar: mode, operator, quantity, note -->
    <div class="card op-bar">
      <div class="op-field">
        <label>{{ t('op.mode') }}</label>
        <div class="segmented">
          <button
            v-for="m in MODES"
            :key="m.value"
            type="button"
            :class="[m.cls, { active: mode === m.value }]"
            @click="setMode(m.value)"
          >
            {{ t(m.labelKey) }}
          </button>
        </div>
      </div>

      <div class="op-field op-operator">
        <label>{{ t('op.operator') }}</label>
        <select v-model="operatorId" :class="{ 'is-missing': !operatorId }">
          <option :value="null">— {{ t('op.selectOperator') }} —</option>
          <option v-for="op in operators" :key="op.id" :value="op.id">
            {{ op.display_name }}
          </option>
        </select>
      </div>

      <div class="op-field op-qty">
        <label>{{ t('op.qty') }}</label>
        <input v-model.number="quantity" type="number" min="1" step="1" />
      </div>

      <div class="op-field op-note">
        <label>{{ t('op.note') }}</label>
        <input v-model="note" :placeholder="t('op.notePlaceholder')" />
      </div>
    </div>

    <!-- scan box -->
    <div class="card scan-card" :class="modeClass">
      <input
        ref="scanRef"
        v-model="code"
        class="scan-input"
        type="text"
        lang="en"
        inputmode="latin"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :disabled="busy || !operatorId"
        :placeholder="operatorId ? t('op.scanReady', { mode: modeLabel }) : t('op.scanBlocked')"
        @keydown.enter.prevent="onScan"
        @input="onCodeInput"
        @focus="setScanFocus(true)"
        @blur="setScanFocus(false)"
        @compositionstart="onCompositionStart"
        @compositionend="onCompositionEnd"
      />
      <p class="ime-hint" :class="{ warn: imeState === 'warn' }">
        {{ imeState === 'warn' ? t('op.imeDetected') : t('op.imeHint') }}
      </p>
      <p v-if="errorText" class="msg error">{{ errorText }}</p>
      <p v-else-if="okText" class="msg ok">{{ okText }}</p>
    </div>

    <!-- current item + this session -->
    <div class="op-grid">
      <section class="card current-card">
        <div class="card-title">
          {{ t('op.current') }}
          <span v-if="current" class="sub">{{ current.program }} · {{ current.track_mode }}</span>
        </div>
        <template v-if="current">
          <div class="current-name">{{ displayName(current) }}</div>
          <div class="mono">{{ current.barcode }}</div>
          <div class="current-stats">
            <div>
              <span class="stat-label">{{ t('op.stock') }}</span>
              <span
                class="stat-value"
                :class="{
                  zero: current.quantity <= 0,
                  low: current.quantity > 0 && current.min_stock > 0 && current.quantity < current.min_stock,
                }"
              >{{ current.quantity }}<small>{{ current.unit }}</small></span>
            </div>
            <div v-if="current.location_code">
              <span class="stat-label">{{ t('op.location') }}</span>
              <span class="stat-value small">{{ current.location_code }}</span>
            </div>
            <div v-if="current.min_stock > 0">
              <span class="stat-label">{{ t('item.minStock') }}</span>
              <span class="stat-value small">{{ current.min_stock }}</span>
            </div>
          </div>
        </template>
        <p v-else class="empty-hint">{{ t('op.noCurrent') }}</p>
      </section>

      <section class="card session-card">
        <div class="card-title">
          {{ t('op.session') }}
          <span class="sub">{{ t('op.count', { n: totals.count }) }}</span>
          <button
            v-if="ops.length"
            type="button"
            class="ghost clear-btn"
            @click="clearSession"
          >{{ t('op.clear') }}</button>
        </div>

        <div class="totals">
          <span class="total in">{{ t('batch.in') }} {{ totals.IN }}</span>
          <span class="total out">{{ t('batch.out') }} {{ totals.OUT }}</span>
        </div>

        <ul v-if="ops.length" class="op-list">
          <li v-for="op in ops" :key="op.id" :class="{ undone: op.undone }">
            <span class="op-type" :class="op.type.toLowerCase()">{{ t(`batch.${op.type.toLowerCase()}`) }}</span>
            <span class="op-name">{{ op.name }}</span>
            <span class="op-qty">{{ op.quantity }}{{ op.unit }}</span>
            <span class="op-time">{{ op.at.toLocaleTimeString() }}</span>
            <button
              v-if="!op.undone"
              type="button"
              class="ghost undo-btn"
              @click="undo(op)"
            >{{ t('op.undo') }}</button>
            <span v-else class="tag">{{ t('op.undone') }}</span>
          </li>
        </ul>
        <p v-else class="empty-hint">{{ t('op.sessionEmpty') }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.op {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* toolbar */
.op-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 16px;
  padding: 14px 16px;
}

.op-field {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.op-field label {
  margin-bottom: 5px;
}

.op-operator {
  min-width: 170px;
}

.op-qty {
  width: 92px;
}

.op-qty input {
  text-align: center;
  font-weight: 650;
}

.op-note {
  flex: 1;
  min-width: 160px;
}

select.is-missing {
  border-color: var(--warn);
  box-shadow: 0 0 0 3px var(--warn-soft);
}

/* scan */
.scan-card {
  position: relative;
  padding: 16px;
  border-left: 4px solid var(--muted);
}

.scan-card.mode-in { border-left-color: var(--success); }
.scan-card.mode-out { border-left-color: var(--danger); }

.scan-input {
  width: 100%;
  padding: 16px 18px;
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: 0.02em;
  text-align: center;
  border: 2px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--surface-2);
}

.scan-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 4px var(--accent-soft);
}

.scan-input:disabled {
  opacity: 0.6;
}

.scan-card .msg {
  margin-top: 10px;
}

.ime-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 10px 0 0;
  font-size: 0.8125rem;
  color: var(--muted);
}

.ime-hint::before {
  content: '';
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--success);
  flex: none;
}

.ime-hint.warn {
  padding: 8px 10px;
  border-radius: var(--radius);
  background: var(--danger-soft);
  color: var(--danger);
  font-weight: 650;
}

.ime-hint.warn::before {
  background: var(--danger);
}

/* two columns */
.op-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.25fr);
  gap: 14px;
  align-items: start;
}

@media (max-width: 900px) {
  .op-grid {
    grid-template-columns: 1fr;
  }
}

/* current item */
.current-card {
  min-height: 190px;
}

.current-name {
  font-size: 1.25rem;
  font-weight: 700;
  line-height: 1.3;
  margin-bottom: 2px;
}

.current-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 26px;
  margin-top: 16px;
}

.stat-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted);
}

.stat-value {
  font-size: 1.9rem;
  font-weight: 750;
  line-height: 1.15;
  color: var(--success);
}

.stat-value small {
  margin-left: 3px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--muted);
}

.stat-value.small {
  font-size: 1.0625rem;
  color: var(--text);
}

.stat-value.low { color: var(--warn); }
.stat-value.zero { color: var(--danger); }

/* session */
.session-card {
  min-height: 190px;
}

.clear-btn {
  margin-left: auto;
  padding: 3px 8px;
  font-size: 0.75rem;
  font-weight: 600;
}

.totals {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
}

.total {
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 650;
  background: var(--surface-3);
  color: var(--text-soft);
}

.total.in { background: var(--success-soft); color: var(--success); }
.total.out { background: var(--danger-soft); color: var(--danger); }

.op-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
}

.op-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 0;
  border-bottom: 1px solid var(--border);
  font-size: 0.875rem;
}

.op-list li:last-child {
  border-bottom: none;
}

.op-list li.undone {
  opacity: 0.45;
}

.op-list li.undone .op-name {
  text-decoration: line-through;
}

.op-type {
  flex: none;
  padding: 1px 7px;
  border-radius: var(--radius-sm);
  font-size: 0.6875rem;
  font-weight: 700;
}

.op-type.in { background: var(--success-soft); color: var(--success); }
.op-type.out { background: var(--danger-soft); color: var(--danger); }
/* Legacy rows created by older versions (type RETURN) still need a colour. */
.op-type.return { background: var(--accent-soft); color: var(--accent); }

.op-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.op-qty {
  flex: none;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.op-time {
  flex: none;
  font-size: 0.75rem;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

.undo-btn {
  flex: none;
  padding: 2px 8px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted);
}

.undo-btn:hover:not(:disabled) {
  color: var(--danger);
  background: var(--danger-soft);
}
</style>
