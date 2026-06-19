<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { X } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import {
    exercisesApi,
    type Exercise,
    type ExerciseMatchCard,
    type MatchPairAttempt,
    type MatchPairResult,
    type MatchPairsCompleteResult,
    type RandomExercise,
    type VerifyResult,
} from '@/api/exercises.ts'
import { Button } from '@/components/ui/button'
import { Kbd } from '@/components/ui/kbd'
import { Progress } from '@/components/ui/progress'
import MatchPairsBoard from '@/components/quiz/MatchPairsBoard.vue'
import { useI18n } from '@/composables/useI18n'
import { useSettingsStore } from '@/stores/settings.ts'
import { formatNumber } from '@/lib/utils.ts'

const QUIZ_SIZE = 10
const FEEDBACK_ADVANCE_DELAY_MS = 1800
const MATCH_FEEDBACK_ADVANCE_DELAY_MS = FEEDBACK_ADVANCE_DELAY_MS * 2
type QuizState = 'loading' | 'question' | 'feedback' | 'results'
type MatchVocabularyResult = MatchPairResult | null
type MatchVocabularyState = {
    result: MatchVocabularyResult
}

const router = useRouter()
const { t } = useI18n()
const settingsStore = useSettingsStore()

const state = ref<QuizState>('loading')
const currentExercise = ref<RandomExercise | null>(null)
const verifyResult = ref<VerifyResult | null>(null)
const matchCompleteResult = ref<MatchPairsCompleteResult | null>(null)
const exerciseIds = ref<string[]>([])
const results = ref<Exercise[]>([])
const isSubmitting = ref(false)
const isLoadingResults = ref(false)
const error = ref<string | null>(null)
const emptyState = ref<'error' | 'mastered' | null>(null)
const feedbackTimeoutId = ref<number | null>(null)
const choiceSubmitTimeoutId = ref<number | null>(null)
const matchResolveTimeoutId = ref<number | null>(null)
const answer = ref('')
const selectedChoiceIndex = ref<number | null>(null)
const selectedMatchCardIds = ref<string[]>([])
const matchPairAttempts = ref<MatchPairAttempt[]>([])
const matchVocabularyStates = ref<Record<string, MatchVocabularyState>>({})
const matchCardWrongAttempts = ref<Record<string, number>>({})
const answerInputRef = ref<HTMLInputElement | null>(null)
const quizRootRef = ref<HTMLElement | null>(null)

const isMatchQuestion = computed(() => currentExercise.value?.type === 'match/pairs')
const isChoiceQuestion = computed(
    () => currentExercise.value?.type === 'choice/direct' || currentExercise.value?.type === 'choice/reversed'
)
const isChoiceAnswerPending = computed(() => choiceSubmitTimeoutId.value != null || isSubmitting.value)
const quizContentClass = computed(() => {
    if (state.value === 'results') return 'w-full max-w-5xl'
    if (isMatchQuestion.value) return 'w-full max-w-2xl'
    return 'w-full max-w-xl'
})

const questionHint = computed(() => {
    if (!currentExercise.value) return ''
    if (currentExercise.value.type === 'match/pairs') {
        return t.value.quizTypeMatchPairsHint
    }
    if (currentExercise.value.type === 'basic/reversed' || currentExercise.value.type === 'choice/reversed') {
        return t.value.quizTypeReversedHint
    }

    return t.value.quizTypeDirectHint
})

const questionNumber = computed(() =>
    Math.min(exerciseIds.value.length + (state.value === 'question' || state.value === 'feedback' ? 1 : 0), QUIZ_SIZE)
)
const quizProgress = computed(() => (questionNumber.value / QUIZ_SIZE) * 100)
const quizShortcuts = computed(() => {
    if (state.value === 'question') {
        if (isMatchQuestion.value) {
            return []
        }

        return isChoiceQuestion.value
            ? [{ label: t.value.quizShortcutChoose, keys: '1 - 4' }]
            : [
                  { label: t.value.quizShortcutSubmit, keys: 'Enter' },
                  { label: t.value.quizShortcutSkip, keys: 'Esc' },
              ]
    }

    if (state.value === 'feedback') {
        return [{ label: t.value.quizShortcutContinue, keys: 'Enter' }]
    }

    if (state.value === 'results') {
        return [
            { label: t.value.quizShortcutMore, keys: 'Enter' },
            { label: t.value.quizShortcutClose, keys: 'Esc' },
        ]
    }

    return []
})

async function startQuiz() {
    state.value = 'loading'
    error.value = null
    emptyState.value = null
    exerciseIds.value = []
    results.value = []
    verifyResult.value = null
    matchCompleteResult.value = null
    await loadNextQuestion()
}

