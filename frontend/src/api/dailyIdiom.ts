import type { TranslateResponse } from '@/api/translation'
import apiCall, { unwrapBody } from '@/api/index.ts'

export type DailyIdiom = {
    date: string
    language: string
    idiom: { id: string; word_id: string; word: string } | null
}

export const dailyIdiomApi = {
    translate(id: string, toLanguage: string): Promise<TranslateResponse> {
        return apiCall<TranslateResponse>(`/daily-idiom/${encodeURIComponent(id)}/translate`, 'POST', {
            to_language: toLanguage,
        }).then(unwrapBody)
    },
    get(): Promise<DailyIdiom> {
        return apiCall<DailyIdiom>('/daily-idiom').then(unwrapBody)
    },
    describe(id: string): Promise<{ description: string }> {
        return apiCall<{ description: string }>(`/daily-idiom/${encodeURIComponent(id)}/description`).then(unwrapBody)
    },
}
