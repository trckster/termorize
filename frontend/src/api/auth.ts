import apiCall, { unwrapBody } from '@/api/index.ts'

export interface TelegramLoginStartResponse {
    auth_url: string
}

export interface TelegramLoginCallbackPayload {
    code: string
    state: string
}

export interface User {
    id: number
    username: string
    name: string
    settings: UserSettings
    created_at: string
}

export interface UserSettings {
    system_language: string
    main_learning_language: string
    translation_source_language: string
    translation_target_language: string
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
    async startTelegramLogin(): Promise<string> {
        return apiCall<TelegramLoginStartResponse>('/telegram/login/start', 'POST').then((response) => response.body.auth_url)
    },

    async completeTelegramLogin(payload: TelegramLoginCallbackPayload): Promise<User | null> {
        return apiCall<User>('/telegram/login/callback', 'POST', payload, {
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
