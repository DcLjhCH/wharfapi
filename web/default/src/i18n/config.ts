/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import fr from './locales/fr.json'
import ja from './locales/ja.json'
import ru from './locales/ru.json'
import vi from './locales/vi.json'
import zh from './locales/zh.json'

export const resources = {
  en,
  zh,
  fr,
  ru,
  ja,
  vi,
} as const

const DEFAULT_LANGUAGE = 'zh'
const LANGUAGE_STORAGE_KEY = 'i18nextLng'
const LANGUAGE_MIGRATION_KEY = 'wharfapi.defaultLanguageMigrated'

// The new frontend previously fell back to English and cached that result.
// Apply Chinese once for that cached default; an explicit choice made after
// this migration is left untouched.
function useChineseAsDefaultLanguage() {
  if (typeof window === 'undefined') return
  try {
    const stored = window.localStorage.getItem(LANGUAGE_STORAGE_KEY)
    const normalized = stored?.trim().replace(/_/g, '-').toLowerCase() ?? ''
    if (normalized.startsWith('zh') && stored !== DEFAULT_LANGUAGE) {
      window.localStorage.setItem(LANGUAGE_STORAGE_KEY, DEFAULT_LANGUAGE)
      return
    }
    if (window.localStorage.getItem(LANGUAGE_MIGRATION_KEY)) return
    if (!normalized || normalized === 'en' || normalized.startsWith('en-')) {
      window.localStorage.setItem(LANGUAGE_STORAGE_KEY, DEFAULT_LANGUAGE)
    }
    window.localStorage.setItem(LANGUAGE_MIGRATION_KEY, DEFAULT_LANGUAGE)
  } catch {
    // localStorage can be unavailable; fallbackLng still applies.
  }
}

useChineseAsDefaultLanguage()

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: DEFAULT_LANGUAGE,
    supportedLngs: ['en', 'zh', 'fr', 'ru', 'ja', 'vi'],
    nonExplicitSupportedLngs: true,
    load: 'languageOnly', // Convert zh-CN -> zh
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
      convertDetectedLanguage: (lng: string) =>
        lng.toLowerCase().startsWith('zh') ? DEFAULT_LANGUAGE : lng,
    },
  })

export default i18n