async function loadNextQuestion() {
    state.value = 'loading'
    error.value = null
    emptyState.value = null
    answer.value = ''
    clearChoiceSubmit()
    clearMatchResolve()
    selectedChoiceIndex.value = null
    selectedMatchCardIds.value = []
    matchPairAttempts.value = []
    matchVocabularyStates.value = {}
    matchCardWrongAttempts.value = {}

    try {
        currentExercise.value = await exercisesApi.getRandomExercise()
        verifyResult.value = null
        matchCompleteResult.value = null
        if (isMatchQuestion.value) {
            setupMatchExercise(currentExercise.value.cards ?? [])
        }
        state.value = 'question'
        await nextTick()

        if (isChoiceQuestion.value || isMatchQuestion.value) {
            quizRootRef.value?.focus()
        } else {
            answerInputRef.value?.focus()
        }
    } catch (err: unknown) {
        const apiErr = err as { status?: number; body?: { error?: string } }
        if (apiErr?.status === 422) {
            const isMastered = apiErr.body?.error === 'all vocabulary is already mastered'
            emptyState.value = isMastered ? 'mastered' : 'error'
            error.value = isMastered ? t.value.quizAllVocabularyMastered : t.value.quizNoVocabulary
        } else {
            emptyState.value = 'error'
            error.value = t.value.quizLoadError
        }
    }
}

async function submitAnswer(answer: string) {
    if (!currentExercise.value || !answer.trim() || isSubmitting.value) return

    isSubmitting.value = true
    error.value = null

    try {
        verifyResult.value = await exercisesApi.verifyExercise(currentExercise.value.exercise_id, answer)
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        state.value = 'feedback'
        await nextTick()
        quizRootRef.value?.focus()
        scheduleFeedbackAdvance()
    } catch (err: unknown) {
        const apiErr = err as { status?: number; body?: { error?: string } }
        if (apiErr?.status === 409 && apiErr.body?.error === 'exercise vocabulary was deleted') {
            await loadNextQuestion()
            return
        }

        selectedChoiceIndex.value = null
        error.value = t.value.quizVerifyError
    } finally {
        isSubmitting.value = false
    }
}

function getSkipAnswer(): string {
    return 'termorize skipped answer intentionally incorrect'
}

async function skipAnswer() {
    if (!currentExercise.value || isSubmitting.value || state.value !== 'question' || isChoiceQuestion.value || isMatchQuestion.value) return

    isSubmitting.value = true
    error.value = null

    try {
        verifyResult.value = await exercisesApi.verifyExercise(currentExercise.value.exercise_id, getSkipAnswer())
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        state.value = 'feedback'
        await nextTick()
        quizRootRef.value?.focus()
        scheduleFeedbackAdvance()
    } catch (err: unknown) {
        const apiErr = err as { status?: number; body?: { error?: string } }
        if (apiErr?.status === 409 && apiErr.body?.error === 'exercise vocabulary was deleted') {
            await loadNextQuestion()
            return
        }

        error.value = t.value.quizSkipError
    } finally {
        isSubmitting.value = false
    }
}

function chooseOption(option: string, index: number) {
    if (isSubmitting.value || state.value !== 'question' || !isChoiceQuestion.value) return

    selectedChoiceIndex.value = index
    clearChoiceSubmit()
    choiceSubmitTimeoutId.value = window.setTimeout(() => {
        choiceSubmitTimeoutId.value = null
        void submitAnswer(option)
    }, 220)
}

function setupMatchExercise(cards: ExerciseMatchCard[]) {
    const states: Record<string, MatchVocabularyState> = {}
    for (const card of cards) {
        states[card.vocabulary_id] = {
            result: null,
        }
    }

    matchVocabularyStates.value = states
    matchCardWrongAttempts.value = {}
}

function chooseMatchCard(card: ExerciseMatchCard) {
    if (isSubmitting.value || state.value !== 'question' || !isMatchQuestion.value || isMatchCardResolved(card)) return

    if (selectedMatchCardIds.value.includes(card.id)) {
        clearMatchResolve()
        selectedMatchCardIds.value = selectedMatchCardIds.value.filter((id) => id !== card.id)
        return
    }

    if (selectedMatchCardIds.value.length === 0) {
        selectedMatchCardIds.value = [card.id]
        return
    }

    const firstCardId = selectedMatchCardIds.value[0]
    const firstCard = firstCardId ? getMatchCardById(firstCardId) : null
    if (!firstCard) {
        selectedMatchCardIds.value = [card.id]
        return
    }

    selectedMatchCardIds.value = [firstCard.id, card.id]
    clearMatchResolve()
    matchResolveTimeoutId.value = window.setTimeout(() => {
        matchResolveTimeoutId.value = null
        resolveMatchSelection(firstCard, card)
    }, 180)
}

