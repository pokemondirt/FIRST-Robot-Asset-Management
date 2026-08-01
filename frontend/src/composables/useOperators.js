import { ref } from 'vue'
import { api } from '../api/client'

const operators = ref([])
const loading = ref(false)
const error = ref('')

export function useOperators() {
  async function loadOperators() {
    loading.value = true
    error.value = ''
    try {
      operators.value = await api.listOperators()
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  async function addOperator(name) {
    const trimmed = (name || '').trim()
    if (!trimmed) {
      throw new Error('EMPTY_NAME')
    }
    const op = await api.createOperator(trimmed)
    await loadOperators()
    return op
  }

  return { operators, loading, error, loadOperators, addOperator }
}
