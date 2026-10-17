<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'
import { categoryDesc, suggestCategory } from '../data/categories'
import { useUi } from '../composables/useUi'

const { t, locale } = useI18n()
const { openGuide } = useUi()
const items = ref([])
const total = ref(0)
const q = ref('')
const page = ref(1)
const pageSize = 50
const showInactive = ref(false)
const message = ref('')
const messageError = ref(false)
const showForm = ref(false)
const editingId = ref(null)
const saving = ref(false)

const form = ref(emptyForm())
// True once the user picks a category by hand; from then on the keyword
// suggestion stops touching their choice.
const categoryTouched = ref(false)

// Values captured when the edit form opens, so the save handler knows what the
// user actually changed (the stock becomes an ADJUST movement, the barcode is
// only sent when it really differs).
const originalBarcode = ref('')
const originalBarcodeAuto = ref(true)
const originalStock = ref(0)

function emptyForm() {
  return {
    barcode: '',
    name_zh: '',
    name_en: '',
    program: 'BOTH',
    track_mode: 'BLK',
    category: '',
    spec: '',
    unit: 'pcs',
    min_stock: 0,
    note: '',
    location_id: null,
    quantity_initial: 0,
    stock_qty: 0,
  }
}

const barcodeHint = computed(() => {
  if (!editingId.value) return t('items.barcodeAutoHint')
  return originalBarcodeAuto.value ? t('items.barcodeSystemHint') : t('items.barcodeCustomHint')
})

// "其他" is the seeded catch-all; falling back to the first category keeps the
// form usable if someone renamed or deleted it.
function defaultCategory() {
  const misc = categories.value.find((c) => c.name_zh === '其他')
  return misc?.name_zh || categories.value[0]?.name_zh || ''
}

const suggestion = computed(() =>
  suggestCategory(
    [form.value.name_zh, form.value.name_en, form.value.spec].join(' '),
    categories.value.map((c) => c.name_zh),
  ),
)

// Auto-pick the category while the operator types the item name, but only
// until they choose one themselves. This is the part that saves students from
// having to know the taxonomy by heart.
const autoSuggested = computed(
  () =>
    showForm.value &&
    !categoryTouched.value &&
    !!suggestion.value &&
    form.value.category === suggestion.value,
)

watch(
  () => `${form.value.name_zh}|${form.value.name_en}|${form.value.spec}`,
  () => {
    if (!showForm.value || categoryTouched.value) return
    if (suggestion.value) form.value.category = suggestion.value
  },
)

function pickCategory(name) {
  categoryTouched.value = true
  form.value.category = name
}

function resetForm() {
  editingId.value = null
  categoryTouched.value = false
  form.value = emptyForm()
}

const formTitle = computed(() => (editingId.value ? t('items.editTitle') : t('items.add')))

const categories = ref([])
const locations = ref([])

async function loadCategories() {
  try { categories.value = await api.listCategories() } catch (_) { categories.value = [] }
}

async function loadLocations() {
  try { locations.value = await api.listLocations() } catch (_) { locations.value = [] }
}