function resolveMatchSelection(firstCard: ExerciseMatchCard, secondCard: ExerciseMatchCard) {
    const isCorrectPair = firstCard.vocabulary_id === secondCard.vocabulary_id && firstCard.side !== secondCard.side
    const nextStates = { ...matchVocabularyStates.value }
    const nextCardWrongAttempts = { ...matchCardWrongAttempts.value }

    matchPairAttempts.value = [
        ...matchPairAttempts.value,
        {
            first_card_id: firstCard.id,
            second_card_id: secondCard.id,
        },
    ]

    if (isCorrectPair) {
        const state = nextStates[firstCard.vocabulary_id]
        if (state && state.result == null) {
            nextStates[firstCard.vocabulary_id] = {
                result: hasMatchVocabularyCardWrongAttempt(firstCard.vocabulary_id, nextCardWrongAttempts)
                    ? 'almost'
                    : 'correct',
            }
        }
    } else {
        for (const card of [firstCard, secondCard]) {
            const state = nextStates[card.vocabulary_id]
            if (!state || state.result != null) continue

            const wrongAttempts = (nextCardWrongAttempts[card.id] ?? 0) + 1
            nextCardWrongAttempts[card.id] = wrongAttempts

            if (wrongAttempts >= 2) {
                nextStates[card.vocabulary_id] = {
                    result: 'wrong',
                }
            }
        }
    }

    matchVocabularyStates.value = nextStates
    matchCardWrongAttempts.value = nextCardWrongAttempts
    selectedMatchCardIds.value = []

    if (isMatchExerciseComplete()) {
        void completeMatchPairsExercise()
    }
}

function getMatchCardById(cardId: string): ExerciseMatchCard | null {
    return currentExercise.value?.cards?.find((card) => card.id === cardId) ?? null
}

function hasMatchVocabularyCardWrongAttempt(
    vocabularyId: string,
    cardWrongAttempts: Record<string, number> = matchCardWrongAttempts.value
): boolean {
    return (
        currentExercise.value?.cards?.some(
            (card) => card.vocabulary_id === vocabularyId && (cardWrongAttempts[card.id] ?? 0) > 0
        ) ?? false
    )
}

function isMatchCardResolved(card: ExerciseMatchCard): boolean {
    const result = matchVocabularyStates.value[card.vocabulary_id]?.result
    return result === 'correct' || result === 'almost' || result === 'wrong'
}

function isMatchExerciseComplete(): boolean {
    const cards = currentExercise.value?.cards
    if (!cards) return false
    const vocabularyIds = new Set(cards.map((card) => card.vocabulary_id))
    const states = Object.values(matchVocabularyStates.value)
    return states.length === vocabularyIds.size && states.every((state) => state.result != null)
}

async function completeMatchPairsExercise() {
    if (!currentExercise.value || !isMatchQuestion.value || isSubmitting.value) return
    if (!isMatchExerciseComplete() || matchPairAttempts.value.length === 0) return

    isSubmitting.value = true
    error.value = null

    try {
        matchCompleteResult.value = await exercisesApi.completeMatchPairsExercise(
            currentExercise.value.exercise_id,
            matchPairAttempts.value
        )
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        state.value = 'feedback'
        await nextTick()
        quizRootRef.value?.focus()
        scheduleFeedbackAdvance()
    } catch (err: unknown) {
        const apiErr = err as { status?: number; body?: { error?: string } }
        if (apiErr?.status === 409 && apiErr.body?.error === 'exercise vocabulary was deleted') {
            await loadNextQuestion()
            return
        }

        error.value = t.value.quizVerifyError
    } finally {
        isSubmitting.value = false
    }
}

function clearChoiceSubmit() {
    if (choiceSubmitTimeoutId.value != null) {
        window.clearTimeout(choiceSubmitTimeoutId.value)
        choiceSubmitTimeoutId.value = null
    }
}

function clearMatchResolve() {
    if (matchResolveTimeoutId.value != null) {
        window.clearTimeout(matchResolveTimeoutId.value)
        matchResolveTimeoutId.value = null
    }
}

function clearFeedbackAdvance() {
    if (feedbackTimeoutId.value != null) {
        window.clearTimeout(feedbackTimeoutId.value)
        feedbackTimeoutId.value = null
    }
}

