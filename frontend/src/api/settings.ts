import apiCall, { unwrapBody } from './index.ts'
import type { User, UserSettings } from '@/api/auth.ts'

export interface Settings {
    languages: string[]
}

export interface UpdateSettingsPayload extends UserSettings {}

export const settingsApi = {
    async getSettings(): Promise<Settings> {
        return apiCall<Settings>('/settings', 'GET').then(unwrapBody)
    },

    async updateSettings(payload: UpdateSettingsPayload): Promise<User> {
        return apiCall<User>('/settings', 'PUT', payload).then(unwrapBody)
    },

    async removeIgnoredAudioLanguage(language: string): Promise<User> {
        return apiCall<User>(`/settings/ignored-audio-languages/${encodeURIComponent(language)}`, 'DELETE').then(
            unwrapBody
        )
    },

    async removeIgnoredDescriptionLanguage(language: string): Promise<User> {
        return apiCall<User>(`/settings/ignored-description-languages/${encodeURIComponent(language)}`, 'DELETE').then(
            unwrapBody
        )
    },
}
