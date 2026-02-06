import apiCall from '@/api/index.ts'

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
    photo_url: string
    created_at: string
}

export const authApi = {
    async login(authData: TelegramAuthData): Promise<User | null> {
        const response = await apiCall<User>('/telegram/login', 'POST', authData)

        return response.body
    },

    async getCurrentUser(): Promise<User | null> {
        return await apiCall<User>('/me').then((r) => r.body)
    },

    async logout(): Promise<void> {
        await apiCall<void>('/logout', 'POST')
    },
}
