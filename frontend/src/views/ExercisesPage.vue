<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import { exercisesApi, type Exercise } from '@/api/exercises.ts'
import type { PaginationData } from '@/api/pagination.ts'
import { Pagination, PaginationContent, PaginationEllipsis, PaginationItem } from '@/components/ui/pagination'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { useSettingsStore } from '@/stores/settings.ts'
import { formatDate, formatNumber } from '@/lib/utils.ts'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const maxVisibleWordLength = 15

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
const expandedTranslationExerciseIds = ref<Set<string>>(new Set())

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
            return 'border-info/30 bg-info/10 text-info'
        case 'completed':
            return 'border-success/30 bg-success/10 text-success'
        case 'failed':
            return 'border-destructive/30 bg-destructive/10 text-destructive'
        case 'ignored':
            return 'border-warning/30 bg-warning/10 text-warning'
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
        case 'choice/direct':
            return t.value.exerciseTypeChoiceDirect
        case 'choice/reversed':
            return t.value.exerciseTypeChoiceReversed
        case 'match/pairs':
            return t.value.exerciseTypeMatchPairs
        default:
            return type
    }
}

const getTypeBadgeClass = (type: string) => {
    switch (type) {
        case 'basic/direct':
            return 'border-border bg-muted text-muted-foreground'
        case 'basic/reversed':
            return 'border-primary/30 bg-primary/10 text-primary'
        case 'choice/direct':
            return 'border-info/30 bg-info/10 text-info'
        case 'choice/reversed':
            return 'border-warning/30 bg-warning/10 text-warning'
        case 'match/pairs':
            return 'border-success/30 bg-success/10 text-success'
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

const getExerciseVocabularyChanges = (exercise: Exercise) => {
    const vocabularies = [...(exercise.vocabularies ?? [])].sort(
        (left, right) => (left.position ?? 0) - (right.position ?? 0)
    )

    if (vocabularies.length === 0 && exercise.vocabulary?.translation) {
        vocabularies.push({
            id: exercise.id,
            translation: exercise.vocabulary.translation,
            is_correct: true,
            position: 0,
        })
    }

    if (vocabularies.length === 0 && exercise.original_word && exercise.translation_word) {
        vocabularies.push({
            id: exercise.id,
            translation: {
                original: {
                    word: exercise.original_word,
                    language: exercise.original_language ?? '',
                },
                translation: {
                    word: exercise.translation_word,
                    language: exercise.translation_language ?? '',
                },
            },
            is_correct: true,
            position: 0,
        })
    }

    return vocabularies
        .map((vocabulary) => {
            const original = vocabulary.translation?.original
            const translated = vocabulary.translation?.translation

            if (!original?.word || !translated?.word) {
                return {
                    ...vocabulary,
                    translation: null,
                }
            }

            return {
                ...vocabulary,
                translation: {
                    original,
                    translation: translated,
                },
            }
        })
        .filter((vocabulary) => vocabulary.translation || vocabulary.exercise_result)
}

const isExerciseTranslationsExpanded = (exerciseId: string) => expandedTranslationExerciseIds.value.has(exerciseId)

const toggleExerciseTranslations = (exerciseId: string) => {
    const nextExpandedIds = new Set(expandedTranslationExerciseIds.value)

    if (nextExpandedIds.has(exerciseId)) {
        nextExpandedIds.delete(exerciseId)
    } else {
        nextExpandedIds.add(exerciseId)
    }

    expandedTranslationExerciseIds.value = nextExpandedIds
}

const getVisibleExerciseVocabularyChanges = (exercise: Exercise) => {
    const vocabularies = getExerciseVocabularyChanges(exercise)

    if (vocabularies.length <= 1 || isExerciseTranslationsExpanded(exercise.id)) {
        return vocabularies
    }

    return vocabularies.slice(0, 1)
}

const getResultLabel = (result?: string | null) => {
    switch (result) {
        case 'correct':
            return t.value.exerciseResultCorrect
        case 'almost':
            return t.value.exerciseResultAlmost
        case 'wrong':
            return t.value.exerciseResultWrong
        case 'ignored':
            return t.value.exerciseResultIgnored
        default:
            return ''
    }
}

const getResultBadgeClass = (result?: string | null) => {
    switch (result) {
        case 'correct':
            return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:border-emerald-400/30 dark:bg-emerald-400/10 dark:text-emerald-300'
        case 'almost':
            return 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:border-amber-400/30 dark:bg-amber-400/10 dark:text-amber-300'
        case 'wrong':
            return 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:border-rose-400/30 dark:bg-rose-400/10 dark:text-rose-300'
        case 'ignored':
            return 'border-zinc-400/40 bg-zinc-500/10 text-zinc-700 dark:border-zinc-500/40 dark:bg-zinc-400/10 dark:text-zinc-300'
        default:
            return 'border-border bg-muted/40 text-muted-foreground'
    }
}

const formatProgressDelta = (delta?: number | null) => {
    if (delta == null) {
        return t.value.exerciseNoProgressChange
    }

    if (delta > 0) {
        return `+${formatNumber(delta)}`
    }

    return formatNumber(delta)
}

const getProgressDeltaClass = (delta?: number | null) => {
    if (delta == null) {
        return 'text-muted-foreground'
    }

    if (delta > 0) {
        return 'text-emerald-700 dark:text-emerald-300'
    }

    return 'text-rose-700 dark:text-rose-300'
}

const getLanguageBadge = (language: string) => {
    if (!language) {
        return ''
    }

    return `${settingsStore.getFlag(language)} ${language.toUpperCase()}`
}

const isWordShortened = (word: string) => word.length > maxVisibleWordLength

const formatExerciseWord = (word: string) => {
    if (!isWordShortened(word)) {
        return word
    }

    return `${word.slice(0, maxVisibleWordLength)}...`
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
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-6xl space-y-6">
            <h1 class="sr-only">{{ t.exercisesHeading }}</h1>
            <div
                v-if="errorMessage"
                class="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
            >
                {{ errorMessage }}
            </div>

            <section class="overflow-hidden rounded-2xl border border-border bg-card">
                <div class="space-y-3 p-3 lg:hidden">
                    <div
                        v-if="isLoading"
                        class="rounded-xl border border-border/70 bg-background px-4 py-8 text-center text-sm text-muted-foreground"
                    >
                        {{ t.exercisesLoading }}
                    </div>
                    <div
                        v-else-if="exercises.length === 0"
                        class="rounded-xl border border-border/70 bg-background px-4 py-8 text-center text-sm text-muted-foreground"
                    >
                        {{ t.exercisesEmpty }}
                    </div>
                    <template v-else>
                        <article
                            v-for="exercise in exercises"
                            :key="exercise.id"
                            class="space-y-4 rounded-xl border border-border/70 bg-background px-4 py-4"
                        >
                            <div class="flex flex-wrap items-start justify-between gap-2">
                                <span
                                    class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold"
                                    :class="getStatusBadgeClass(exercise.status)"
                                >
                                    {{ getStatusLabel(exercise.status) }}
                                </span>
                                <span
                                    class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold"
                                    :class="getTypeBadgeClass(exercise.type)"
                                >
                                    {{ getTypeLabel(exercise.type) }}
                                </span>
                            </div>

                            <div v-if="getExerciseVocabularyChanges(exercise).length > 0" class="space-y-2">
                                <div
                                    v-for="(vocabulary, vocabularyIndex) in getVisibleExerciseVocabularyChanges(exercise)"
                                    :key="vocabulary.id"
                                    class="flex items-start gap-2"
                                >
                                    <div class="min-w-0 flex-1 space-y-1.5">
                                        <div
                                            v-if="vocabulary.translation"
                                            class="flex flex-wrap items-center gap-2 text-sm text-foreground"
                                        >
                                            <span
                                                class="inline-flex min-w-0 max-w-full rounded-full bg-muted/50 px-3 py-1.5 font-medium"
                                            >
                                                <span class="truncate">{{ vocabulary.translation.original.word }}</span>
                                            </span>
                                            <span class="text-muted-foreground">→</span>
                                            <span
                                                class="inline-flex min-w-0 max-w-full rounded-full bg-muted/50 px-3 py-1.5 font-medium"
                                            >
                                                <span class="truncate">
                                                    {{ vocabulary.translation.translation.word }}
                                                </span>
                                            </span>
                                        </div>
                                        <p v-else class="text-sm text-muted-foreground">
                                            {{ t.exercisesDeletedVocabulary }}
                                        </p>
                                        <div class="flex flex-wrap items-center gap-2">
                                            <span
                                                v-if="vocabulary.translation"
                                                class="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground"
                                            >
                                                {{ getLanguageBadge(vocabulary.translation.original.language) }}
                                                <span class="px-1 text-muted-foreground/70">→</span>
                                                {{ getLanguageBadge(vocabulary.translation.translation.language) }}
                                            </span>
                                            <span
                                                v-if="vocabulary.exercise_result"
                                                class="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold"
                                                :class="getResultBadgeClass(vocabulary.exercise_result)"
                                            >
                                                {{ getResultLabel(vocabulary.exercise_result) }}
                                            </span>
                                            <span
                                                v-if="vocabulary.exercise_result || vocabulary.progress_delta != null"
                                                class="text-xs font-semibold"
                                                :class="getProgressDeltaClass(vocabulary.progress_delta)"
                                            >
                                                {{ formatProgressDelta(vocabulary.progress_delta) }}
                                            </span>
                                        </div>
                                    </div>
                                    <button
                                        v-if="vocabularyIndex === 0 && getExerciseVocabularyChanges(exercise).length > 1"
                                        type="button"
                                        class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-border/70 bg-muted/30 text-muted-foreground transition hover:border-border hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                        :aria-expanded="isExerciseTranslationsExpanded(exercise.id)"
                                        :aria-label="
                                            isExerciseTranslationsExpanded(exercise.id)
                                                ? t.exercisesHideTranslations
                                                : t.exercisesShowTranslations
                                        "
                                        @click="toggleExerciseTranslations(exercise.id)"
                                    >
                                        <ChevronDown
                                            class="h-4 w-4 transition-transform duration-200"
                                            :class="isExerciseTranslationsExpanded(exercise.id) ? 'rotate-180' : ''"
                                        />
                                    </button>
                                </div>
                            </div>
                            <p v-else class="text-sm text-muted-foreground">
                                {{ t.exercisesTranslationUnavailable }}
                            </p>

                            <dl class="grid grid-cols-1 gap-3 text-sm text-muted-foreground">
                                <div class="flex items-start justify-between gap-3">
                                    <dt>{{ t.exercisesColumnStartedAt }}</dt>
                                    <dd class="text-right text-foreground">
                                        {{
                                            getStartedAt(exercise)
                                                ? formatDate(getStartedAt(exercise) as string)
                                                : t.exercisesNotStarted
                                        }}
                                    </dd>
                                </div>
                                <div class="flex items-start justify-between gap-3">
                                    <dt>{{ t.exercisesColumnFinishedAt }}</dt>
                                    <dd class="text-right text-foreground">
                                        {{
                                            getFinishedAt(exercise)
                                                ? formatDate(getFinishedAt(exercise) as string)
                                                : t.exercisesNotFinished
                                        }}
                                    </dd>
                                </div>
                                <div class="flex items-start justify-between gap-3">
                                    <dt>{{ t.exercisesColumnWhere }}</dt>
                                    <dd class="text-right text-foreground">{{ getWhereLabel(exercise) }}</dd>
                                </div>
                            </dl>
                        </article>
                    </template>
                </div>

                <div class="hidden overflow-x-auto lg:block">
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
                            <tr
                                v-for="exercise in exercises"
                                :key="exercise.id"
                                class="border-t border-border/70 text-sm"
                            >
                                <td class="px-4 py-3 text-center text-foreground">
                                    <span
                                        class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold"
                                        :class="getStatusBadgeClass(exercise.status)"
                                    >
                                        {{ getStatusLabel(exercise.status) }}
                                    </span>
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    {{
                                        getStartedAt(exercise)
                                            ? formatDate(getStartedAt(exercise) as string)
                                            : t.exercisesNotStarted
                                    }}
                                </td>
                                <td class="px-4 py-3 text-center text-muted-foreground">
                                    {{
                                        getFinishedAt(exercise)
                                            ? formatDate(getFinishedAt(exercise) as string)
                                            : t.exercisesNotFinished
                                    }}
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
                                <td class="px-4 py-3 text-muted-foreground">
                                    <div
                                        v-if="getExerciseVocabularyChanges(exercise).length > 0"
                                        class="mx-auto flex max-w-[420px] flex-col items-center gap-1.5"
                                    >
                                        <template
                                            v-for="(vocabulary, vocabularyIndex) in getVisibleExerciseVocabularyChanges(
                                                exercise
                                            )"
                                            :key="vocabulary.id"
                                        >
                                            <div class="flex items-center justify-center gap-2">
                                                <div
                                                    class="flex flex-wrap items-center justify-center gap-2 rounded-full border border-border/70 bg-muted/30 px-3 py-1.5 text-left"
                                                >
                                                    <template v-if="vocabulary.translation">
                                                        <span
                                                            class="inline-flex min-w-0 max-w-[7rem] rounded-full bg-background px-2.5 py-1 text-xs font-medium text-foreground"
                                                            :title="vocabulary.translation.original.word"
                                                        >
                                                            <span class="block truncate">
                                                                {{
                                                                    formatExerciseWord(
                                                                        vocabulary.translation.original.word
                                                                    )
                                                                }}
                                                            </span>
                                                        </span>
                                                        <span
                                                            class="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground"
                                                        >
                                                            {{
                                                                getLanguageBadge(
                                                                    vocabulary.translation.original.language
                                                                )
                                                            }}
                                                        </span>
                                                        <span class="shrink-0 text-muted-foreground">→</span>
                                                        <span
                                                            class="inline-flex min-w-0 max-w-[7rem] rounded-full bg-background px-2.5 py-1 text-xs font-medium text-foreground"
                                                            :title="vocabulary.translation.translation.word"
                                                        >
                                                            <span class="block truncate">
                                                                {{
                                                                    formatExerciseWord(
                                                                        vocabulary.translation.translation.word
                                                                    )
                                                                }}
                                                            </span>
                                                        </span>
                                                        <span
                                                            class="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground"
                                                        >
                                                            {{
                                                                getLanguageBadge(
                                                                    vocabulary.translation.translation.language
                                                                )
                                                            }}
                                                        </span>
                                                    </template>
                                                    <span v-else class="text-xs text-muted-foreground">
                                                        {{ t.exercisesDeletedVocabulary }}
                                                    </span>
                                                    <span
                                                        v-if="vocabulary.exercise_result"
                                                        class="inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold"
                                                        :class="getResultBadgeClass(vocabulary.exercise_result)"
                                                    >
                                                        {{ getResultLabel(vocabulary.exercise_result) }}
                                                    </span>
                                                    <span
                                                        v-if="
                                                            vocabulary.exercise_result ||
                                                            vocabulary.progress_delta != null
                                                        "
                                                        class="text-xs font-semibold"
                                                        :class="getProgressDeltaClass(vocabulary.progress_delta)"
                                                    >
                                                        {{ formatProgressDelta(vocabulary.progress_delta) }}
                                                    </span>
                                                </div>
                                                <button
                                                    v-if="
                                                        vocabularyIndex === 0 &&
                                                        getExerciseVocabularyChanges(exercise).length > 1
                                                    "
                                                    type="button"
                                                    class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-border/70 bg-muted/30 text-muted-foreground transition hover:border-border hover:bg-muted/60 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                                    :aria-expanded="isExerciseTranslationsExpanded(exercise.id)"
                                                    :aria-label="
                                                        isExerciseTranslationsExpanded(exercise.id)
                                                            ? t.exercisesHideTranslations
                                                            : t.exercisesShowTranslations
                                                    "
                                                    @click="toggleExerciseTranslations(exercise.id)"
                                                >
                                                    <ChevronDown
                                                        class="h-4 w-4 transition-transform duration-200"
                                                        :class="
                                                            isExerciseTranslationsExpanded(exercise.id)
                                                                ? 'rotate-180'
                                                                : ''
                                                        "
                                                    />
                                                </button>
                                            </div>
                                        </template>
                                    </div>
                                    <span v-else>{{ t.exercisesTranslationUnavailable }}</span>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </section>

            <div v-if="paginationData.total > 0" class="space-y-3">
                <p class="text-center text-sm text-muted-foreground">
                    {{ t.exercisesTotalCount }}: {{ formatNumber(paginationData.total) }}
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
