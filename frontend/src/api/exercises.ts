import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

export type ExerciseStatistics = {
    in_progress: number
    done: number
    failed: number
    ignored: number
}

export type Exercise = {
    id: string
    type: string
    status: string
    starts_at?: string | null
    started_at?: string | null
    finishes_at?: string | null
    finished_at?: string | null
    telegram_message_id?: number | null
    original_word?: string | null
    original_language?: string | null
    translation_word?: string | null
    translation_language?: string | null
    vocabulary?: {
        translation?: {
            original?: {
                word: string
                language: string
            } | null
            translation?: {
                word: string
                language: string
            } | null
        } | null
    } | null
}

export const exercisesApi = {
    async getStatistics(): Promise<ExerciseStatistics> {
        return apiCall<ExerciseStatistics>('/exercises/statistics', 'GET').then(unwrapBody)
    },

    async getExercises(page: number = 1, pageSize: number = 20): Promise<Paginated<Exercise>> {
        return apiCall<Paginated<Exercise>>('/exercises', 'GET', {
            page,
            page_size: pageSize,
        }).then(unwrapBody)
    },
}
