<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { X } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { exercisesApi, type Exercise, type RandomExercise, type VerifyResult } from '@/api/exercises.ts'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { useSettingsStore } from '@/stores/settings.ts'

const QUIZ_SIZE = 10

type QuizState = 'loading' | 'question' | 'feedback' | 'results'

const router = useRouter()
const { t } = useI18n()
const settingsStore = useSettingsStore()

const state = ref<QuizState>('loading')
const currentExercise = ref<RandomExercise | null>(null)
const currentAnswer = ref('')
const verifyResult = ref<VerifyResult | null>(null)
const exerciseIds = ref<string[]>([])
const results = ref<Exercise[]>([])
const isSubmitting = ref(false)
const isLoadingResults = ref(false)
const error = ref<string | null>(null)
const answerInputRef = ref<HTMLInputElement | null>(null)
const feedbackTimeoutId = ref<number | null>(null)

const questionNumber = computed(() =>
    Math.min(
        exerciseIds.value.length + (state.value === 'question' || state.value === 'feedback' ? 1 : 0),
        QUIZ_SIZE
    )
)

async function startQuiz() {
    state.value = 'loading'
    error.value = null
    exerciseIds.value = []
    results.value = []
    currentAnswer.value = ''
    verifyResult.value = null
    await loadNextQuestion()
}

async function loadNextQuestion() {
    state.value = 'loading'
    error.value = null

    try {
        currentExercise.value = await exercisesApi.getRandomExercise()
        currentAnswer.value = ''
        verifyResult.value = null
        state.value = 'question'
        await nextTick()
        answerInputRef.value?.focus()
    } catch (err: unknown) {
        const apiErr = err as { status?: number }
        if (apiErr?.status === 422) {
            error.value = t.value.quizNoVocabulary
        } else {
            error.value = t.value.quizLoadError
        }
    }
}

async function submitAnswer() {
    if (!currentExercise.value || !currentAnswer.value.trim() || isSubmitting.value) return

    isSubmitting.value = true
    error.value = null

    try {
        verifyResult.value = await exercisesApi.verifyExercise(currentExercise.value.exercise_id, currentAnswer.value)
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        state.value = 'feedback'
        scheduleFeedbackAdvance()
    } catch {
        error.value = t.value.quizVerifyError
    } finally {
        isSubmitting.value = false
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
    feedbackTimeoutId.value = window.setTimeout(advanceFromFeedback, 1800)
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
    }
}

function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
        event.preventDefault()

        if (state.value === 'results') {
            void startQuiz()
            return
        }

        if (state.value === 'feedback') {
            advanceFromFeedback()
            return
        }

        if (state.value === 'question') {
            void submitAnswer()
        }

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

const correctResults = computed(() => results.value.filter(e => e.status === 'completed'))
const wrongResults = computed(() => results.value.filter(e => e.status === 'failed'))
const score = computed(() => correctResults.value.length)

function getFlag(lang?: string | null): string {
    if (!lang) return ''
    return settingsStore.getFlag(lang)
}

function getVocabularyLabel(exercise: Exercise): string {
    const vocab = exercise.vocabularies?.[0]
    if (!vocab?.translation) return '—'
    const orig = vocab.translation.original
    const trans = vocab.translation.translation
    if (!orig && !trans) return '—'
    const origFlag = getFlag(orig?.language)
    const transFlag = getFlag(trans?.language)
    return `${origFlag} ${orig?.word ?? ''} — ${trans?.word ?? ''} ${transFlag}`.trim()
}

const resultLabel = computed(() => {
    if (!verifyResult.value) return ''
    if (verifyResult.value.result === 'correct') return t.value.quizResultCorrect
    if (verifyResult.value.result === 'almost') return t.value.quizResultAlmost
    return t.value.quizResultWrong
})

const resultClass = computed(() => {
    if (!verifyResult.value) return ''
    if (verifyResult.value.result === 'correct') return 'text-green-600 dark:text-green-400'
    if (verifyResult.value.result === 'almost') return 'text-yellow-600 dark:text-yellow-400'
    return 'text-red-600 dark:text-red-400'
})

onMounted(() => {
    window.addEventListener('keydown', handleKeydown)
    void startQuiz()
})

