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
}

const languageEmojis: Record<string, string> = {
    en: '🇬🇧',
    ru: '🇷🇺',
    it: '🇮🇹',
    de: '🇩🇪',
}

const fallbackFlag = '🏳'

const getLanguageName = (code: string) => languageNames[code] || code.toUpperCase()
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
    }
})
