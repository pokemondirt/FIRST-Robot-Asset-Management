const BASE = '/api'

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = await res.json()
      const d = body.detail
      detail =
        (typeof d === 'object' && (d?.message || d?.detail)) ||
        (typeof d === 'string' ? d : null) ||
        body.message ||
        JSON.stringify(body)
    } catch (_) {}
    throw new Error(typeof detail === 'string' ? detail : JSON.stringify(detail))
  }
  if (res.status === 204) return null
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/json')) return res.json()
  return res.blob()
}

export const api = {
  health: () => request('/health'),
  lookup: (code) => request(`/items/lookup/${encodeURIComponent(code)}`),
  listItems: (params) => {
    const q = new URLSearchParams(params).toString()
    return request(`/items?${q}`)
  },
  getItem: (id) => request(`/items/${id}`),
  createItem: (body) => request('/items', { method: 'POST', body: JSON.stringify(body) }),
  updateItem: (id, body) =>
    request(`/items/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteItem: (id) => request(`/items/${id}`, { method: 'DELETE' }),
  purgeItem: (id, password) =>
    request(`/items/${id}/purge`, { method: 'DELETE', body: JSON.stringify({ password }) }),
  nextBarcode: (program, trackMode) =>
    request(`/items/next-barcode?program=${program}&track_mode=${trackMode}`, {
      method: 'POST',
    }),
  createTransaction: (body) =>
    request('/transactions', { method: 'POST', body: JSON.stringify(body) }),
  batchCreateTransactions: (body) =>
    request('/transactions/batch', { method: 'POST', body: JSON.stringify(body) }),
  listTransactions: (params) => {
    const q = new URLSearchParams(params).toString()
    return request(`/transactions?${q}`)
  },
  listOperators: () => request('/operators'),
  createOperator: (name) =>
    request('/operators', { method: 'POST', body: JSON.stringify({ display_name: name }) }),
  deleteOperator: (id) => request(`/operators/${id}`, { method: 'DELETE' }),
  getSettings: () => request('/settings'),
  updateSettings: (body) =>
    request('/settings', { method: 'PATCH', body: JSON.stringify(body) }),
  getStats: () => request('/settings/stats'),
  getDashboard: () => request('/dashboard'),
  backup: () => request('/data/backup', { method: 'POST' }),
  importFile: async (file) => {
    const fd = new FormData()
    fd.append('file', file)
    const res = await fetch(`${BASE}/data/import`, { method: 'POST', body: fd })
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  },
  exportCsvUrl: () => `${BASE}/data/export/csv`,
  templateUrl: () => `${BASE}/data/export/template.csv`,

  // Categories
  listCategories: () => request('/categories'),
  createCategory: (body) =>
    request('/categories', { method: 'POST', body: JSON.stringify(body) }),
  updateCategory: (id, body) =>
    request(`/categories/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteCategory: (id) => request(`/categories/${id}`, { method: 'DELETE' }),

  // Locations
  listLocations: () => request('/locations'),
  createLocation: (body) =>
    request('/locations', { method: 'POST', body: JSON.stringify(body) }),
  updateLocation: (id, body) =>
    request(`/locations/${id}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteLocation: (id) => request(`/locations/${id}`, { method: 'DELETE' }),
}

export function playBeep(success = true) {
  try {
    const ctx = new (window.AudioContext || window.webkitAudioContext)()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.frequency.value = success ? 880 : 220
    gain.gain.value = 0.15
    osc.start()
    osc.stop(ctx.currentTime + (success ? 0.08 : 0.25))
  } catch (_) {}
}