onBeforeUnmount(() => {
    clearFeedbackAdvance()
    window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
    <main class="min-h-full bg-background">
        <div class="flex items-center justify-between border-b border-border px-4 py-4 sm:px-6">
            <span class="text-sm font-medium text-muted-foreground">{{ t.quizTitle }}</span>
            <span v-if="state === 'question' || state === 'feedback'" class="text-sm tabular-nums text-muted-foreground">
                {{ questionNumber }} / {{ QUIZ_SIZE }}
            </span>
            <button
                :aria-label="t.cancel"
                class="rounded-sm p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring"
                @click="closeQuiz"
            >
                <X class="h-5 w-5" />
            </button>
        </div>

        <div class="flex min-h-[calc(100vh-73px)] flex-col items-center justify-center px-4 py-8 sm:px-6 sm:py-12">
            <div :class="state === 'results' ? 'w-full max-w-5xl' : 'w-full max-w-sm'">
                <template v-if="state === 'loading'">
                    <div v-if="error" class="space-y-4 text-center">
                        <p class="text-destructive">{{ error }}</p>
                        <Button variant="outline" @click="startQuiz">{{ t.quizRetry }}</Button>
                    </div>
                    <div v-else class="flex flex-col items-center gap-3">
                        <div class="h-8 w-8 rounded-full border-b-2 border-primary motion-safe:animate-spin"></div>
                        <p class="text-sm text-muted-foreground">{{ t.quizLoading }}</p>
                    </div>
                </template>

                <template v-else-if="state === 'question'">
                    <div class="space-y-8">
                        <div class="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                            <span class="text-base">{{ getFlag(currentExercise?.language) }}</span>
                            <span>→</span>
                            <span class="text-base">{{ getFlag(currentExercise?.answer_language) }}</span>
                        </div>

                        <div class="text-center">
                            <p class="break-words text-4xl font-semibold tracking-tight">
                                {{ currentExercise?.question_word }}
                            </p>
                        </div>

                        <div class="space-y-3">
                            <input
                                ref="answerInputRef"
                                v-model="currentAnswer"
                                :disabled="isSubmitting"
                                :placeholder="t.quizAnswerPlaceholder"
                                autocomplete="off"
                                autocapitalize="none"
                                autocorrect="off"
                                spellcheck="false"
                                class="w-full rounded-lg border border-border bg-background px-4 py-3 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
                            />
                            <Button class="w-full" size="lg" :disabled="!currentAnswer.trim() || isSubmitting" @click="submitAnswer">
                                {{ isSubmitting ? t.quizChecking : t.quizSubmit }}
                            </Button>
                            <p v-if="error" class="text-center text-sm text-destructive">{{ error }}</p>
                        </div>
                    </div>
                </template>

                <template v-else-if="state === 'feedback'">
                    <div class="space-y-6 text-center">
                        <p class="text-3xl font-bold" :class="resultClass">{{ resultLabel }}</p>
                        <div class="space-y-1">
                            <p class="text-sm text-muted-foreground">{{ t.quizCorrectAnswer }}</p>
                            <p class="text-xl font-medium">{{ verifyResult?.correct_answer }}</p>
                        </div>
                        <p class="text-sm text-muted-foreground">{{ t.quizKnowledge }}: {{ verifyResult?.knowledge }}%</p>
                    </div>
                </template>

                <template v-else-if="state === 'results'">
                    <div v-if="isLoadingResults" class="flex items-center justify-center py-8">
                        <div class="h-6 w-6 rounded-full border-b-2 border-primary motion-safe:animate-spin"></div>
                    </div>

                    <div v-else class="space-y-6">
                        <p class="text-center text-4xl font-bold">{{ score }} / {{ QUIZ_SIZE }}</p>

                        <div class="grid gap-8 sm:grid-cols-2">
                            <div class="space-y-2">
                                <p class="text-base font-medium text-green-600 dark:text-green-400 sm:text-lg">✓ {{ t.quizCorrect }}</p>
                                <ul class="space-y-1">
                                    <li v-for="exercise in correctResults" :key="exercise.id" class="text-base text-foreground sm:text-lg">
                                        {{ getVocabularyLabel(exercise) }}
                                    </li>
                                    <li v-if="correctResults.length === 0" class="text-base italic text-muted-foreground sm:text-lg">—</li>
                                </ul>
                            </div>
                            <div class="space-y-2">
                                <p class="text-base font-medium text-red-600 dark:text-red-400 sm:text-lg">✗ {{ t.quizWrong }}</p>
                                <ul class="space-y-1">
                                    <li v-for="exercise in wrongResults" :key="exercise.id" class="text-base text-foreground sm:text-lg">
                                        {{ getVocabularyLabel(exercise) }}
                                    </li>
                                    <li v-if="wrongResults.length === 0" class="text-base italic text-muted-foreground sm:text-lg">—</li>
                                </ul>
                            </div>
                        </div>

                        <div class="flex flex-col gap-3 pt-2 sm:flex-row">
                            <Button class="w-full sm:flex-1" size="lg" @click="startQuiz">{{ t.quizMore }}</Button>
                            <Button class="w-full sm:flex-1" size="lg" variant="outline" @click="closeQuiz">{{ t.quizEnough }}</Button>
                        </div>
                    </div>
                </template>
            </div>
        </div>
    </main>
</template>
