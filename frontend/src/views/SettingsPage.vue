<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../api/client'
import { useOperators } from '../composables/useOperators'

const { t, locale } = useI18n()
const settings = ref({ label_width_mm: 40, label_height_mm: 30, has_purge_password: false })
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

async function loadLocations() {
  try { locations.value = await api.listLocations() } catch (_) { locations.value = [] }
}

onMounted(async () => {
  try {
    settings.value = await api.getSettings()
    purgePasswordSet.value = !!settings.value.has_purge_password
    stats.value = await api.getStats()
    await loadOperators()
    await loadLocations()
    await loadCats()
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
})

async function save() {
  settings.value = await api.updateSettings({
    label_width_mm: settings.value.label_width_mm,
    label_height_mm: settings.value.label_height_mm,
  })
  message.value = t('common.saved')
  messageError.value = false
}

async function backup() {
  const res = await api.backup()
  message.value = res.filename
  messageError.value = false
}

async function savePurgePassword() {
  const pw = purgePasswordNew.value
  if (pw && pw.length < 4) {
    message.value = locale.value === 'zh-CN' ? '密码至少 4 位' : 'Password must be at least 4 characters'
    messageError.value = true
    return
  }
  try {
    settings.value = await api.updateSettings({ purge_password: pw })
    purgePasswordSet.value = !!settings.value.has_purge_password
    purgePasswordNew.value = ''
    message.value = t('common.saved')
    messageError.value = false
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}

async function submitOperator() {
  message.value = ''
  if (!newOperator.value.trim()) {
    message.value = t('operators.nameRequired')
    messageError.value = true
    return
  }
  adding.value = true
  try {
    await addOperator(newOperator.value)
    newOperator.value = ''
    message.value = t('operators.added')
    messageError.value = false
  } catch (e) {
    message.value =
      e.message === 'EMPTY_NAME' ? t('operators.nameRequired') : e.message
    messageError.value = true
  } finally {
    adding.value = false
  }
}

async function removeOperator(id) {
  try {
    await api.deleteOperator(id)
    await loadOperators()
    message.value = '已删除'
    messageError.value = false
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}

// --- Locations ---
async function submitLocation() {
  const code = newLoc.value.code.trim()
  if (!code) return
  try {
    await api.createLocation({
      code, name_zh: newLoc.value.name_zh.trim(), name_en: newLoc.value.name_en.trim(),
    })
    message.value = '库位已添加'; messageError.value = false
    newLoc.value = { code: '', name_zh: '', name_en: '' }
    await loadLocations()
  } catch (e) { message.value = e.message; messageError.value = true }
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
    message.value = '库位已更新'; messageError.value = false
    editingLocId.value = null
    await loadLocations()
  } catch (e) { message.value = e.message; messageError.value = true }
}

async function removeLocation(id) {
  try {
    await api.deleteLocation(id)
    await loadLocations()
    message.value = '库位已删除'
    messageError.value = false
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}

// --- Categories ---
async function loadCats() {
  try { categories.value = await api.listCategories() } catch (_) { categories.value = [] }
}

async function submitCategory() {
  const nameZh = newCat.value.name_zh.trim()
  if (!nameZh) return
  try {
    await api.createCategory({ name_zh: nameZh, name_en: newCat.value.name_en.trim() })
    message.value = '分类已添加'; messageError.value = false
    newCat.value = { name_zh: '', name_en: '' }
    await loadCats()
  } catch (e) { message.value = e.message; messageError.value = true }
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
    message.value = '分类已更新'; messageError.value = false
    editingCatId.value = null
    await loadCats()
  } catch (e) { message.value = e.message; messageError.value = true }
}

async function removeCategory(id) {
  try {
    await api.deleteCategory(id)
    await loadCats()
    message.value = '分类已删除'
    messageError.value = false
  } catch (e) {
    message.value = e.message
    messageError.value = true
  }
}
</script>

<template>
  <div class="form-grid">
    <div class="card">
      <h3>{{ t('settings.labelSize') }}</h3>
      <label>{{ t('settings.width') }}</label>
      <input v-model.number="settings.label_width_mm" type="number" step="1" />
      <label style="margin-top: 12px">{{ t('settings.height') }}</label>
      <input v-model.number="settings.label_height_mm" type="number" step="1" />
      <p style="color: var(--muted); font-size: 0.9rem">{{ t('settings.printerHint') }}</p>
      <button type="button" style="margin-top: 12px" @click="save">{{ t('common.save') }}</button>
    </div>

    <div v-if="stats" class="card">
      <h3>{{ t('settings.stats') }}</h3>
      <p>{{ t('settings.totalItems') }}: {{ stats.total_items }}</p>
      <p>{{ t('settings.lowStock') }}: {{ stats.low_stock_count }}</p>
      <p>Total qty: {{ stats.total_quantity }}</p>
    </div>

    <div class="card">
      <h3>{{ t('settings.backup') }}</h3>
      <button type="button" class="secondary" @click="backup">{{ t('settings.backup') }}</button>
    </div>

    <div class="card">
      <h3>删除保护密码</h3>
      <p style="color: var(--muted); font-size: 0.9rem; margin: 0 0 12px">
        {{ locale === 'zh-CN' ? '彻底删除物资前需输入此密码。当前状态：' : 'Required before permanently deleting an item. Status: ' }}
        <b :style="{ color: purgePasswordSet ? 'var(--success)' : 'var(--warn)' }">
          {{ purgePasswordSet ? (locale === 'zh-CN' ? '已设置' : 'Set') : (locale === 'zh-CN' ? '未设置' : 'Not set') }}
        </b>
      </p>
      <div class="row-actions" style="margin-top: 0">
        <input
          v-model="purgePasswordNew"
          type="password"
          :placeholder="locale === 'zh-CN' ? '新密码（留空保存则清除）' : 'New password (empty clears it)'"
          style="flex: 1"
          @keydown.enter.prevent="savePurgePassword"
        />
        <button type="button" @click="savePurgePassword">{{ t('common.save') }}</button>
      </div>
      <p style="color: var(--muted); font-size: 0.85rem; margin: 10px 0 0">
        {{ locale === 'zh-CN' ? '密码至少 4 位。未设置时无法彻底删除物资（只能停用）。' : 'Min 4 characters. Without it, items can only be deactivated, not purged.' }}
      </p>
    </div>

    <div class="card">
      <h3>{{ t('operators.title') }}</h3>
      <p style="color: var(--muted); font-size: 0.9rem; margin: 0 0 12px">
        {{ t('operators.hint') }}
      </p>
      <div class="row-actions" style="margin-top: 0">
        <input
          v-model="newOperator"
          :placeholder="t('operators.placeholder')"
          style="flex: 1"
          @keydown.enter.prevent="submitOperator"
        />
        <button type="button" :disabled="adding" @click="submitOperator">
          {{ t('operators.add') }}
        </button>
      </div>

      <ul v-if="operators.length" class="operator-list">
        <li v-for="op in operators" :key="op.id">
          <span>{{ op.display_name }}</span>
          <button
            type="button"
            class="danger operator-del"
            @click="removeOperator(op.id)"
          >{{ t('common.delete') }}</button>
        </li>
      </ul>
      <p v-else class="empty-hint">{{ t('operators.empty') }}</p>
    </div>

    <div class="card">
      <h3>库位管理</h3>
      <div class="row-actions" style="margin-top: 0">
        <input v-model="newLoc.code" placeholder="库位编号 (如 A-01)" style="flex: 2" @keydown.enter="submitLocation" />
        <input v-model="newLoc.name_zh" placeholder="中文名" style="flex: 1" @keydown.enter="submitLocation" />
        <button type="button" @click="submitLocation">添加</button>
      </div>

      <ul v-if="locations.length" class="mgmt-list">
        <li v-for="loc in locations" :key="loc.id">
          <template v-if="editingLocId === loc.id">
            <div class="inline-edit">
              <input v-model="editLoc.code" style="flex:2" @keydown.enter="saveEditLocation(loc.id)" />
              <input v-model="editLoc.name_zh" style="flex:1" @keydown.enter="saveEditLocation(loc.id)" />
              <button type="button" @click="saveEditLocation(loc.id)">保存</button>
              <button type="button" class="secondary" @click="cancelEditLocation">取消</button>
            </div>
          </template>
          <template v-else>
            <span>
              <strong>{{ loc.code }}</strong>
              <template v-if="loc.name_zh"> — {{ loc.name_zh }}</template>
            </span>
            <div style="display:flex;gap:4px">
              <button type="button" class="secondary" style="padding:4px 10px;font-size:0.75rem" @click="startEditLocation(loc)">编辑</button>
              <button type="button" class="danger operator-del" @click="removeLocation(loc.id)">删除</button>
            </div>
          </template>
        </li>
      </ul>
      <p v-else class="empty-hint">暂无库位</p>
    </div>

    <div class="card">
      <h3>分类管理</h3>
      <div class="row-actions" style="margin-top: 0">
        <input v-model="newCat.name_zh" placeholder="分类中文名" style="flex: 1" @keydown.enter="submitCategory" />
        <input v-model="newCat.name_en" placeholder="英文名(可选)" style="flex: 1" @keydown.enter="submitCategory" />
        <button type="button" @click="submitCategory">添加</button>
      </div>

      <ul v-if="categories.length" class="mgmt-list">
        <li v-for="cat in categories" :key="cat.id">
          <template v-if="editingCatId === cat.id">
            <div class="inline-edit">
              <input v-model="editCat.name_zh" @keydown.enter="saveEditCategory(cat.id)" />
              <input v-model="editCat.name_en" @keydown.enter="saveEditCategory(cat.id)" />
              <button type="button" @click="saveEditCategory(cat.id)">保存</button>
              <button type="button" class="secondary" @click="cancelEditCategory">取消</button>
            </div>
          </template>
          <template v-else>
            <span>{{ cat.name_zh }}{{ cat.name_en ? ' / ' + cat.name_en : '' }}</span>
            <div style="display:flex;gap:4px">
              <button type="button" class="secondary" style="padding:4px 10px;font-size:0.75rem" @click="startEditCategory(cat)">编辑</button>
              <button type="button" class="danger operator-del" @click="removeCategory(cat.id)">删除</button>
            </div>
          </template>
        </li>
      </ul>
      <p v-else class="empty-hint">暂无分类</p>
    </div>

    <p v-if="message" class="msg" :class="messageError ? 'error' : 'ok'">{{ message }}</p>
  </div>
</template>

<style scoped>
.operator-list, .mgmt-list {
  list-style: none;
  margin: 16px 0 0;
  padding: 0;
  max-height: 260px;
  overflow-y: auto;
}

.operator-list li, .mgmt-list li {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  font-weight: 600;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.inline-edit {
  display: flex;
  gap: 4px;
  align-items: center;
  width: 100%;
}

.inline-edit input {
  padding: 6px 8px;
  font-size: 0.85rem;
  min-width: 0;
}

.inline-edit button {
  padding: 4px 10px;
  font-size: 0.8rem;
  flex-shrink: 0;
}

.operator-del {
  padding: 4px 12px;
  font-size: 0.8rem;
}

.empty-hint {
  margin: 12px 0 0;
  color: var(--muted);
}
</style>
