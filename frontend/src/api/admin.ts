import apiCall, { unwrapBody } from '@/api/index.ts'

export type AdminUser = {
    id: number
    name: string
    username: string
    vocabulary_size: number
    openrouter_cost: number
    latest_usage: string
}

export type AdminUsersResponse = {
    data: AdminUser[]
    total: number
}

export const adminApi = {
    async getUsers(): Promise<AdminUsersResponse> {
        return apiCall<AdminUsersResponse>('/admin/users').then(unwrapBody)
    },
}
