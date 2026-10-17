<script setup>
import { onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'
import { useOperators } from '../composables/useOperators'

const { t } = useI18n()
const settings = ref({ has_purge_password: false })
const stats = ref(null)
const message = ref('')
const messageError = ref(false)
const newOperator = ref('')
const purgePasswordNew = ref('')
const purgePasswordSet = ref(false)
const { operators, loadOperators, addOperator } = useOperators()
const adding = ref(false)

// Locations
const locations = ref([])
const newLoc = ref({ code: '', name_zh: '', name_en: '' })
const editingLocId = ref(null)
const editLoc = ref({ code: '', name_zh: '', name_en: '' })

// Categories
const categories = ref([])
const newCat = ref({ name_zh: '', name_en: '' })
const editingCatId = ref(null)
const editCat = ref({ name_zh: '', name_en: '' })

// Actions are spread over several sections, so feedback shows as a toast
// instead of an inline paragraph that may be scrolled out of view.
let toastTimer = null
watch(message, (v) => {
  if (!v) return
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { message.value = '' }, 4000)
})

function notify(text, isError = false) {
  message.value = text
  messageError.value = isError
}

async function loadLocations() {
  try { locations.value = await api.listLocations() } catch (_) { locations.value = [] }
}

async function loadCats() {
  try { categories.value = await api.listCategories() } catch (_) { categories.value = [] }
}

async function loadStats() {
  try { stats.value = await api.getStats() } catch (_) { stats.value = null }
}

onMounted(async () => {
  try {
    settings.value = await api.getSettings()
    purgePasswordSet.value = !!settings.value.has_purge_password
    await loadStats()
    await loadOperators()
    await loadLocations()
    await loadCats()
  } catch (e) {
    notify(e.message, true)
  }
})

async function backup() {
  try {
    const res = await api.backup()
    notify(`${t('settings.backupDone')}: ${res.filename}`)
  } catch (e) {
    notify(e.message, true)
  }
}

async function savePurgePassword() {
  const pw = purgePasswordNew.value
  if (pw && pw.length < 4) {
    notify(t('settings.passwordTooShort'), true)
    return
  }
  try {
    settings.value = await api.updateSettings({ purge_password: pw })
    purgePasswordSet.value = !!settings.value.has_purge_password
    purgePasswordNew.value = ''
    notify(t('common.saved'))
  } catch (e) {
    notify(e.message, true)
  }
}

async function submitOperator() {
  if (!newOperator.value.trim()) {
    notify(t('operators.nameRequired'), true)
    return
  }
  adding.value = true
  try {
    await addOperator(newOperator.value)
    newOperator.value = ''
    notify(t('operators.added'))
  } catch (e) {
    notify(e.message === 'EMPTY_NAME' ? t('operators.nameRequired') : e.message, true)
  } finally {
    adding.value = false
  }
}

async function removeOperator(id) {
  try {
    await api.deleteOperator(id)
    await loadOperators()
    notify(t('operators.deactivated'))
  } catch (e) {
    notify(e.message, true)
  }
}

// --- Locations ---
async function submitLocation() {
  const code = newLoc.value.code.trim()
  if (!code) return
  try {
    await api.createLocation({
      code,
      name_zh: newLoc.value.name_zh.trim(),
      name_en: newLoc.value.name_en.trim(),
    })
    notify(t('master.locAdded'))
    newLoc.value = { code: '', name_zh: '', name_en: '' }
    await loadLocations()
  } catch (e) {
    notify(e.message, true)
  }
}

function startEditLocation(loc) {
  editingLocId.value = loc.id
  editLoc.value = { code: loc.code, name_zh: loc.name_zh, name_en: loc.name_en }
}

function cancelEditLocation() {
  editingLocId.value = null
}