function advanceFromFeedback() {
    clearFeedbackAdvance()

    if (exerciseIds.value.length >= QUIZ_SIZE) {
        void showResults()
        return
    }

    void loadNextQuestion()
}

function scheduleFeedbackAdvance() {
    clearFeedbackAdvance()
    const delay = matchCompleteResult.value ? MATCH_FEEDBACK_ADVANCE_DELAY_MS : FEEDBACK_ADVANCE_DELAY_MS
    feedbackTimeoutId.value = window.setTimeout(advanceFromFeedback, delay)
}

async function showResults() {
    state.value = 'results'
    isLoadingResults.value = true

    try {
        results.value = await exercisesApi.getExercisesByIds(exerciseIds.value)
    } catch {
        results.value = []
    } finally {
        isLoadingResults.value = false
        await nextTick()
        quizRootRef.value?.focus()
    }
}

function getChoiceIndexFromKeyboardEvent(event: KeyboardEvent): number | null {
    const code = typeof event.code === 'string' ? event.code : ''
    const key = typeof event.key === 'string' ? event.key : ''
    const codeMatch = code.match(/^(?:Digit|Numpad)([1-4])$/)
    if (codeMatch) {
        return Number(codeMatch[1]) - 1
    }

    const keyMatch = key.match(/^[1-4]$/)
    if (keyMatch) {
        return Number(keyMatch[0]) - 1
    }

    return null
}

function handleKeydown(event: KeyboardEvent) {
    if (event.altKey || event.ctrlKey || event.metaKey) {
        return
    }

    if (event.key === 'Enter') {
        if (state.value === 'results') {
            event.preventDefault()
            void startQuiz()
            return
        }

        if (state.value === 'feedback') {
            event.preventDefault()
            advanceFromFeedback()
            return
        }

        if (state.value === 'question' && !isChoiceQuestion.value && !isMatchQuestion.value) {
            event.preventDefault()
            void submitAnswer(answer.value)
        }

        return
    }

    if (event.key === 'Escape' && state.value === 'question' && !isChoiceQuestion.value && !isMatchQuestion.value) {
        event.preventDefault()
        void skipAnswer()
        return
    }

    if (state.value === 'question' && currentExercise.value && isChoiceQuestion.value) {
        const optionIndex = getChoiceIndexFromKeyboardEvent(event)
        if (optionIndex == null || event.repeat || isSubmitting.value) {
            return
        }

        const selectedOption = currentExercise.value.options[optionIndex]
        if (!selectedOption) {
            return
        }

        event.preventDefault()
        chooseOption(selectedOption, optionIndex)
        return
    }

    if (event.key === 'Escape' && state.value === 'results') {
        event.preventDefault()
        closeQuiz()
    }
}

function closeQuiz() {
    void router.push({ name: 'translation' })
}

const correctResults = computed(() => results.value.filter((e) => e.status === 'completed'))
const wrongResults = computed(() => results.value.filter((e) => e.status === 'failed'))
const score = computed(() => correctResults.value.length)
const matchResolvedCount = computed(
    () => Object.values(matchVocabularyStates.value).filter((state) => state.result != null).length
)
const canRetryMatchCompletion = computed(
    () => state.value === 'question' && isMatchQuestion.value && isMatchExerciseComplete() && !isSubmitting.value
)
const matchFinalCounts = computed(() => {
    const rows = matchCompleteResult.value?.results ?? []
    return {
        correct: rows.filter((row) => row.exercise_result === 'correct').length,
        almost: rows.filter((row) => row.exercise_result === 'almost').length,
        wrong: rows.filter((row) => row.exercise_result === 'wrong').length,
    }
})
const matchFinalPointSummaries = computed(() => {
    const rows = matchCompleteResult.value?.results ?? []
    return {
        correct: formatPointDeltas(
            rows
                .filter((row) => row.exercise_result === 'correct')
                .map((row) => row.progress_delta)
                .filter((delta): delta is number => typeof delta === 'number')
        ),
        almost: formatPointDeltas(
            rows
                .filter((row) => row.exercise_result === 'almost')
                .map((row) => row.progress_delta)
                .filter((delta): delta is number => typeof delta === 'number')
        ),
        wrong: formatPointDeltas(
            rows
                .filter((row) => row.exercise_result === 'wrong')
                .map((row) => row.progress_delta)
                .filter((delta): delta is number => typeof delta === 'number')
        ),
    }
})
const quizPointDeltas = computed(() =>
    results.value.flatMap((exercise) =>
        (exercise.vocabularies ?? [])
            .map((vocabulary) => vocabulary.progress_delta)
            .filter((delta): delta is number => typeof delta === 'number')
    )
)
const feedbackPointDeltas = computed(() => {
    if (matchCompleteResult.value) {
        return matchCompleteResult.value.results
            .map((vocabulary) => vocabulary.progress_delta)
            .filter((delta): delta is number => typeof delta === 'number')
    }

    if (verifyResult.value) {
        return [verifyResult.value.progress_delta]
    }

    return []
})
const quizPointsSummary = computed(() => formatPointDeltas(quizPointDeltas.value))
const feedbackPointsSummary = computed(() => formatPointDeltas(feedbackPointDeltas.value))

