<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'
import GuideSidebar from '../components/GuideSidebar.vue'

const { t, locale } = useI18n()
const items = ref([])
const total = ref(0)
const q = ref('')
const page = ref(1)
const pageSize = 50
const showInactive = ref(false)
const message = ref('')
const showForm = ref(false)
const editingId = ref(null)
const saving = ref(false)

const form = ref({
  name_zh: '',
  name_en: '',
  program: 'BOTH',
  track_mode: 'BLK',
  category: 'MISC',
  spec: '',
  unit: 'pcs',
  min_stock: 0,
  note: '',
  location_id: null,
  quantity_initial: 0,
})

function resetForm() {
  editingId.value = null
  form.value = {
    name_zh: '', name_en: '', program: 'BOTH', track_mode: 'BLK',
    category: 'MISC', spec: '', unit: 'pcs', min_stock: 0,
    note: '', location_id: null, quantity_initial: 0,
  }
}

const formTitle = computed(() => {
  if (editingId.value) return locale.value === 'zh-CN' ? '编辑物资' : 'Edit Item'
  return t('items.add')
})

// Categories
const categories = ref([])

// Locations
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
  const res = await api.listItems(params)
  items.value = res.items
  total.value = res.total
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
  showForm.value = true
}

function openEdit(it) {
  editingId.value = it.id
  form.value = {
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
  }
  showForm.value = true
}

async function submitForm() {
  saving.value = true
  try {
    if (editingId.value) {
      const body = {
        name_zh: form.value.name_zh,
        name_en: form.value.name_en,
        program: form.value.program,
        track_mode: form.value.track_mode,
        category: form.value.category,
        spec: form.value.spec,
        unit: form.value.unit,
        min_stock: form.value.min_stock,
        note: form.value.note,
        location_id: form.value.location_id,
      }
      await api.updateItem(editingId.value, body)
      message.value = locale.value === 'zh-CN' ? '已更新' : 'Updated'
    } else {
      await api.createItem(form.value)
      message.value = 'OK'
    }
    showForm.value = false
    resetForm()
    await load()
  } catch (e) {
    message.value = e.message
  } finally {
    saving.value = false
  }
}

async function onImport(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    const res = await api.importFile(file)
    message.value = `Created: ${res.created}, errors: ${res.errors.length}`
    await load()
  } catch (err) { message.value = err.message }
  e.target.value = ''
}

async function deleteItem(id) {
  try {
    await api.deleteItem(id)
    message.value = locale.value === 'zh-CN' ? '已停用' : 'Deactivated'
    await load()
  } catch (e) { message.value = e.message }
}

async function activateItem(id) {
  try {
    await api.updateItem(id, { active: true })
    message.value = locale.value === 'zh-CN' ? '已启用' : 'Activated'
    await load()
  } catch (e) { message.value = e.message }
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
    message.value = locale.value === 'zh-CN' ? '已彻底删除' : 'Deleted'
    await load()
  } catch (e) {
    purgeError.value = e.message
  } finally {
    purgeBusy.value = false
  }
}
</script>

