import apiCall, { unwrapBody } from '@/api/index.ts'

export interface TelegramAuthData {
    id: number
    auth_date: number
    username: string
    first_name: string
    last_name: string
    photo_url: string
    hash: string
}

export interface User {
    id: number
    username: string
    name: string
    settings: UserSettings
    created_at: string
}

export interface UserSettings {
    native_language: string
    main_learning_language: string
    time_zone: string
    telegram: UserTelegramSettings
}

export interface UserTelegramSettings {
    bot_enabled: boolean
    daily_questions_enabled: boolean
    daily_questions_count: number
    daily_questions_schedule: UserTelegramScheduleItem[]
}

export interface UserTelegramScheduleItem {
    from: string
    to: string
}

export const authApi = {
    async login(authData: TelegramAuthData): Promise<User | null> {
        return apiCall<User>('/telegram/login', 'POST', authData, {
            'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
        }).then(unwrapBody)
    },

    async getCurrentUser(): Promise<User | null> {
        return apiCall<User>('/me').then(unwrapBody)
    },

    async logout(): Promise<void> {
        await apiCall<void>('/logout', 'POST')
    },
}
