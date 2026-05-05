import { create } from 'zustand'
import { type Lang, t as tFn } from '../i18n'

export type Theme = 'dark' | 'light' | 'slate' | 'ocean' | 'sunset'

export const themeNames: Record<Theme, string> = {
  dark: '深邃黑',
  light: '明亮白',
  slate: '石板灰',
  ocean: '海洋蓝',
  sunset: '日落橙',
}

interface SettingsState {
  lang: Lang
  theme: Theme
  setLang: (l: Lang) => void
  setTheme: (t: Theme) => void
  initSettings: () => void
}

export const useSettingsStore = create<SettingsState>((set) => {
  return {
    lang: 'zh',
    theme: 'light',
    setLang: (lang) => {
      set({ lang })
      localStorage.setItem('oiwest-settings', JSON.stringify({ ...useSettingsStore.getState(), lang }))
    },
    setTheme: (theme) => {
      set({ theme })
      document.documentElement.setAttribute('data-theme', theme)
      localStorage.setItem('oiwest-settings', JSON.stringify({ ...useSettingsStore.getState(), theme }))
    },
    initSettings: () => {
      document.documentElement.setAttribute('data-theme', 'light')
    },
  }
})

export function useT() {
  const lang = useSettingsStore((s) => s.lang)
  return { lang, t: (key: string, vars?: Record<string, string | number>) => tFn(lang, key, vars) }
}
