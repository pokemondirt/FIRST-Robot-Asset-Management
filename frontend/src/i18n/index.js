import { createI18n } from 'vue-i18n'
import zh from './locales/zh-CN.json'
import en from './locales/en-US.json'

const saved = localStorage.getItem('locale') || 'zh-CN'

export default createI18n({
  legacy: false,
  locale: saved,
  fallbackLocale: 'en-US',
  messages: { 'zh-CN': zh, 'en-US': en },
})
