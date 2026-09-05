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

export type AdminWordDescription = {
    id: string
    word_id: string
    word: string
    language: string
    model: string
    description: string
    created_at: string
    approved_at: string | null
}
export type DescriptionModel = { id: string; name: string; tier: 'basic' | 'medium' | 'smart' }
export type DescriptionPreview = { id: string; model: string; description: string; created_at: string }

export const adminApi = {
    async getDescriptionModels(): Promise<DescriptionModel[]> {
        return apiCall<DescriptionModel[]>('/admin/description-models').then(unwrapBody)
    },
    async getWordDescriptions(page: number, search?: string): Promise<Paginated<AdminWordDescription>> {
        return apiCall<Paginated<AdminWordDescription>>('/admin/word-descriptions', 'GET', {
            page,
            page_size: 20,
            search,
        }).then(unwrapBody)
    },
    async previewWordDescription(id: string, model: string): Promise<DescriptionPreview> {
        return apiCall<DescriptionPreview>(`/admin/word-descriptions/${encodeURIComponent(id)}/preview`, 'POST', {
            model,
        }).then(unwrapBody)
    },
    async approveWordDescription(id: string, previewId: string): Promise<unknown> {
        return apiCall(`/admin/word-descriptions/${encodeURIComponent(id)}/approve`, 'POST', {
            preview_id: previewId,
        }).then(unwrapBody)
    },
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
