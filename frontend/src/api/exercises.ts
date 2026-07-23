import apiCall, { unwrapBody } from '@/api/index.ts'
import type { Paginated } from '@/api/pagination.ts'

export type ExerciseStatistics = {
    in_progress: number
    done: number
    failed: number
    ignored: number
    exercise_activity: ExerciseDailyActivity[]
    vocabulary_activity: VocabularyDailyActivity[]
}

export type ExerciseDailyActivity = {
    date: string
    completed: number
    failed: number
}

export type VocabularyDailyActivity = {
    date: string
    count: number
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
    exercise_result?: 'correct' | 'almost' | 'wrong' | 'ignored' | null
    result_reason?: string | null
    progress_delta?: number | null
    knowledge_after?: number | null
    answered_at?: string | null
    is_correct?: boolean
    position?: number
}

export type ExerciseMatchCard = {
    id: string
    vocabulary_id: string
    word: string
    language: string
    side: 'original' | 'translation'
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
    type:
        | 'basic/direct'
        | 'basic/reversed'
        | 'choice/direct'
        | 'choice/reversed'
        | 'characters/direct'
        | 'characters/reversed'
        | 'match/pairs'
    question_word: string
    language: string
    answer_language: string
    options: string[]
    cards?: ExerciseMatchCard[]
}

export type VerifyResult = {
    result: 'correct' | 'almost' | 'wrong'
    correct_answer: string
    knowledge: number
    progress_delta: number
}

export type MatchPairResult = 'correct' | 'almost' | 'wrong'

export type MatchPairAttempt = {
    first_card_id: string
    second_card_id: string
}

export type MatchPairsCompleteResult = {
    status: 'completed' | 'failed'
    results: ExerciseVocabulary[]
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

    async ignoreExercise(exerciseId: string): Promise<void> {
        await apiCall(`/exercises/${exerciseId}/ignore`, 'POST')
    },

    async completeMatchPairsExercise(
        exerciseId: string,
        attempts: MatchPairAttempt[]
    ): Promise<MatchPairsCompleteResult> {
        return apiCall<MatchPairsCompleteResult>(`/exercises/${exerciseId}/match-pairs/complete`, 'POST', {
            attempts,
        }).then(unwrapBody)
    },
}
