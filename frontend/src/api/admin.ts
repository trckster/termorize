import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'
import type { Word, VocabularyItem } from '@/api/vocabulary'
import type { PartOfSpeech } from '@/lib/partOfSpeech'

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

export type CategorizationStats = {
    total: number
    categorized: number
    unknown: number
    pending: number
    worker: {
        active: boolean
        queued: number
        processed: number
        failed: number
        scan_failed: boolean
    } | null
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
    translation: string
    translation_language: string
    model: string
    description: string
    created_at: string
}
export type DescriptionModel = { id: string; name: string; tier: 'basic' | 'medium' | 'smart' }
export type DescriptionPreview = {
    translation_word_id: string | null
    translation: string
    translation_language: string
    model: string
    description: string
    original_description: string
}

export type Dictionary = {
    id: string
    name: string
    edition: string
    url: string
    license: string
    attribution: string
    download_url: string
}

export type DictionaryImportJob = {
    id: string
    dictionary_id: string
    source_name: string
    edition: string
    download_url: string
    status: 'queued' | 'downloading' | 'importing' | 'succeeded' | 'failed' | 'interrupted'
    processed: number
    inserted: number
    classified: number
    skipped: number
    failed: number
    downloaded_bytes: number
    total_bytes: number | null
    error: string
    record_errors: string[]
    created_at: string
    updated_at: string
    started_at: string | null
    finished_at: string | null
}

export const adminApi = {
    async getCategorizationStats(): Promise<CategorizationStats> {
        return apiCall<CategorizationStats>('/admin/categorization/stats').then(unwrapBody)
    },
    async restartCategorization(): Promise<void> {
        await apiCall('/admin/categorization/restart', 'POST')
    },
    async getCategoryWords(page = 1, search = ''): Promise<Paginated<Word>> {
        return apiCall<Paginated<Word>>('/admin/categorization/words', 'GET', { page, page_size: 20, search }).then(
            unwrapBody
        )
    },
    async getUnknownWords(page = 1): Promise<Paginated<Word>> {
        return apiCall<Paginated<Word>>('/admin/categorization/unknown', 'GET', { page, page_size: 20 }).then(
            unwrapBody
        )
    },
    async getMismatchedVocabulary(page = 1): Promise<Paginated<VocabularyItem>> {
        return apiCall<Paginated<VocabularyItem>>('/admin/categorization/mismatches', 'GET', {
            page,
            page_size: 20,
        }).then(unwrapBody)
    },
    async setWordPartOfSpeech(id: string, partOfSpeech: PartOfSpeech): Promise<Word> {
        return apiCall<Word>(`/admin/words/${encodeURIComponent(id)}/part-of-speech`, 'PUT', {
            part_of_speech: partOfSpeech,
        }).then(unwrapBody)
    },
    async getDictionaries(): Promise<Dictionary[]> {
        return apiCall<Dictionary[]>('/admin/dictionaries').then(unwrapBody)
    },
    async getDictionaryImports(id: string, page = 1): Promise<Paginated<DictionaryImportJob>> {
        return apiCall<Paginated<DictionaryImportJob>>(`/admin/dictionaries/${encodeURIComponent(id)}/imports`, 'GET', {
            page,
        }).then(unwrapBody)
    },
    async startDictionaryImport(id: string): Promise<DictionaryImportJob> {
        return apiCall<DictionaryImportJob>(`/admin/dictionaries/${encodeURIComponent(id)}/imports`, 'POST').then(
            unwrapBody
        )
    },
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
    async approveWordDescription(id: string, preview: DescriptionPreview): Promise<unknown> {
        return apiCall(`/admin/word-descriptions/${encodeURIComponent(id)}/approve`, 'POST', {
            model: preview.model,
            description: preview.description,
            translation_word_id: preview.translation_word_id,
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
