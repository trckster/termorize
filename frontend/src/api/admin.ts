import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

const API_URL = import.meta.env.VITE_API_URL.replace(/\/$/, '')

export type AdminUser = {
    id: number
    name: string
    username: string
    vocabulary_size: number
    latest_usage: string | null
    deleted_at: string | null
}

export type AdminUsersResponse = {
    data: AdminUser[]
    total: number
}

export type AdminWordPronunciation = {
    id: string
    word_id: string
    word: string
    language: string
    model: string
    voice: string
    mime_type: string
    size_bytes: number
    has_telegram_file: boolean
    created_at: string
}

export const adminApi = {
    async getUsers(): Promise<AdminUsersResponse> {
        return apiCall<AdminUsersResponse>('/admin/users').then(unwrapBody)
    },

    async getWordPronunciations(
        page: number,
        pageSize: number,
        search?: string
    ): Promise<Paginated<AdminWordPronunciation>> {
        return apiCall<Paginated<AdminWordPronunciation>>('/admin/word-pronunciations', 'GET', {
            page,
            page_size: pageSize,
            search,
        }).then(unwrapBody)
    },

    async getWordPronunciationAudio(id: string): Promise<Blob> {
        const response = await fetch(`${API_URL}/admin/word-pronunciations/${encodeURIComponent(id)}/audio`, {
            credentials: 'include',
            headers: { Accept: 'audio/mpeg' },
        })

        if (!response.ok) {
            throw new Error(`Admin pronunciation request failed with status ${response.status}`)
        }

        return response.blob()
    },

    async regenerateWordPronunciation(id: string): Promise<AdminWordPronunciation> {
        return apiCall<AdminWordPronunciation>(`/admin/word-pronunciations/${encodeURIComponent(id)}`, 'DELETE').then(
            unwrapBody
        )
    },
}
