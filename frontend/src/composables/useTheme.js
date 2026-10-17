import { computed, ref } from 'vue'

// Stored preference: 'auto' follows the operating system, otherwise an
// explicit 'light' / 'dark'. The concrete theme is written to
// <html data-theme="..."> which is what the stylesheet keys off.
const PREF_KEY = 'theme'
const ORDER = ['auto', 'light', 'dark']

const preference = ref(readPreference())
const systemPrefersLight = ref(
  typeof window !== 'undefined' &&
    window.matchMedia &&
    window.matchMedia('(prefers-color-scheme: light)').matches,
)

function readPreference() {
  try {
    const saved = localStorage.getItem(PREF_KEY)
    if (ORDER.includes(saved)) return saved
  } catch (_) {}
  return 'auto'
}

function resolved(pref) {
  if (pref === 'auto') return systemPrefersLight.value ? 'light' : 'dark'
  return pref
}

function apply(pref) {
  preference.value = pref
  document.documentElement.dataset.theme = resolved(pref)
  try {
    localStorage.setItem(PREF_KEY, pref)
  } catch (_) {}
}

if (typeof window !== 'undefined' && window.matchMedia) {
  const mq = window.matchMedia('(prefers-color-scheme: light)')
  const onChange = (e) => {
    systemPrefersLight.value = e.matches
    if (preference.value === 'auto') apply('auto')
  }
  // addEventListener is unavailable in older Safari; fall back to the
  // deprecated API so following the OS keeps working there.
  if (mq.addEventListener) mq.addEventListener('change', onChange)
  else if (mq.addListener) mq.addListener(onChange)
}

export function useTheme() {
  function cycleTheme() {
    const next = ORDER[(ORDER.indexOf(preference.value) + 1) % ORDER.length]
    apply(next)
  }

  function setTheme(pref) {
    if (ORDER.includes(pref)) apply(pref)
  }

  return {
    preference,
    theme: computed(() => resolved(preference.value)),
    cycleTheme,
    setTheme,
  }
}
