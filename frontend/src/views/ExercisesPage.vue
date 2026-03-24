<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { exercisesApi, type Exercise } from '@/api/exercises.ts'
import type { PaginationData } from '@/api/pagination.ts'
import { Pagination, PaginationContent, PaginationEllipsis, PaginationItem } from '@/components/ui/pagination'
import { Button } from '@/components/ui/button'
import ExerciseMigrationNotice from '@/components/ExerciseMigrationNotice.vue'
import { useI18n } from '@/composables/useI18n'
import { formatDate } from '@/lib/utils.ts'

const { t } = useI18n()

const exercises = ref<Exercise[]>([])
const currentPage = ref(1)
const paginationData = ref<PaginationData>({
    page: 1,
    page_size: 20,
    total: 0,
    total_pages: 0,
})
const isLoading = ref(false)
const errorMessage = ref('')

const getStatusLabel = (status: string) => {
    switch (status) {
        case 'pending':
            return t.value.exerciseStatusPending
        case 'inProgress':
            return t.value.exerciseStatusInProgress
        case 'completed':
            return t.value.exerciseStatusCompleted
        case 'failed':
            return t.value.exerciseStatusFailed
        case 'ignored':
            return t.value.exerciseStatusIgnored
        default:
            return status
    }
}

const getStatusBadgeClass = (status: string) => {
    switch (status) {
        case 'inProgress':
            return 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:border-sky-400/30 dark:bg-sky-400/10 dark:text-sky-300'
        case 'completed':
            return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-300'
        case 'failed':
            return 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:border-rose-400/30 dark:bg-rose-400/10 dark:text-rose-300'
        case 'ignored':
            return 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:border-amber-400/30 dark:bg-amber-400/10 dark:text-amber-300'
        default:
            return 'border-border bg-muted/40 text-foreground'
    }
}

const getTypeLabel = (type: string) => {
    switch (type) {
        case 'basic/direct':
            return t.value.exerciseTypeBasicDirect
        case 'basic/reversed':
            return t.value.exerciseTypeBasicReversed
        default:
            return type
    }
}

const getTypeBadgeClass = (type: string) => {
    switch (type) {
        case 'basic/direct':
            return 'border-zinc-300 bg-zinc-100 text-zinc-800 dark:border-zinc-400 dark:bg-zinc-200 dark:text-zinc-900'
        case 'basic/reversed':
            return 'border-zinc-900 bg-zinc-900 text-white dark:border-zinc-700 dark:bg-zinc-900 dark:text-white'
        default:
            return 'border-border bg-muted/40 text-foreground'
    }
}

const getStartedAt = (exercise: Exercise) => exercise.started_at ?? exercise.starts_at ?? null
const getFinishedAt = (exercise: Exercise) => exercise.finished_at ?? exercise.finishes_at ?? null

const getWhereLabel = (exercise: Exercise) => {
    if (exercise.telegram_message_id != null) {
        return t.value.exercisesWhereTelegram
    }

    return t.value.exercisesWhereWebsite
}

const getTranslationLabel = (exercise: Exercise) => {
    const original = exercise.vocabulary?.translation?.original ?? {
        word: exercise.original_word ?? '',
        language: exercise.original_language ?? '',
    }
    const translated = exercise.vocabulary?.translation?.translation ?? {
        word: exercise.translation_word ?? '',
        language: exercise.translation_language ?? '',
    }

    if (!original || !translated || !original.word || !translated.word) {
        return t.value.exercisesTranslationUnavailable
    }

    return `${original.word} (${original.language}) -> ${translated.word} (${translated.language})`
}

const fetchExercises = async (page: number) => {
    isLoading.value = true
    errorMessage.value = ''
    currentPage.value = page

    try {
        const response = await exercisesApi.getExercises(page, paginationData.value.page_size)
        exercises.value = response.data
        paginationData.value = response.pagination
    } catch {
        errorMessage.value = t.value.exercisesErrorMessage
    } finally {
        isLoading.value = false
    }
}

const handlePageChange = async (page: number) => {
    await fetchExercises(page)
}

onMounted(() => {
    void fetchExercises(1)
})
</script>

<template>
    <main class="px-6 py-8">
        <div class="mx-auto max-w-6xl space-y-6">
            <ExerciseMigrationNotice />

            <div v-if="errorMessage" class="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                {{ errorMessage }}
            </div>

            <section class="overflow-hidden rounded-2xl border border-border bg-card">
                <div class="overflow-x-auto">
                    <table class="w-full min-w-[1120px]">
                        <thead class="bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground">
                            <tr>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnStatus }}</th>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnStartedAt }}</th>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnFinishedAt }}</th>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnWhere }}</th>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnType }}</th>
                                <th class="px-4 py-3 text-center">{{ t.exercisesColumnTranslation }}</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-if="isLoading">
                                <td colspan="6" class="px-4 py-8 text-center text-sm text-muted-foreground">
                                    {{ t.exercisesLoading }}
                                </td>
                            </tr>
                            <tr v-else-if="exercises.length === 0">
                                <td colspan="6" class="px-4 py-8 text-center text-sm text-muted-foreground">
                                    {{ t.exercisesEmpty }}
                                </td>
                            </tr>
                            <tr v-for="exercise in exercises" :key="exercise.id" class="border-t border-border/70 text-sm">
                                <td class="px-4 py-3 text-center text-foreground">
                                    <span
                                        class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold"
                                        :class="getStatusBadgeClass(exercise.status)"
                                    >
                                        {{ getStatusLabel(exercise.status) }}
                                    </span>
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    {{ getStartedAt(exercise) ? formatDate(getStartedAt(exercise) as string) : t.exercisesNotStarted }}
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    {{ getFinishedAt(exercise) ? formatDate(getFinishedAt(exercise) as string) : t.exercisesNotFinished }}
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    {{ getWhereLabel(exercise) }}
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    <span
                                        class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold"
                                        :class="getTypeBadgeClass(exercise.type)"
                                    >
                                        {{ getTypeLabel(exercise.type) }}
                                    </span>
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">{{ getTranslationLabel(exercise) }}</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </section>

            <div v-if="paginationData.total > 0" class="space-y-3">
                <p class="text-center text-sm text-muted-foreground">
                    {{ t.exercisesTotalCount }}: {{ paginationData.total }}
                </p>

                <Pagination
                    v-slot="{ page }"
                    :total="paginationData.total"
                    :items-per-page="paginationData.page_size"
                    :sibling-count="1"
                    show-edges
                    :default-page="1"
                    :page="currentPage"
                    @update:page="handlePageChange"
                >
                    <PaginationContent v-slot="{ items }" class="flex justify-center gap-1">
                        <template v-for="(item, index) in items">
                            <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                                <Button class="h-11 w-11 p-0" :variant="item.value === page ? 'default' : 'outline'">
                                    {{ item.value }}
                                </Button>
                            </PaginationItem>
                            <PaginationEllipsis v-else :key="item.type + index" :index="index" />
                        </template>
                    </PaginationContent>
                </Pagination>
            </div>
        </div>
    </main>
</template>
