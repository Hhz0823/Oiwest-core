import zh from './zh'
import en from './en'
import ja from './ja'

export type Lang = 'zh' | 'en' | 'ja'

const translations: Record<Lang, Record<string, string>> = { zh, en, ja }

export function t(lang: Lang, key: string, vars?: Record<string, string | number>): string {
  const dict = translations[lang] || translations.en
  let text = dict[key]
  if (text === undefined) {
    text = translations.en[key] || key
  }
  if (vars) {
    Object.entries(vars).forEach(([k, v]) => {
      text = text.replace(`{${k}}`, String(v))
    })
  }
  return text
}

export const langNames: Record<Lang, string> = {
  zh: '中文',
  en: 'English',
  ja: '日本語',
}

export const langFlags: Record<Lang, string> = {
  zh: '🇨🇳',
  en: '🇺🇸',
  ja: '🇯🇵',
}