function formatPointDeltas(deltas: number[]): string {
    const total = deltas.reduce((sum, delta) => sum + delta, 0)
    if (deltas.length === 0) return formatSignedNumber(total)

    const groups = new Map<number, number>()
    for (const delta of deltas) {
        groups.set(delta, (groups.get(delta) ?? 0) + 1)
    }

    const breakdown = Array.from(groups.entries())
        .sort(([left], [right]) => right - left)
        .map(([delta, count]) => `${formatSignedNumber(delta)} x${formatNumber(count)}`)
        .join(', ')

    return `${formatSignedNumber(total)} (${breakdown})`
}

function formatSignedNumber(value: number): string {
    if (value > 0) return `+${formatNumber(value)}`
    if (value < 0) return `-${formatNumber(Math.abs(value))}`
    return formatNumber(value)
}

function getFlag(lang?: string | null): string {
    if (!lang) return ''
    return settingsStore.getFlag(lang)
}

function getVocabularyLabel(exercise: Exercise): string {
    const vocabularies = exercise.vocabularies
    if (!vocabularies?.length) return '—'
    const first = vocabularies[0]
    if (!first?.translation) return '—'
    const orig = first.translation.original
    const trans = first.translation.translation
    if (!orig && !trans) return '—'
    const origFlag = getFlag(orig?.language)
    const transFlag = getFlag(trans?.language)
    let label = `${origFlag} ${orig?.word ?? ''} — ${trans?.word ?? ''} ${transFlag}`.trim()
    if (vocabularies.length > 1) {
        label += ` (+${vocabularies.length - 1} more)`
    }
    return label
}

function getMatchSideLanguage(side: ExerciseMatchCard['side']): string {
    return currentExercise.value?.cards?.find((card) => card.side === side)?.language ?? ''
}

const resultLabel = computed(() => {
    if (matchCompleteResult.value) {
        return matchCompleteResult.value.status === 'completed' ? t.value.quizResultCorrect : t.value.quizResultWrong
    }
    if (!verifyResult.value) return ''
    if (verifyResult.value.result === 'correct') return t.value.quizResultCorrect
    if (verifyResult.value.result === 'almost') return t.value.quizResultAlmost
    return t.value.quizResultWrong
})

const resultClass = computed(() => {
    if (matchCompleteResult.value) {
        return matchCompleteResult.value.status === 'completed'
            ? 'text-green-600 dark:text-green-400'
            : 'text-red-600 dark:text-red-400'
    }
    if (!verifyResult.value) return ''
    if (verifyResult.value.result === 'correct') return 'text-success'
    if (verifyResult.value.result === 'almost') return 'text-warning'
    return 'text-destructive'
})

onMounted(() => {
    void startQuiz()
})

onBeforeUnmount(() => {
    clearChoiceSubmit()
    clearMatchResolve()
    clearFeedbackAdvance()
})
</script>