async function load() {
  const params = { page: page.value, page_size: pageSize }
  if (q.value) params.q = q.value
  if (showInactive.value) params.active_only = 'false'
  try {
    const res = await api.listItems(params)
    items.value = res.items
    total.value = res.total
  } catch (e) {
    items.value = []
    total.value = 0
    message.value = e.message
    messageError.value = true
  }
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function search() {
  page.value = 1
  load()
}

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

onMounted(() => { load(); loadCategories(); loadLocations() })

function displayName(item) {
  return locale.value === 'zh-CN'
    ? item.name_zh || item.name_en
    : item.name_en || item.name_zh
}

function openCreate() {
  resetForm()
  form.value.category = defaultCategory()
  showForm.value = true
}

function openEdit(it) {
  editingId.value = it.id
  categoryTouched.value = true // never override an existing item's category
  originalBarcode.value = it.barcode
  originalBarcodeAuto.value = it.barcode_auto !== false
  originalStock.value = it.quantity
  form.value = {
    barcode: it.barcode,
    name_zh: it.name_zh,
    name_en: it.name_en,
    program: it.program,
    track_mode: it.track_mode,
    category: it.category,
    spec: it.spec || '',
    unit: it.unit,
    min_stock: it.min_stock,
    note: it.note || '',
    location_id: it.location_id,
    quantity_initial: 0,
    stock_qty: it.quantity,
  }
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  resetForm()
}

async function submitForm() {
  saving.value = true
  message.value = ''
  try {
    // v-model.number yields '' when a numeric input is cleared, and the API
    // only accepts integers there, so coerce before sending.
    const asCount = (v) => {
      const n = Number(v)
      return Number.isFinite(n) ? Math.max(0, Math.floor(n)) : 0
    }
    const barcode = (form.value.barcode || '').trim()
    const payload = {
      name_zh: form.value.name_zh,
      name_en: form.value.name_en,
      program: form.value.program,
      track_mode: form.value.track_mode,
      category: form.value.category,
      spec: form.value.spec,
      unit: form.value.unit,
      min_stock: asCount(form.value.min_stock),
      note: form.value.note,
      location_id: form.value.location_id || null,
    }

    if (!editingId.value) {
      payload.quantity_initial = asCount(form.value.quantity_initial)
      // Empty means "let the server number it".
      if (barcode) payload.barcode = barcode
      await api.createItem(payload)
      message.value = t('items.created')
    } else {
      // Only send the barcode when it actually changed, so an untouched form
      // never rewrites it (and never trips the uniqueness check against self).
      if (barcode && barcode !== originalBarcode.value) payload.barcode = barcode

      const updated = await api.updateItem(editingId.value, payload)

      // Stock is not a plain column: changing it records an ADJUST movement so
      // the correction shows up in the history like every other change.
      const newStock = asCount(form.value.stock_qty)
      if (newStock !== originalStock.value) {
        await api.createTransaction({
          barcode: updated.barcode,
          type: 'ADJUST',
          quantity: newStock,
          note: t('items.stockAdjustNote'),
        })
      }
      message.value = t('items.updated')
    }
    messageError.value = false
    closeForm()
    await load()
  } catch (e) {
    message.value = e.message
    messageError.value = true
  } finally {
    saving.value = false
  }
}

async function onImport(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    const res = await api.importFile(file)
    message.value = t('items.imported', {
      created: res.created,
      errors: res.errors.length,
    })
    messageError.value = res.errors.length > 0
    if (res.errors.length) {
      message.value += '  ' + res.errors.slice(0, 3).join(' | ')
    }
    await load()
    await loadLocations()
  } catch (err) {
    message.value = err.message
    messageError.value = true
  }
  e.target.value = ''
}

