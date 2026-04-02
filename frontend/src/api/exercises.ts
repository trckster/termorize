import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

export type ExerciseStatistics = {
    in_progress: number
    done: number
    failed: number
    ignored: number
}

type ExerciseWord = {
    word: string
    language: string
}

type ExerciseTranslation = {
    original?: ExerciseWord | null
    translation?: ExerciseWord | null
}

type ExerciseVocabulary = {
    id: string
    translation?: ExerciseTranslation | null
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
        translation?: ExerciseTranslation | null
    } | null
    vocabularies?: ExerciseVocabulary[]
}

export type RandomExercise = {
    exercise_id: string
    type: 'basic/direct' | 'basic/reversed'
    question_word: string
    language: string
    answer_language: string
}

export type VerifyResult = {
    result: 'correct' | 'almost' | 'wrong'
    correct_answer: string
    knowledge: number
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

    async getExercisesByIds(ids: string[]): Promise<Exercise[]> {
        return apiCall<Exercise[]>('/exercises/by-ids', 'GET', { ids: ids.join(',') }).then(unwrapBody)
    },

    async getRandomExercise(): Promise<RandomExercise> {
        return apiCall<RandomExercise>('/exercises/random', 'POST').then(unwrapBody)
    },

    async verifyExercise(exerciseId: string, answer: string): Promise<VerifyResult> {
        return apiCall<VerifyResult>(`/exercises/${exerciseId}/verify`, 'POST', { answer }).then(unwrapBody)
    },
}
