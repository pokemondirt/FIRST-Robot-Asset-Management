<script setup>
import { onUnmounted, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  width: { type: String, default: '380px' },
})
const emit = defineEmits(['update:modelValue'])

function close() {
  emit('update:modelValue', false)
}

// A full-height overlay must not let the page behind it scroll.
watch(
  () => props.modelValue,
  (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
  },
)

onUnmounted(() => {
  document.body.style.overflow = ''
})

function onKey(e) {
  if (e.key === 'Escape' && props.modelValue) close()
}
if (typeof window !== 'undefined') window.addEventListener('keydown', onKey)
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div v-if="modelValue" class="drawer-mask" @click.self="close">
    <aside class="drawer" :style="{ width }">
      <header class="drawer-head">
        <h3>{{ title }}</h3>
        <button type="button" class="icon-btn" :title="'Esc'" @click="close">✕</button>
      </header>
      <div class="drawer-body">
        <slot />
      </div>
    </aside>
  </div>
</template>