<template>
    <main
        ref="quizRootRef"
        class="min-h-full bg-background focus:outline-none"
        tabindex="-1"
        @keydown.capture="handleKeydown"
    >
        <h1 class="sr-only">{{ t.quizTitle }}</h1>
        <div class="border-b border-border px-4 py-3 sm:px-6">
            <div class="flex items-center justify-between gap-3">
                <span class="text-sm font-medium text-muted-foreground">{{ t.quizTitle }}</span>
                <span
                    v-if="state === 'question' || state === 'feedback'"
                    class="text-sm tabular-nums text-muted-foreground"
                >
                    {{ questionNumber }} / {{ QUIZ_SIZE }}
                </span>
                <span v-else class="h-11 w-11" aria-hidden="true"></span>
                <button
                    :aria-label="t.cancel"
                    class="inline-flex h-11 w-11 items-center justify-center rounded-sm text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    @click="closeQuiz"
                >
                    <X class="h-5 w-5" />
                </button>
            </div>
            <Progress
                v-if="state === 'question' || state === 'feedback'"
                :model-value="quizProgress"
                class="mt-2 h-1.5 bg-muted/70"
            />
        </div>

        <div class="flex min-h-[calc(100vh-83px)] flex-col items-center justify-center px-4 py-8 sm:px-6 sm:py-12">
            <div :class="quizContentClass">
                <template v-if="state === 'loading'">
                    <div v-if="error" class="space-y-4 text-center">
                        <p
                            :class="emptyState === 'mastered' ? 'text-success' : 'text-destructive'"
                        >
                            {{ error }}
                        </p>
                        <Button variant="outline" @click="startQuiz">{{ t.quizRetry }}</Button>
                    </div>
                    <div v-else class="flex flex-col items-center gap-3">
                        <div class="h-8 w-8 rounded-full border-b-2 border-primary motion-safe:animate-spin"></div>
                        <p class="text-sm text-muted-foreground">{{ t.quizLoading }}</p>
                    </div>
                </template>

                <template v-else-if="state === 'question'">
                    <div class="space-y-7">
                        <template v-if="isMatchQuestion">
                            <div class="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <span class="text-base">{{ getFlag(getMatchSideLanguage('original')) }}</span>
                                <span aria-hidden="true">↔</span>
                                <span class="text-base">{{ getFlag(getMatchSideLanguage('translation')) }}</span>
                            </div>

                            <p class="text-center text-sm text-muted-foreground">
                                {{ questionHint }}
                            </p>

                            <MatchPairsBoard
                                :cards="currentExercise?.cards ?? []"
                                :selected-card-ids="selectedMatchCardIds"
                                :vocabulary-states="matchVocabularyStates"
                                :card-wrong-attempts="matchCardWrongAttempts"
                                :disabled="isSubmitting"
                                :is-submitting="isSubmitting"
                                :checking-text="t.quizChecking"
                                :board-label="questionHint"
                                @choose="chooseMatchCard"
                            />

                            <p class="text-center text-sm text-muted-foreground">
                                {{ matchResolvedCount }} / 5
                            </p>
                        </template>

                        <template v-else>
                            <div class="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <span class="text-base">{{ getFlag(currentExercise?.language) }}</span>
                                <span aria-hidden="true">→</span>
                                <span class="text-base">{{ getFlag(currentExercise?.answer_language) }}</span>
                            </div>

                            <p class="text-center text-sm text-muted-foreground">
                                {{ questionHint }}
                            </p>

                            <div class="text-center">
                                <p class="break-words text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">
                                    {{ currentExercise?.question_word }}
                                </p>
                            </div>

                            <div
                                v-if="isChoiceQuestion"
                                class="grid gap-2.5 sm:grid-cols-2 sm:gap-3"
                                role="group"
                                :aria-label="questionHint"
                            >
                                <button
                                    v-for="(option, index) in currentExercise?.options ?? []"
                                    :key="option"
                                    type="button"
                                    :aria-pressed="selectedChoiceIndex === index"
                                    :disabled="isSubmitting"
                                    :class="
                                        selectedChoiceIndex === index
                                            ? 'quiz-choice-button--selected'
                                            : 'quiz-choice-button--idle'
                                    "
                                    class="quiz-choice-button flex min-h-16 w-full items-center gap-3 rounded-lg border px-3.5 py-3 text-left text-primary-foreground transition-[border-color,box-shadow,filter,transform] duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-90 sm:min-h-24 sm:items-center sm:gap-4 sm:px-4 sm:py-4"
                                    @click="chooseOption(option, index)"
                                >
                                    <span
                                        class="quiz-choice-index inline-flex h-7 min-w-7 shrink-0 items-center justify-center rounded-md px-2 text-xs font-semibold tabular-nums"
                                    >
                                        {{ index + 1 }}
                                    </span>
                                    <span class="min-w-0 flex-1 break-words text-base font-semibold leading-snug">
                                        {{ option }}
                                    </span>
                                    <span
                                        v-if="isChoiceAnswerPending && selectedChoiceIndex === index"
                                        class="quiz-inline-spinner ml-auto h-4 w-4 shrink-0 rounded-full border-2 border-primary-foreground/35 border-t-primary-foreground"
                                        aria-hidden="true"
                                    ></span>
                                </button>
                            </div>

                            <form v-else class="space-y-3" @submit.prevent="submitAnswer(answer)">
                                <input
                                    ref="answerInputRef"
                                    v-model="answer"
                                    :placeholder="t.quizAnswerPlaceholder"
                                    :disabled="isSubmitting"
                                    class="w-full rounded-md border border-input bg-background px-4 py-3 text-base shadow-sm outline-none transition focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                                    autocomplete="off"
                                    autocapitalize="off"
                                    autocorrect="off"
                                    spellcheck="false"
                                />
                                <div class="grid gap-2 sm:grid-cols-[auto_1fr]">
                                    <Button
                                        class="w-full"
                                        size="lg"
                                        type="button"
                                        variant="outline"
                                        :disabled="isSubmitting"
                                        @click="skipAnswer"
                                    >
                                        {{ t.quizSkip }}
                                    </Button>
                                    <Button
                                        class="w-full"
                                        size="lg"
                                        type="submit"
                                        :disabled="isSubmitting || !answer.trim()"
                                    >
                                        <span
                                            v-if="isSubmitting"
                                            class="quiz-inline-spinner mr-2 h-4 w-4 rounded-full border-2 border-primary-foreground/35 border-t-primary-foreground"
                                            aria-hidden="true"
                                        ></span>
                                        {{ isSubmitting ? t.quizChecking : t.quizSubmit }}
                                    </Button>
                                </div>
                            </form>
                        </template>

                        <div v-if="error" class="space-y-3 text-center">
                            <p class="text-sm text-destructive">{{ error }}</p>
                            <Button
                                v-if="canRetryMatchCompletion"
                                size="sm"
                                variant="outline"
                                @click="completeMatchPairsExercise"
                            >
                                {{ t.quizRetry }}
                            </Button>
                        </div>
                    </div>
                </template>

                <template v-else-if="state === 'feedback'">
                    <div class="space-y-6 text-center">
                        <div class="space-y-2">
                            <p class="text-3xl font-bold" :class="resultClass">{{ resultLabel }}</p>
                            <p
                                v-if="feedbackPointDeltas.length > 0 && !matchCompleteResult"
                                class="text-sm font-medium text-muted-foreground"
                            >
                                {{ t.quizPoints }}: {{ feedbackPointsSummary }}
                            </p>
                        </div>
                        <template v-if="matchCompleteResult">
                            <div class="mx-auto grid max-w-sm grid-cols-3 gap-2 text-sm">
                                <div class="rounded-md border border-emerald-500/25 bg-emerald-500/10 px-3 py-2">
                                    <p class="font-semibold text-emerald-700 dark:text-emerald-300">
                                        {{ matchFinalCounts.correct }}
                                    </p>
                                    <p class="text-muted-foreground">{{ t.exerciseResultCorrect }}</p>
                                    <p class="mt-1 text-xs font-medium text-emerald-700 dark:text-emerald-300">
                                        {{ matchFinalPointSummaries.correct }}
                                    </p>
                                </div>
                                <div class="rounded-md border border-amber-500/25 bg-amber-500/10 px-3 py-2">
                                    <p class="font-semibold text-amber-700 dark:text-amber-300">
                                        {{ matchFinalCounts.almost }}
                                    </p>
                                    <p class="text-muted-foreground">{{ t.exerciseResultAlmost }}</p>
                                    <p class="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">
                                        {{ matchFinalPointSummaries.almost }}
                                    </p>
                                </div>
                                <div class="rounded-md border border-rose-500/25 bg-rose-500/10 px-3 py-2">
                                    <p class="font-semibold text-rose-700 dark:text-rose-300">
                                        {{ matchFinalCounts.wrong }}
                                    </p>
                                    <p class="text-muted-foreground">{{ t.exerciseResultWrong }}</p>
                                    <p class="mt-1 text-xs font-medium text-rose-700 dark:text-rose-300">
                                        {{ matchFinalPointSummaries.wrong }}
                                    </p>
                                </div>
                            </div>
                        </template>
                        <template v-else>
                            <div class="space-y-1">
                                <p class="text-sm text-muted-foreground">{{ t.quizCorrectAnswer }}</p>
                                <p class="text-xl font-medium">{{ verifyResult?.correct_answer }}</p>
                            </div>
                            <p class="text-sm text-muted-foreground">
                                {{ t.quizKnowledge }}: {{ verifyResult?.knowledge }}%
                            </p>
                        </template>
                    </div>
                </template>

                <template v-else-if="state === 'results'">
                    <div v-if="isLoadingResults" class="flex items-center justify-center py-8">
                        <div class="h-6 w-6 rounded-full border-b-2 border-primary motion-safe:animate-spin"></div>
                    </div>

                    <div v-else class="space-y-6">
                        <p class="text-center text-4xl font-bold">
                            {{ formatNumber(score) }} / {{ formatNumber(QUIZ_SIZE) }}
                        </p>
                        <p class="text-center text-sm font-medium text-muted-foreground">
                            {{ t.quizPoints }}: {{ quizPointsSummary }}
                        </p>

                        <div class="grid gap-8 sm:grid-cols-2">
                            <div class="space-y-2">
                                <p class="text-base font-medium text-success sm:text-lg">
                                    ✓ {{ t.quizCorrect }}
                                </p>
                                <ul class="space-y-1">
                                    <li
                                        v-for="exercise in correctResults"
                                        :key="exercise.id"
                                        class="text-base text-foreground sm:text-lg"
                                    >
                                        {{ getVocabularyLabel(exercise) }}
                                    </li>
                                    <li
                                        v-if="correctResults.length === 0"
                                        class="text-base italic text-muted-foreground sm:text-lg"
                                    >
                                        —
                                    </li>
                                </ul>
                            </div>
                            <div class="space-y-2">
                                <p class="text-base font-medium text-destructive sm:text-lg">
                                    ✗ {{ t.quizWrong }}
                                </p>
                                <ul class="space-y-1">
                                    <li
                                        v-for="exercise in wrongResults"
                                        :key="exercise.id"
                                        class="text-base text-foreground sm:text-lg"
                                    >
                                        {{ getVocabularyLabel(exercise) }}
                                    </li>
                                    <li
                                        v-if="wrongResults.length === 0"
                                        class="text-base italic text-muted-foreground sm:text-lg"
                                    >
                                        —
                                    </li>
                                </ul>
                            </div>
                        </div>

                        <div class="flex flex-col gap-3 pt-2 sm:flex-row">
                            <Button class="w-full sm:flex-1" size="lg" @click="startQuiz">{{ t.quizMore }}</Button>
                            <Button class="w-full sm:flex-1" size="lg" variant="outline" @click="closeQuiz">{{
                                t.quizEnough
                            }}</Button>
                        </div>
                    </div>
                </template>
            </div>

            <div v-if="quizShortcuts.length > 0" class="mt-6 flex justify-center">
                <div
                    class="hidden w-fit grid-cols-[max-content_max-content] items-center gap-x-3 gap-y-2 text-xs text-muted-foreground md:grid"
                >
                    <template v-for="shortcut in quizShortcuts" :key="shortcut.label">
                        <span class="justify-self-end text-right">{{ shortcut.label }}</span>
                        <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">{{ shortcut.keys }}</Kbd>
                    </template>
                </div>
            </div>
        </div>
    </main>
