import apiCall, { unwrapBody } from '@/api/index.ts'

export type ExerciseStatistics = {
    in_progress: number
    done: number
    failed: number
    ignored: number
}

export const exercisesApi = {
    async getStatistics(): Promise<ExerciseStatistics> {
        return apiCall<ExerciseStatistics>('/exercises/statistics', 'GET').then(unwrapBody)
    },
}
