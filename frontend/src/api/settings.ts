import apiCall from './index.ts'

export interface Settings {
    languages: string[]
}

export const settingsApi = {
    async getSettings(): Promise<Settings> {
        const response = await apiCall<Settings>('/settings', 'GET')
        return response.body
    },
}
