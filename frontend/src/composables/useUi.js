import { ref } from 'vue'

// Small pieces of app-wide UI state. Kept in a module-level ref so any page can
// open the same drawer that App.vue renders, without prop drilling or events.
const guideOpen = ref(false)

export function useUi() {
  function openGuide() {
    guideOpen.value = true
  }

  return { guideOpen, openGuide }
}
