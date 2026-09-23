import apiCall, { unwrapBody } from '@/api/index.ts'

export type DailyIdiom = {
    date: string
    language: string
    idiom: { id: string; word_id: string; word: string } | null
}

export const dailyIdiomApi = {
    get(language: string): Promise<DailyIdiom> {
        return apiCall<DailyIdiom>('/daily-idiom', 'GET', { language }).then(unwrapBody)
    },
    describe(id: string): Promise<{ description: string }> {
        return apiCall<{ description: string }>(`/daily-idiom/${encodeURIComponent(id)}/description`).then(unwrapBody)
    },
}
