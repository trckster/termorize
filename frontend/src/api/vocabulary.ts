import apiCall from '@/api/index.ts'
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
    word_1: Word
    word_2: Word
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
    async getVocabulary(page: number = 1, pageSize: number = 100): Promise<Paginated<VocabularyItem>> {
        const response = await apiCall<Paginated<VocabularyItem>>('/vocabulary', 'GET', {
            page,
            page_size: pageSize,
        })

        return response.body
    },

    async deleteVocabulary(id: string): Promise<void> {
        await apiCall<void>(`/vocabulary/${id}`, 'DELETE')
    },

    async addVocabulary(word1: string, word2: string, language1: string, language2: string): Promise<VocabularyItem> {
        const response = await apiCall<VocabularyItem>('/vocabulary', 'POST', {
            word_1: word1,
            word_2: word2,
            language_1: language1,
            language_2: language2,
        })
        return response.body
    },
}
