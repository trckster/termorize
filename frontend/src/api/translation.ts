import apiCall, { unwrapBody } from '@/api/index.ts'

export type TranslateRequest = {
    from_word: string
    from_language: string
    to_language: string
}

export type TranslateResponse = {
    id: string
    translation: string
    source: 'user' | 'dictionary' | 'google'
}

export const translationApi = {
    async translate(request: TranslateRequest): Promise<TranslateResponse> {
        return apiCall<TranslateResponse>('/translate', 'POST', request).then(unwrapBody)
    },

    async addVocabularyByTranslation(translationId: string): Promise<void> {
        await apiCall('/vocabulary/translation', 'POST', {
            translation_id: translationId,
        })
    },
}