async function deleteItem(id) {
  try {
    await api.deleteItem(id)
    message.value = t('items.deactivated')
    messageError.value = false
    await load()
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}

async function activateItem(id) {
  try {
    await api.updateItem(id, { active: true })
    message.value = t('items.activated')
    messageError.value = false
    await load()
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}

// ---- purge confirm modal ----
const purgeTarget = ref(null)
const purgePassword = ref('')
const purgeError = ref('')
const purgeBusy = ref(false)

function askPurge(it) {
  purgeTarget.value = it
  purgePassword.value = ''
  purgeError.value = ''
}

function closePurge() {
  purgeTarget.value = null
  purgePassword.value = ''
  purgeError.value = ''
}

async function confirmPurge() {
  if (!purgeTarget.value) return
  purgeBusy.value = true
  purgeError.value = ''
  try {
    await api.purgeItem(purgeTarget.value.id, purgePassword.value)
    closePurge()
    message.value = t('items.purged')
    messageError.value = false
    await load()
  } catch (e) {
    purgeError.value = e.message
  } finally {
    purgeBusy.value = false
  }
}
</script>

<template>
  <div class="items-page">
    <div class="toolbar card">
      <div class="search-box">
        <input
          v-model="q"
          :placeholder="t('items.search')"
          @keydown.enter="search"
        />
        <button type="button" class="secondary" @click="search">↻</button>
      </div>

      <label class="switch">
        <input v-model="showInactive" type="checkbox" @change="search" />
        <span>{{ t('items.showInactive') }}</span>
      </label>

      <span class="spacer"></span>

      <a class="btn secondary" :href="api.templateUrl()" download>{{ t('items.template') }}</a>
      <label class="btn secondary import-btn">
        {{ t('items.import') }}
        <input type="file" accept=".csv" hidden @change="onImport" />
      </label>
      <a class="btn secondary" :href="api.exportCsvUrl()" download>{{ t('items.export') }}</a>
      <button type="button" @click="openCreate">+ {{ t('items.add') }}</button>
    </div>

    <p v-if="message" class="msg" :class="messageError ? 'error' : 'ok'">{{ message }}</p>

    <div class="table-wrap card table-card">
      <table>
        <thead>
          <tr>
            <th>{{ t('item.barcode') }}</th>
            <th>{{ t('item.nameZh') }}</th>
            <th class="num">{{ t('item.stock') }}</th>
            <th class="num">{{ t('item.minStock') }}</th>
            <th>{{ t('item.program') }}</th>
            <th>{{ t('item.category') }}</th>
            <th class="actions-col"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="it in items"
            :key="it.id"
            :class="{ 'row-off': !it.active }"
          >
            <td class="mono">{{ it.barcode }}</td>
            <td>
              <div class="item-name">{{ displayName(it) }}</div>
              <div v-if="it.spec" class="item-spec">{{ it.spec }}</div>
            </td>
            <td class="num">
              <span
                class="stock-num"
                :class="{
                  zero: it.quantity <= 0,
                  low: it.quantity > 0 && it.min_stock > 0 && it.quantity < it.min_stock,
                }"
              >{{ it.quantity }}</span>
            </td>
            <td class="num muted">{{ it.min_stock || '—' }}</td>
            <td>{{ it.program }}</td>
            <td>{{ it.category }}</td>
            <td class="actions-col">
              <button type="button" class="secondary btn-sm" @click="openEdit(it)">
                {{ t('common.edit') }}
              </button>
              <button
                v-if="it.active"
                type="button"
                class="secondary btn-sm"
                @click="deleteItem(it.id)"
              >{{ t('common.delete') }}</button>
              <button
                v-else
                type="button"
                class="secondary btn-sm"
                @click="activateItem(it.id)"
              >{{ t('items.activate') }}</button>
              <button
                type="button"
                class="danger btn-sm"
                :title="t('items.purgeTitle')"
                @click="askPurge(it)"
              >{{ t('common.purge') }}</button>
            </td>
          </tr>
          <tr v-if="!items.length">
            <td colspan="7" class="empty-hint">
              {{ t('items.empty') }}
            </td>
          </tr>
        </tbody>
      </table>

      <div class="pager">
        <button type="button" class="secondary btn-sm" :disabled="page <= 1" @click="prevPage">‹</button>
        <span>{{ page }} / {{ totalPages }} · {{ total }}</span>
        <button type="button" class="secondary btn-sm" :disabled="page >= totalPages" @click="nextPage">›</button>
      </div>
    </div>

    <!-- create / edit -->
    <div v-if="showForm" class="modal-mask" @click.self="closeForm">
      <div class="modal">
        <h3 class="modal-title">{{ formTitle }}</h3>

        <div class="form-grid">
          <div>
            <label>{{ t('item.nameZh') }}</label>
            <input v-model="form.name_zh" />
          </div>
          <div>
            <label>{{ t('item.nameEn') }}</label>
            <input v-model="form.name_en" />
          </div>

          <div class="span-2">
            <label>{{ t('item.barcode') }}</label>
            <input
              v-model="form.barcode"
              class="mono-input"
              autocomplete="off"
              spellcheck="false"
              :placeholder="editingId ? t('items.barcodeKeep') : t('items.barcodeAuto')"
            />
            <small>{{ barcodeHint }}</small>
          </div>

          <div>
            <label>{{ t('item.program') }}</label>
            <select v-model="form.program">
              <option>FRC</option>
              <option>FTC</option>
              <option>BOTH</option>
            </select>
          </div>
          <div>
            <label>{{ t('item.trackMode') }}</label>
            <select v-model="form.track_mode">
              <option value="SNP">{{ t('item.snp') }}</option>
              <option value="BLK">{{ t('item.blk') }}</option>
            </select>
            <small v-if="editingId && originalBarcodeAuto">⚠ 改项目或模式会重新生成条码；填入自带条码后不会再变</small>
          </div>

          <div class="span-2">
            <div class="field-head">
              <label>{{ t('item.category') }}</label>
              <button type="button" class="link-btn" @click="openGuide">{{ t('items.guideLink') }}</button>
            </div>
            <div class="cat-grid">
              <button
                v-for="c in categories"
                :key="c.id"
                type="button"
                class="cat-chip"
                :class="{ active: form.category === c.name_zh }"
                @click="pickCategory(c.name_zh)"
              >
                <span class="cat-name">
                  {{ c.name_zh }}<em v-if="c.name_en">{{ c.name_en }}</em>
                </span>
                <span v-if="categoryDesc(c)" class="cat-desc">{{ categoryDesc(c) }}</span>
              </button>
            </div>
            <p v-if="autoSuggested" class="cat-hint">
              {{ t('items.autoSuggested', { name: form.category }) }}
            </p>
          </div>
          <div>
            <label>{{ t('item.location') }}</label>
            <select v-model="form.location_id">
              <option :value="null">—</option>
              <option v-for="loc in locations" :key="loc.id" :value="loc.id">
                {{ loc.code }}{{ loc.name_zh ? ' ' + loc.name_zh : '' }}
              </option>
            </select>
          </div>

          <div>
            <label>Spec / 规格</label>
            <input v-model="form.spec" />
          </div>
          <div>
            <label>{{ t('item.unit') }}</label>
            <input v-model="form.unit" />
          </div>

          <div>
            <label>{{ t('item.minStock') }}</label>
            <input v-model.number="form.min_stock" type="number" min="0" />
          </div>
          <div v-if="!editingId">
            <label>{{ t('items.initialQty') }}</label>
            <input v-model.number="form.quantity_initial" type="number" min="0" />
          </div>
          <div v-else>
            <label>{{ t('item.stock') }}</label>
            <input v-model.number="form.stock_qty" type="number" min="0" />
            <small>{{ t('items.stockHint') }}</small>
          </div>

          <div class="span-2">
            <label>{{ t('scan.note') }}</label>
            <input v-model="form.note" />
          </div>
        </div>

        <div class="modal-actions">
          <button type="button" class="secondary" @click="closeForm">{{ t('common.cancel') }}</button>
          <button type="button" :disabled="saving" @click="submitForm">
            {{ saving ? t('common.loading') : t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- purge confirm -->
    <div v-if="purgeTarget" class="modal-mask" @click.self="closePurge">
      <div class="modal narrow">
        <h3 class="modal-title danger-title">
          ⚠ {{ t('items.purgeTitle') }}
        </h3>
        <p class="modal-text">
          {{ t('items.purgeWarn') }}
        </p>
        <div class="purge-target">
          <strong>{{ displayName(purgeTarget) }}</strong>
          <span class="mono">{{ purgeTarget.barcode }}</span>
        </div>
        <label>{{ t('items.purgePassword') }}</label>
        <input
          v-model="purgePassword"
          type="password"
          :placeholder="t('items.purgePasswordHint')"
          @keydown.enter.prevent="confirmPurge"
        />
        <p v-if="purgeError" class="msg error">{{ purgeError }}</p>
        <div class="modal-actions">
          <button type="button" class="secondary" @click="closePurge" :disabled="purgeBusy">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="danger-solid" @click="confirmPurge" :disabled="purgeBusy">
            {{ purgeBusy ? t('common.loading') : t('items.purgeConfirm') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.items-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
}

.search-box {
  display: flex;
  gap: 6px;
  flex: 1;
  min-width: 220px;
  max-width: 380px;
}

.search-box input {
  flex: 1;
  min-width: 0;
}

.search-box button {
  flex: none;
  padding: 0 12px;
}

.switch {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--muted);
  font-size: 0.8125rem;
  cursor: pointer;
  white-space: nowrap;
}

.switch input {
  width: auto;
  margin: 0;
  padding: 0;
}

.switch span {
  font-weight: 600;
}

.spacer {
  flex: 1;
}

.import-btn {
  cursor: pointer;
}

.table-card {
  padding: 6px 14px 14px;
}

.item-name {
  font-weight: 600;
}

.item-spec {
  font-size: 0.75rem;
  color: var(--muted);
}

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.muted {
  color: var(--muted);
}

.actions-col {
  width: 1%;
  white-space: nowrap;
  text-align: right;
}

.btn-sm {
  padding: 4px 10px;
  font-size: 0.75rem;
}

.stock-num {
  font-weight: 700;
  color: var(--success);
}

.stock-num.low { color: var(--warn); }
.stock-num.zero { color: var(--danger); }

.row-off {
  opacity: 0.5;
}

/* category picker */
.field-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
}

.field-head label {
  margin-bottom: 0;
}

.link-btn {
  padding: 0;
  background: transparent;
  border: none;
  color: var(--accent);
  font-size: 0.75rem;
  font-weight: 600;
}

.link-btn:hover:not(:disabled) {
  background: transparent;
  text-decoration: underline;
}

.cat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  gap: 6px;
}

.cat-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 8px 10px;
  text-align: left;
  white-space: normal;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text-soft);
  font-weight: 500;
}