</template>

<style scoped>
.quiz-choice-button {
    background: var(--primary);
    border-color: color-mix(in oklab, var(--primary-foreground) 18%, var(--primary));
    box-shadow:
        inset 0 1px 0 color-mix(in oklab, var(--primary-foreground) 16%, transparent),
        0 10px 24px -22px var(--primary);
}

.quiz-choice-button--idle:hover:not(:disabled) {
    filter: brightness(1.06) saturate(1.02);
    box-shadow:
        inset 0 1px 0 color-mix(in oklab, var(--primary-foreground) 18%, transparent),
        0 16px 30px -22px var(--primary);
    transform: translateY(-1px);
}

.quiz-choice-button--selected {
    filter: brightness(1.03) saturate(1.02);
    border-color: color-mix(in oklab, var(--primary-foreground) 35%, var(--primary));
    box-shadow:
        inset 0 1px 0 color-mix(in oklab, var(--primary-foreground) 18%, transparent),
        0 0 0 2px color-mix(in oklab, var(--primary-foreground) 18%, transparent),
        0 14px 28px -22px var(--primary);
}

.quiz-choice-button:active:not(:disabled) {
    filter: brightness(0.96) saturate(1.01);
    transform: translateY(1px);
    box-shadow:
        inset 0 2px 3px color-mix(in oklab, var(--primary) 38%, transparent),
        0 4px 14px -18px var(--primary);
}

.quiz-choice-index {
    background: color-mix(in oklab, var(--primary-foreground) 14%, transparent);
    color: color-mix(in oklab, var(--primary-foreground) 88%, var(--primary));
    box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--primary-foreground) 16%, transparent);
}

.quiz-inline-spinner {
    animation: quiz-spin 0.7s linear infinite;
}

@keyframes quiz-spin {
    to {
        transform: rotate(360deg);
    }
}

@media (prefers-reduced-motion: reduce) {
    .quiz-choice-button {
        transition: none;
    }

    .quiz-inline-spinner {
        animation: none;
    }

    .quiz-choice-button--idle:hover:not(:disabled),
    .quiz-choice-button:active:not(:disabled) {
        transform: none;
    }
}
</style>
