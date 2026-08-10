import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { settingsApi, type Settings } from '@/api/settings.ts'

interface LanguageOption {
    code: string
    name: string
    emoji: string
}

const languageNames: Record<string, string> = {
    en: 'English',
    ru: 'Russian',
    it: 'Italian',
    de: 'German',
    es: 'Spanish',
    fr: 'French',
    pl: 'Polish',
    tr: 'Turkish',
    pt: 'Portuguese',
    uk: 'Ukrainian',
}

const russianLanguageNames: Record<string, string> = {
    en: 'Английский',
    ru: 'Русский',
    it: 'Итальянский',
    de: 'Немецкий',
    es: 'Испанский',
    fr: 'Французский',
    pl: 'Польский',
    tr: 'Турецкий',
    pt: 'Португальский',
    uk: 'Украинский',
}

const languageEmojis: Record<string, string> = {
    en: '🇬🇧',
    ru: '🇷🇺',
    it: '🇮🇹',
    de: '🇩🇪',
    es: '🇪🇸',
    fr: '🇫🇷',
    pl: '🇵🇱',
    tr: '🇹🇷',
    pt: '🇵🇹',
    uk: '🇺🇦',
}

const fallbackFlag = '🏳'

const getLanguageName = (code: string, locale = 'en') =>
    (locale === 'ru' ? russianLanguageNames[code] : languageNames[code]) || code.toUpperCase()
const getLanguageFlag = (code: string) => languageEmojis[code] || fallbackFlag

export const useSettingsStore = defineStore('settings', () => {
    const settings = ref<Settings | null>(null)

    const languages = computed<string[]>(() => {
        return settings.value?.languages || []
    })

    const languageOptions = computed<LanguageOption[]>(() => {
        return languages.value.map((code) => ({
            code,
            name: getLanguageName(code),
            emoji: getLanguageFlag(code),
        }))
    })

    const getFlag = (languageCode: string) => {
        return getLanguageFlag(languageCode)
    }

    const getLanguageNameForCode = (languageCode: string, locale = 'en') => {
        return getLanguageName(languageCode, locale)
    }

    const fetchSettings = async () => {
        try {
            settings.value = await settingsApi.getSettings()
        } catch (error) {
            console.error('Failed to fetch settings:', error)
        }
    }

    return {
        settings,
        languageOptions,
        fetchSettings,
        getFlag,
        getLanguageName: getLanguageNameForCode,
    }
})
