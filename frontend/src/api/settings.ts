import apiCall from './index.ts'
import type { User, UserSettings } from '@/api/auth.ts'

export interface Settings {
    languages: string[]
}

export interface UpdateSettingsPayload extends UserSettings {}

export const settingsApi = {
    async getSettings(): Promise<Settings> {
        const response = await apiCall<Settings>('/settings', 'GET')
        return response.body
    },

    async updateSettings(payload: UpdateSettingsPayload): Promise<User> {
        const response = await apiCall<User>('/settings', 'PUT', payload)
        return response.body
    },
}