<template>
  <div class="items-layout">
    <GuideSidebar :default-open="false" />

    <!-- Main content -->
    <div class="items-main">
    <div class="items-toolbar">
      <div class="toolbar-row">
        <div class="search-box">
          <input
            v-model="q"
            :placeholder="t('items.search')"
            @keydown.enter="search"
          />
          <button type="button" class="secondary" @click="search" title="刷新">↻</button>
        </div>
        <label class="inactive-toggle">
          <input type="checkbox" v-model="showInactive" @change="search" />
          {{ locale === 'zh-CN' ? '显示已停用' : 'Show inactive' }}
        </label>
      </div>

      <div class="toolbar-row">
        <button type="button" @click="openCreate">{{ t('items.add') }}</button>
        <a class="btn secondary" :href="api.templateUrl()" download>{{ t('items.template') }}</a>
        <label class="btn secondary">
          {{ t('items.import') }}
          <input type="file" accept=".csv" hidden @change="onImport" />
        </label>
        <a class="btn secondary" :href="api.exportCsvUrl()" download>{{ t('items.export') }}</a>
      </div>
    </div>

    <!-- Form -->
    <div v-if="showForm" class="card form-grid" style="margin-top: 16px">
      <h3 style="margin: 0 0 4px; grid-column: 1 / -1">{{ formTitle }}</h3>

      <div>
        <label>{{ t('item.nameZh') }}</label>
        <input v-model="form.name_zh" />
      </div>
      <div>
        <label>{{ t('item.nameEn') }}</label>
        <input v-model="form.name_en" />
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
        <small v-if="editingId" style="color: var(--warn)">⚠ 修改模式将自动生成新条码</small>
      </div>
      <div>
        <label>{{ t('item.category') }}</label>
        <select v-model="form.category">
          <option v-for="c in categories" :key="c.id" :value="c.name_zh">
            {{ c.name_zh }}{{ c.name_en ? ' / ' + c.name_en : '' }}
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
        <small style="color: var(--muted)">低于此数量触发预警</small>
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
        <label>{{ t('scan.note') }}</label>
        <input v-model="form.note" />
      </div>
      <div v-if="!editingId">
        <label>初始库存</label>
        <input v-model.number="form.quantity_initial" type="number" min="0" />
      </div>

      <div class="row-actions" style="grid-column: 1 / -1; margin-top: 0">
        <button type="button" :disabled="saving" @click="submitForm">{{ t('common.save') }}</button>
        <button type="button" class="secondary" @click="showForm = false; resetForm()">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <p v-if="message" class="msg ok">{{ message }}</p>

    <!-- Table -->
    <div class="table-wrap card" style="margin-top: 16px">
      <table>
        <thead>
          <tr>
            <th>{{ t('item.barcode') }}</th>
            <th>{{ t('item.nameZh') }}</th>
            <th>{{ t('item.stock') }}</th>
            <th>{{ t('item.minStock') }}</th>
            <th>{{ t('item.program') }}</th>
            <th>{{ t('item.category') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="it in items" :key="it.id" :class="{ 'row-off': !it.active, 'row-low': it.quantity <= 0, 'row-warn': it.quantity > 0 && it.min_stock > 0 && it.quantity < it.min_stock }">
            <td style="font-family: monospace; font-size: 0.85rem">{{ it.barcode }}</td>
            <td>
              <div>{{ displayName(it) }}</div>
              <small v-if="it.spec" style="color: var(--muted)">{{ it.spec }}</small>
            </td>
            <td>
              <span class="stock-num" :class="{ zero: it.quantity <= 0, low: it.quantity > 0 && it.min_stock > 0 && it.quantity < it.min_stock }">
                {{ it.quantity }}
              </span>
            </td>
            <td style="color: var(--muted)">{{ it.min_stock || '—' }}</td>
            <td>{{ it.program }}</td>
            <td>{{ it.category }}</td>
            <td>
              <button type="button" class="secondary" style="padding:4px 10px;font-size:0.8rem" @click="openEdit(it)">
                {{ locale === 'zh-CN' ? '编辑' : 'Edit' }}
              </button>
              <button
                v-if="it.active"
                type="button"
                class="danger"
                style="padding:4px 10px;font-size:0.8rem;margin-left:4px"
                @click="deleteItem(it.id)"
              >{{ t('common.delete') }}</button>
              <button
                v-else
                type="button"
                class="secondary"
                style="padding:4px 10px;font-size:0.8rem;margin-left:4px"
                @click="activateItem(it.id)"
              >{{ locale === 'zh-CN' ? '启用' : 'Activate' }}</button>
              <button
                type="button"
                class="danger"
                style="padding:4px 10px;font-size:0.8rem;margin-left:4px"
                @click="askPurge(it)"
                :title="locale === 'zh-CN' ? '彻底删除（不可恢复）' : 'Purge permanently'"
              >{{ locale === 'zh-CN' ? '删除' : 'Delete' }}</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="pager" style="display:flex;align-items:center;gap:10px;margin-top:8px;color:var(--muted)">
        <button type="button" class="secondary" style="padding:2px 10px;font-size:0.8rem" :disabled="page <= 1" @click="prevPage">‹</button>
        <span>{{ page }} / {{ totalPages }} · {{ total }} total</span>
        <button type="button" class="secondary" style="padding:2px 10px;font-size:0.8rem" :disabled="page >= totalPages" @click="nextPage">›</button>
      </div>
    </div>
    </div><!-- /items-main -->
  </div><!-- /items-layout -->

  <!-- Purge confirm modal -->
  <div v-if="purgeTarget" class="modal-mask" @click.self="closePurge">
    <div class="modal">
      <h3 class="modal-title">⚠ {{ locale === 'zh-CN' ? '彻底删除物资' : 'Permanently delete item' }}</h3>
      <p>
        {{ locale === 'zh-CN' ? '即将删除物资' : 'You are about to delete' }}
        <b style="color: var(--text)">{{ purgeTarget ? displayName(purgeTarget) : '' }}</b>
        {{ locale === 'zh-CN' ? '的档案、库存和全部出入库流水，此操作无法恢复！' : '— its record, stock and all transaction history. This cannot be undone!' }}
      </p>
      <label>{{ locale === 'zh-CN' ? '删除保护密码' : 'Purge password' }}</label>
      <input
        v-model="purgePassword"
        type="password"
        :placeholder="locale === 'zh-CN' ? '输入密码以确认删除' : 'Enter password to confirm'"
        @keydown.enter.prevent="confirmPurge"
      />
      <p v-if="purgeError" class="msg error" style="margin: 10px 0 0">{{ purgeError }}</p>
      <div class="modal-actions">
        <button type="button" class="secondary" @click="closePurge" :disabled="purgeBusy">
          {{ locale === 'zh-CN' ? '取消' : 'Cancel' }}
        </button>
        <button type="button" class="danger" @click="confirmPurge" :disabled="purgeBusy">
          {{ purgeBusy ? (locale === 'zh-CN' ? '删除中…' : 'Deleting…') : (locale === 'zh-CN' ? '确认删除' : 'Delete') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ---- layout ---- */
.items-layout {
  display: flex;
  gap: 16px;
  min-height: 0;
  align-items: flex-start;
}

.items-main {
  flex: 1;
  min-width: 0;
}

/* ---- toolbar ---- */
.items-toolbar {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.toolbar-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

/* 工具栏内按钮统一紧凑尺寸，避免挤占 */
.toolbar-row .btn,
.toolbar-row button {
  padding: 8px 14px;
  font-size: 0.9rem;
}

.search-box {
  display: flex;
  flex: 1;
  min-width: 220px;
  max-width: 420px;
  gap: 8px;
}

.search-box input {
  flex: 1;
  min-width: 0;
}

.inactive-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  white-space: nowrap;
  padding: 9px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--muted);
  font-size: 0.9rem;
  font-weight: 600;
}

.inactive-toggle input {
  width: auto;
  margin: 0;
  padding: 0;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
}
.form-grid label {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
  margin-bottom: 3px;
}
@media (max-width: 500px) {
  .form-grid { grid-template-columns: 1fr; }
}

.stock-num { font-weight: 700; color: var(--success); }
.stock-num.low { color: var(--warn); }
.stock-num.zero { color: var(--danger); }

.row-low { background: rgba(246,109,109,0.06); }
.row-warn { background: rgba(245,166,35,0.06); }
.row-off { opacity: 0.55; }

/* ---- purge confirm modal ---- */
.modal-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  width: 420px;
  max-width: calc(100vw - 40px);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 20px;
}

.modal-title {
  margin: 0 0 10px;
  color: var(--danger);
  font-size: 1.1rem;
}

.modal p {
  margin: 0 0 14px;
  color: var(--muted);
  font-size: 0.95rem;
  line-height: 1.5;
}

.modal label {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
  margin-bottom: 4px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}

small { display: block; margin-top: 2px; }
</style>