.cat-chip:hover:not(:disabled) {
  background: var(--surface-3);
  border-color: var(--border-strong);
}

.cat-chip.active {
  background: var(--accent-soft);
  border-color: var(--accent);
  color: var(--text);
  box-shadow: 0 0 0 2px var(--accent-soft);
}

.cat-name {
  font-size: 0.8125rem;
  font-weight: 650;
  color: var(--text);
}

.cat-name em {
  margin-left: 5px;
  font-style: normal;
  font-weight: 500;
  font-size: 0.6875rem;
  color: var(--muted);
}

.cat-desc {
  font-size: 0.6875rem;
  line-height: 1.35;
  color: var(--muted);
}

.cat-hint {
  margin: 8px 0 0;
  font-size: 0.75rem;
  color: var(--success);
}

.mono-input {
  font-family: ui-monospace, 'Cascadia Mono', Consolas, monospace;
  letter-spacing: 0.02em;
}

.narrow {
  width: 420px;
}

.danger-title {
  color: var(--danger);
}

.modal-text {
  margin: 0 0 12px;
  color: var(--muted);
  font-size: 0.875rem;
}

.purge-target {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 10px;
  border-radius: var(--radius);
  background: var(--surface-2);
}

.danger-solid {
  background: var(--danger);
  border-color: var(--danger);
  color: #fff;
}

.danger-solid:hover:not(:disabled) {
  filter: brightness(1.08);
}
</style>