async function saveEditLocation(id) {
  try {
    await api.updateLocation(id, {
      code: editLoc.value.code.trim(),
      name_zh: editLoc.value.name_zh.trim(),
      name_en: editLoc.value.name_en.trim(),
    })
    notify(t('master.locUpdated'))
    editingLocId.value = null
    await loadLocations()
  } catch (e) {
    notify(e.message, true)
  }
}

async function removeLocation(id) {
  try {
    await api.deleteLocation(id)
    await loadLocations()
    notify(t('master.locDeleted'))
  } catch (e) {
    notify(
      e.code === 'LOCATION_IN_USE'
        ? t('settings.locationInUse', { count: e.payload?.count ?? '?' })
        : e.message,
      true,
    )
  }
}

// --- Categories ---
async function submitCategory() {
  const nameZh = newCat.value.name_zh.trim()
  if (!nameZh) return
  try {
    await api.createCategory({ name_zh: nameZh, name_en: newCat.value.name_en.trim() })
    notify(t('master.catAdded'))
    newCat.value = { name_zh: '', name_en: '' }
    await loadCats()
  } catch (e) {
    notify(e.message, true)
  }
}

function startEditCategory(cat) {
  editingCatId.value = cat.id
  editCat.value = { name_zh: cat.name_zh, name_en: cat.name_en }
}

function cancelEditCategory() {
  editingCatId.value = null
}

async function saveEditCategory(id) {
  try {
    await api.updateCategory(id, {
      name_zh: editCat.value.name_zh.trim(),
      name_en: editCat.value.name_en.trim(),
    })
    notify(t('master.catUpdated'))
    editingCatId.value = null
    await loadCats()
  } catch (e) {
    notify(e.message, true)
  }
}

async function removeCategory(id) {
  try {
    await api.deleteCategory(id)
    await loadCats()
    notify(t('master.catDeleted'))
  } catch (e) {
    notify(
      e.code === 'CATEGORY_IN_USE'
        ? t('settings.categoryInUse', { count: e.payload?.count ?? '?' })
        : e.message,
      true,
    )
  }
}
</script>

