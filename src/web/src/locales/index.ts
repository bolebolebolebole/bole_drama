import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'

// 固定使用中文
const i18n = createI18n({
  legacy: false, // 使用 Composition API 模式
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN
  }
})

export default i18n

// 导出获取当前语言的函数（始终返回中文）
export const getCurrentLanguage = () => {
  return 'zh-CN'
}
