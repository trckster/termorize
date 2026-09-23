import apiCall, { unwrapBody } from '@/api/index.ts'

export type DailyIdiom = {
    date: string
    language: string
    idiom: { id: string; word_id: string; word: string } | null
}

export const dailyIdiomApi = {
    get(): Promise<DailyIdiom> {
        return apiCall<DailyIdiom>('/daily-idiom').then(unwrapBody)
    },
    describe(id: string): Promise<{ description: string }> {
        return apiCall<{ description: string }>(`/daily-idiom/${encodeURIComponent(id)}/description`).then(unwrapBody)
    },
}