<template>
  <div class="settings">
    <!-- 概览 -->
    <section>
      <div class="section-head">
        <h2>{{ t('settings.sectionData') }}</h2>
        <span class="sub">{{ t('settings.sectionDataSub') }}</span>
      </div>

      <div class="grid-2">
        <div class="card">
          <div class="card-title">{{ t('settings.stats') }}</div>
          <div v-if="stats" class="stat-row">
            <div>
              <span class="stat-label">{{ t('settings.totalItems') }}</span>
              <strong>{{ stats.total_items }}</strong>
            </div>
            <div>
              <span class="stat-label">{{ t('settings.lowStock') }}</span>
              <strong :class="{ warn: stats.low_stock_count > 0 }">{{ stats.low_stock_count }}</strong>
            </div>
            <div>
              <span class="stat-label">{{ t('settings.totalUnits') }}</span>
              <strong>{{ stats.total_quantity }}</strong>
            </div>
          </div>
          <p v-else class="desc">{{ t('common.loading') }}</p>
        </div>

        <div class="card">
          <div class="card-title">{{ t('settings.backup') }}</div>
          <p class="desc">{{ t('settings.backupDesc') }}</p>
          <button type="button" class="secondary" @click="backup">{{ t('settings.backup') }}</button>
        </div>
      </div>
    </section>

    <!-- 安全 -->
    <section>
      <div class="section-head">
        <h2>{{ t('settings.sectionSecurity') }}</h2>
        <span class="sub">{{ t('settings.sectionSecuritySub') }}</span>
      </div>

      <div class="card">
        <div class="card-title">
          {{ t('settings.purgeTitle') }}
          <span class="badge" :class="{ hot: !purgePasswordSet }">
            {{ purgePasswordSet ? t('settings.purgeSet') : t('settings.purgeUnset') }}
          </span>
        </div>
        <p class="desc">{{ t('settings.purgeDesc') }}</p>
        <div class="inline-form">
          <input
            v-model="purgePasswordNew"
            type="password"
            :placeholder="t('settings.purgePlaceholder')"
            @keydown.enter.prevent="savePurgePassword"
          />
          <button type="button" @click="savePurgePassword">{{ t('common.save') }}</button>
        </div>
      </div>
    </section>

    <!-- 基础档案 -->
    <section>
      <div class="section-head">
        <h2>{{ t('settings.sectionMaster') }}</h2>
        <span class="sub">{{ t('settings.sectionMasterSub') }}</span>
      </div>

      <div class="grid-3">
        <!-- 操作人 -->
        <div class="card">
          <div class="card-title">
            {{ t('operators.title') }}
            <span class="sub">{{ operators.length }}</span>
          </div>
          <div class="inline-form">
            <input
              v-model="newOperator"
              :placeholder="t('operators.placeholder')"
              @keydown.enter.prevent="submitOperator"
            />
            <button type="button" :disabled="adding" @click="submitOperator">
              {{ t('operators.add') }}
            </button>
          </div>
          <ul v-if="operators.length" class="mgmt-list">
            <li v-for="op in operators" :key="op.id">
              <span>{{ op.display_name }}</span>
              <button type="button" class="icon-text" @click="removeOperator(op.id)">
                {{ t('common.delete') }}
              </button>
            </li>
          </ul>
          <p v-else class="desc">{{ t('operators.empty') }}</p>
        </div>

        <!-- 库位 -->
        <div class="card">
          <div class="card-title">
            {{ t('master.locationTitle') }}
            <span class="sub">{{ locations.length }}</span>
          </div>
          <div class="inline-form">
            <input
              v-model="newLoc.code"
              :placeholder="t('master.locationCode')"
              @keydown.enter.prevent="submitLocation"
            />
            <input
              v-model="newLoc.name_zh"
              :placeholder="t('master.locationName')"
              @keydown.enter.prevent="submitLocation"
            />
            <button type="button" @click="submitLocation">{{ t('operators.add') }}</button>
          </div>

          <ul v-if="locations.length" class="mgmt-list">
            <li v-for="loc in locations" :key="loc.id">
              <template v-if="editingLocId === loc.id">
                <div class="edit-row">
                  <input v-model="editLoc.code" @keydown.enter="saveEditLocation(loc.id)" />
                  <input v-model="editLoc.name_zh" @keydown.enter="saveEditLocation(loc.id)" />
                  <button type="button" class="btn-sm" @click="saveEditLocation(loc.id)">{{ t('common.save') }}</button>
                  <button type="button" class="secondary btn-sm" @click="cancelEditLocation">{{ t('common.cancel') }}</button>
                </div>
              </template>
              <template v-else>
                <span class="mgmt-name">
                  <strong>{{ loc.code }}</strong>
                  <small v-if="loc.name_zh"> — {{ loc.name_zh }}</small>
                </span>
                <span class="mgmt-actions">
                  <button type="button" class="secondary btn-sm" @click="startEditLocation(loc)">
                    {{ t('common.edit') }}
                  </button>
                  <button type="button" class="icon-text danger-text" @click="removeLocation(loc.id)">
                    {{ t('common.remove') }}
                  </button>
                </span>
              </template>
            </li>
          </ul>
          <p v-else class="desc">{{ t('master.none') }}</p>
        </div>

        <!-- 分类 -->
        <div class="card">
          <div class="card-title">
            {{ t('master.categoryTitle') }}
            <span class="sub">{{ categories.length }}</span>
          </div>
          <div class="inline-form">
            <input
              v-model="newCat.name_zh"
              :placeholder="t('master.categoryZh')"
              @keydown.enter.prevent="submitCategory"
            />
            <input
              v-model="newCat.name_en"
              :placeholder="t('master.categoryEn')"
              @keydown.enter.prevent="submitCategory"
            />
            <button type="button" @click="submitCategory">{{ t('operators.add') }}</button>
          </div>

          <ul v-if="categories.length" class="mgmt-list">
            <li v-for="cat in categories" :key="cat.id">
              <template v-if="editingCatId === cat.id">
                <div class="edit-row">
                  <input v-model="editCat.name_zh" @keydown.enter="saveEditCategory(cat.id)" />
                  <input v-model="editCat.name_en" @keydown.enter="saveEditCategory(cat.id)" />
                  <button type="button" class="btn-sm" @click="saveEditCategory(cat.id)">{{ t('common.save') }}</button>
                  <button type="button" class="secondary btn-sm" @click="cancelEditCategory">{{ t('common.cancel') }}</button>
                </div>
              </template>
              <template v-else>
                <span class="mgmt-name">
                  <strong>{{ cat.name_zh }}</strong>
                  <small v-if="cat.name_en"> / {{ cat.name_en }}</small>
                </span>
                <span class="mgmt-actions">
                  <button type="button" class="secondary btn-sm" @click="startEditCategory(cat)">
                    {{ t('common.edit') }}
                  </button>
                  <button type="button" class="icon-text danger-text" @click="removeCategory(cat.id)">
                    {{ t('common.remove') }}
                  </button>
                </span>
              </template>
            </li>
          </ul>
          <p v-else class="desc">{{ t('master.none') }}</p>
        </div>
      </div>
    </section>

    <transition name="toast">
      <div v-if="message" class="toast" :class="{ error: messageError }">{{ message }}</div>
    </transition>
  </div>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
}

.grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.grid-3 {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  align-items: start;
}

@media (max-width: 1000px) {
  .grid-3 { grid-template-columns: 1fr; }
}

@media (max-width: 720px) {
  .grid-2 { grid-template-columns: 1fr; }
}

.desc {
  margin: 0 0 12px;
  color: var(--muted);
  font-size: 0.8125rem;
  line-height: 1.55;
}

.stat-row {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
}

.stat-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--muted);
}

.stat-row strong {
  font-size: 1.75rem;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}

.stat-row strong.warn {
  color: var(--warn);
}

.inline-form {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.inline-form input {
  flex: 1;
  min-width: 110px;
}

.inline-form button {
  flex: none;
}

.mgmt-list {
  list-style: none;
  margin: 12px 0 0;
  padding: 0;
  max-height: 240px;
  overflow-y: auto;
}

.mgmt-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 0;
  border-bottom: 1px solid var(--border);
  font-size: 0.875rem;
}

.mgmt-list li:last-child {
  border-bottom: none;
}

.mgmt-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mgmt-name small {
  color: var(--muted);
}

.mgmt-actions {
  display: flex;
  flex: none;
  gap: 4px;
}

.edit-row {
  display: flex;
  flex: 1;
  gap: 4px;
  align-items: center;
}

.edit-row input {
  min-width: 0;
  flex: 1;
}

.btn-sm {
  padding: 3px 9px;
  font-size: 0.75rem;
}

.icon-text {
  padding: 3px 8px;
  background: transparent;
  border-color: transparent;
  color: var(--muted);
  font-size: 0.75rem;
  font-weight: 600;
}

.icon-text:hover:not(:disabled) {
  background: var(--surface-3);
  color: var(--text);
}

.danger-text:hover:not(:disabled) {
  color: var(--danger);
  background: var(--danger-soft);
}

/* toast */
.toast {
  position: fixed;
  left: 50%;
  bottom: 28px;
  z-index: 120;
  transform: translateX(-50%);
  max-width: min(560px, calc(100vw - 40px));
  padding: 10px 18px;
  border-radius: 999px;
  background: var(--surface-3);
  border: 1px solid var(--border-strong);
  color: var(--text);
  font-size: 0.875rem;
  font-weight: 600;
  box-shadow: var(--shadow-2);
}

.toast.error {
  background: var(--danger);
  border-color: var(--danger);
  color: #fff;
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 8px);
}
</style>
