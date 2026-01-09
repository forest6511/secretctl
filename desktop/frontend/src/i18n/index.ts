import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import ja from './locales/ja.json'

// Get system language or default to English
const getDefaultLanguage = () => {
  const lang = navigator.language.split('-')[0]
  return lang === 'ja' ? 'ja' : 'en'
}

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      ja: { translation: ja },
    },
    lng: getDefaultLanguage(),
    fallbackLng: 'en',
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // React already escapes values
    },
  })

export default i18n
