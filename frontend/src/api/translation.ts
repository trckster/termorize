import apiCall from '@/api/index.ts'

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
        const response = await apiCall<TranslateResponse>('/translate', 'POST', request)
        return response.body
    },
}
