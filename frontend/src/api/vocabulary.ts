import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

type VocabularyItemProgress = {
    knowledge: number
    type: 'translation'
}

type Word = {
    id: string
    language: string
    word: string
    created_at: string
}

type TranslationSource = 'user' | 'google'
type Translation = {
    id: string
    source: TranslationSource
    user_id: string | null
    original: Word
    translation: Word
    created_at: string
}

export type VocabularyItem = {
    id: string

    translation_id: string
    translation: Translation

    progress: VocabularyItemProgress[]
    created_at: string
    mastered_at: string | null
}

export const vocabularyApi = {
    async getVocabulary(page: number = 1, pageSize: number = 100, search?: string): Promise<Paginated<VocabularyItem>> {
        return apiCall<Paginated<VocabularyItem>>('/vocabulary', 'GET', {
            page,
            page_size: pageSize,
            search,
        }).then(unwrapBody)
    },

    async deleteVocabulary(id: string): Promise<void> {
        await apiCall<void>(`/vocabulary/${id}`, 'DELETE')
    },

    async addVocabulary(
        original: string,
        translation: string,
        originalLanguage: string,
        translationLanguage: string
    ): Promise<VocabularyItem> {
        return apiCall<VocabularyItem>('/vocabulary', 'POST', {
            original,
            translation,
            original_language: originalLanguage,
            translation_language: translationLanguage,
        }).then(unwrapBody)
    },
}
