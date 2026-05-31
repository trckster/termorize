import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

export type CollectionWord = {
    id: string
    language: string
    word: string
}

export type CollectionTranslationSource = 'user' | 'dictionary' | 'google' | 'llm'

export type CollectionTranslation = {
    id: string
    source: CollectionTranslationSource
    original: CollectionWord
    translation: CollectionWord
}

export type CollectionSummary = {
    id: string
    title: string
    is_admin: boolean
    is_owner: boolean
    is_published: boolean
    owner_username?: string
    languages: string[]
    translation_count: number
    user_add_count: number
    created_at: string
}

export type CollectionDetail = CollectionSummary & {
    invite_token?: string
    owner_username?: string
    translations: CollectionTranslation[]
}

export type AddCollectionToVocabularyResult = {
    added: number
    skipped: number
    total: number
    user_add_count: number
}

export const collectionsApi = {
    async getCollections(
        page: number = 1,
        pageSize: number = 50,
        search?: string
    ): Promise<Paginated<CollectionSummary>> {
        return apiCall<Paginated<CollectionSummary>>('/collections', 'GET', {
            page,
            page_size: pageSize,
            search,
        }).then(unwrapBody)
    },

    async getCollection(id: string): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collections/${id}`).then(unwrapBody)
    },

    async createCollection(title: string, isAdmin: boolean = false): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>('/collections', 'POST', {
            title,
            is_admin: isAdmin,
        }).then(unwrapBody)
    },

    async deleteCollection(id: string): Promise<void> {
        await apiCall<void>(`/collections/${id}`, 'DELETE')
    },

    async addTranslation(
        id: string,
        original: string,
        translation: string,
        originalLanguage: string,
        translationLanguage: string
    ): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collections/${id}/translations`, 'POST', {
            original,
            translation,
            original_language: originalLanguage,
            translation_language: translationLanguage,
        }).then(unwrapBody)
    },

    async removeTranslation(id: string, translationId: string): Promise<void> {
        await apiCall<void>(`/collections/${id}/translations/${translationId}`, 'DELETE')
    },

    async reorderTranslations(id: string, translationIds: string[]): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collections/${id}/translations/order`, 'PUT', {
            translation_ids: translationIds,
        }).then(unwrapBody)
    },

    async addToVocabulary(id: string, translationIds?: string[]): Promise<AddCollectionToVocabularyResult> {
        return apiCall<AddCollectionToVocabularyResult>(
            `/collections/${id}/add-to-vocabulary`,
            'POST',
            translationIds ? { translation_ids: translationIds } : undefined
        ).then(unwrapBody)
    },

    async joinByToken(token: string): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collection-invites/${token}`, 'POST').then(unwrapBody)
    },

    async generate(prompt: string): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>('/collection-generate', 'POST', { prompt }).then(unwrapBody)
    },

    async setPublished(id: string, published: boolean): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collections/${id}/publish`, 'POST', { is_published: published }).then(unwrapBody)
    },

    async updateTitle(id: string, title: string): Promise<CollectionDetail> {
        return apiCall<CollectionDetail>(`/collections/${id}`, 'PUT', { title }).then(unwrapBody)
    },
}
