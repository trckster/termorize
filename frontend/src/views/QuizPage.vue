<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { TriangleAlert, X } from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
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
import { collectionsApi, type CollectionPracticeRound } from '@/api/collections.ts'
import { settingsApi } from '@/api/settings.ts'
import { Button } from '@/components/ui/button'
import { Kbd } from '@/components/ui/kbd'
import { Progress } from '@/components/ui/progress'
import MatchPairsBoard from '@/components/quiz/MatchPairsBoard.vue'
import PronunciationButton from '@/components/PronunciationButton.vue'
import { useI18n } from '@/composables/useI18n'
import { useAuthStore } from '@/stores/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { formatNumber } from '@/lib/utils.ts'

const NORMAL_QUIZ_SIZE = 10
const FEEDBACK_ADVANCE_DELAY_MS = 1800
const MATCH_FEEDBACK_ADVANCE_DELAY_MS = FEEDBACK_ADVANCE_DELAY_MS * 2
type QuizState = 'loading' | 'question' | 'feedback' | 'results'
type MatchVocabularyResult = MatchPairResult | null
type MatchVocabularyState = {
    result: MatchVocabularyResult
}
type AudioIgnoreState = 'idle' | 'saving' | 'undo' | 'undo-failed'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()

const state = ref<QuizState>('loading')
const currentExercise = ref<RandomExercise | null>(null)
const ignoredExerciseId = ref<string | null>(null)
const verifyResult = ref<VerifyResult | null>(null)
const matchCompleteResult = ref<MatchPairsCompleteResult | null>(null)
const exerciseIds = ref<string[]>([])
const targetExerciseIds = ref<string[]>([])
const results = ref<Exercise[]>([])
const practiceRound = ref<CollectionPracticeRound | null>(null)
const practiceVocabularyIndex = ref(0)
const matchingOffered = ref(false)
const isSubmitting = ref(false)
const isLoadingResults = ref(false)
const error = ref<string | null>(null)
const emptyState = ref<'error' | 'mastered' | null>(null)
const feedbackTimeoutId = ref<number | null>(null)
const choiceSubmitTimeoutId = ref<number | null>(null)
const matchResolveTimeoutId = ref<number | null>(null)
const characterWarningTimeoutId = ref<number | null>(null)
const audioIgnoreTimeoutId = ref<number | null>(null)
const audioIgnoreIntervalId = ref<number | null>(null)
const audioIgnoreState = ref<AudioIgnoreState>('idle')
const audioIgnoreCountdown = ref(5)
const prefersReducedMotion = ref(false)
const answer = ref('')
const showCharacterLanguageWarning = ref(false)
const selectedChoiceIndex = ref<number | null>(null)
const selectedCharacterIndices = ref<number[]>([])
const selectedMatchCardIds = ref<string[]>([])
const matchPairAttempts = ref<MatchPairAttempt[]>([])
const matchVocabularyStates = ref<Record<string, MatchVocabularyState>>({})
const matchCardWrongAttempts = ref<Record<string, number>>({})
const answerInputRef = ref<HTMLInputElement | null>(null)
const quizRootRef = ref<HTMLElement | null>(null)

const collectionId = computed(() => (typeof route.params.collectionId === 'string' ? route.params.collectionId : null))
const isCollectionPractice = computed(() => collectionId.value != null)
const quizSize = computed(() =>
    isCollectionPractice.value ? (practiceRound.value?.vocabulary_ids.length ?? 0) : NORMAL_QUIZ_SIZE
)
const pageTitle = computed(() => (isCollectionPractice.value ? t.value.collectionPracticeTitle : t.value.quizTitle))
const isMatchQuestion = computed(() => currentExercise.value?.type === 'match/pairs')
const isChoiceQuestion = computed(
    () => currentExercise.value?.type === 'choice/direct' || currentExercise.value?.type === 'choice/reversed'
)
const isCharacterQuestion = computed(
    () => currentExercise.value?.type === 'characters/direct' || currentExercise.value?.type === 'characters/reversed'
)
const isAudioQuestion = computed(
    () => currentExercise.value?.type === 'audio/direct' || currentExercise.value?.type === 'audio/reversed'
)
const isDescriptionQuestion = computed(() => currentExercise.value?.type === 'description/reversed')
const isAnswerDisabled = computed(() => isSubmitting.value || audioIgnoreState.value !== 'idle')
const audioSpokenLanguageName = computed(() =>
    settingsStore.getLanguageName(
        currentExercise.value?.language ?? '',
        authStore.user?.settings.system_language ?? 'en'
    )
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
    if (isAudioQuestion.value) {
        return t.value.quizTypeAudioHint
    }
    if (isDescriptionQuestion.value) {
        return t.value.quizTypeDescriptionHint
    }
    if (
        currentExercise.value.type === 'basic/reversed' ||
        currentExercise.value.type === 'choice/reversed' ||
        currentExercise.value.type === 'characters/reversed'
    ) {
        return t.value.quizTypeReversedHint
    }

    return t.value.quizTypeDirectHint
})

const questionNumber = computed(() =>
    isCollectionPractice.value
        ? Math.min(
              practiceVocabularyIndex.value +
                  (state.value === 'question' || state.value === 'feedback' ? (isMatchQuestion.value ? 0 : 1) : 0),
              quizSize.value
          )
        : Math.min(
              exerciseIds.value.length + (state.value === 'question' || state.value === 'feedback' ? 1 : 0),
              quizSize.value
          )
)
const quizProgress = computed(() => (quizSize.value > 0 ? (questionNumber.value / quizSize.value) * 100 : 0))
const quizShortcuts = computed(() => {
    if (state.value === 'question') {
        if (isMatchQuestion.value) {
            return []
        }

        if (isCharacterQuestion.value) {
            return [
                { label: t.value.quizShortcutBuild, keys: t.value.quizShortcutKeyboard },
                { label: t.value.quizShortcutSkip, keys: 'Esc' },
            ]
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
            {
                label: isCollectionPractice.value ? t.value.collectionPracticeAgain : t.value.quizShortcutMore,
                keys: 'Enter',
            },
            {
                label: isCollectionPractice.value ? t.value.collectionPracticeStop : t.value.quizShortcutClose,
                keys: 'Esc',
            },
        ]
    }

    return []
})

async function startQuiz() {
    state.value = 'loading'
    error.value = null
    emptyState.value = null
    exerciseIds.value = []
    targetExerciseIds.value = []
    results.value = []
    verifyResult.value = null
    matchCompleteResult.value = null

    if (isCollectionPractice.value && collectionId.value) {
        try {
            practiceRound.value = await collectionsApi.startPractice(collectionId.value)
            practiceVocabularyIndex.value = 0
            matchingOffered.value = false
        } catch (err: unknown) {
            const apiErr = err as { status?: number }
            emptyState.value = 'error'
            error.value = apiErr?.status === 422 ? t.value.collectionPracticeEmpty : t.value.collectionPracticeLoadError
            return
        }
    }

    await loadNextQuestion()
}

async function loadNextQuestion(excludeAudio: boolean = false) {
    state.value = 'loading'
    error.value = null
    emptyState.value = null
    answer.value = ''
    clearChoiceSubmit()
    clearMatchResolve()
    clearCharacterLanguageWarning()
    clearAudioIgnoreTimer()
    audioIgnoreState.value = 'idle'
    audioIgnoreCountdown.value = 5
    selectedChoiceIndex.value = null
    selectedCharacterIndices.value = []
    selectedMatchCardIds.value = []
    matchPairAttempts.value = []
    matchVocabularyStates.value = {}
    matchCardWrongAttempts.value = {}

    try {
        if (isCollectionPractice.value && collectionId.value) {
            const targetVocabularyId = practiceRound.value?.vocabulary_ids[practiceVocabularyIndex.value]
            if (!targetVocabularyId) {
                await showResults()
                return
            }

            const requestMatching = !matchingOffered.value && Math.random() < 0.1
            if (requestMatching) {
                matchingOffered.value = true
            }
            currentExercise.value = await exercisesApi.getCollectionPracticeExercise(
                collectionId.value,
                targetVocabularyId,
                requestMatching,
                excludeAudio
            )
        } else {
            currentExercise.value = await exercisesApi.getRandomExercise(excludeAudio)
        }
        verifyResult.value = null
        matchCompleteResult.value = null
        if (isMatchQuestion.value) {
            setupMatchExercise(currentExercise.value.cards ?? [])
        }
        state.value = 'question'
        await nextTick()

        if (isChoiceQuestion.value || isCharacterQuestion.value || isMatchQuestion.value) {
            quizRootRef.value?.focus()
        } else {
            answerInputRef.value?.focus()
        }
    } catch (err: unknown) {
        const apiErr = err as { status?: number; body?: { error?: string } }
        if (
            isCollectionPractice.value &&
            apiErr?.status === 409 &&
            apiErr.body?.error === 'collection practice word is no longer available'
        ) {
            practiceVocabularyIndex.value++
            if (practiceVocabularyIndex.value >= quizSize.value) {
                await showResults()
            } else {
                await loadNextQuestion()
            }
            return
        }
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
    if (!currentExercise.value || !answer.trim() || isAnswerDisabled.value || state.value !== 'question') return

    isSubmitting.value = true
    error.value = null

    try {
        verifyResult.value = await exercisesApi.verifyExercise(currentExercise.value.exercise_id, answer)
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        if (isCollectionPractice.value) {
            targetExerciseIds.value = [...targetExerciseIds.value, currentExercise.value.exercise_id]
        }
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
    if (
        !currentExercise.value ||
        isAnswerDisabled.value ||
        state.value !== 'question' ||
        isChoiceQuestion.value ||
        isMatchQuestion.value
    )
        return

    isSubmitting.value = true
    error.value = null

    try {
        verifyResult.value = await exercisesApi.verifyExercise(currentExercise.value.exercise_id, getSkipAnswer())
        exerciseIds.value = [...exerciseIds.value, currentExercise.value.exercise_id]
        if (isCollectionPractice.value) {
            targetExerciseIds.value = [...targetExerciseIds.value, currentExercise.value.exercise_id]
        }
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

function formatQuizText(template: string, values: Record<string, string | number>): string {
    return Object.entries(values).reduce((result, [key, value]) => result.replace(`{${key}}`, String(value)), template)
}

function clearAudioIgnoreTimer() {
    if (audioIgnoreTimeoutId.value != null) {
        window.clearTimeout(audioIgnoreTimeoutId.value)
        audioIgnoreTimeoutId.value = null
    }
    if (audioIgnoreIntervalId.value != null) {
        window.clearInterval(audioIgnoreIntervalId.value)
        audioIgnoreIntervalId.value = null
    }
}

function scheduleAudioReplacement() {
    clearAudioIgnoreTimer()
    audioIgnoreCountdown.value = 5
    audioIgnoreIntervalId.value = window.setInterval(() => {
        audioIgnoreCountdown.value = Math.max(0, audioIgnoreCountdown.value - 1)
    }, 1000)
    audioIgnoreTimeoutId.value = window.setTimeout(() => {
        if (audioIgnoreState.value !== 'undo') return
        audioIgnoreState.value = 'saving'
        clearAudioIgnoreTimer()
        void loadNextQuestion(true)
    }, 5000)
}

async function ignoreCurrentAudioLanguage() {
    if (!currentExercise.value || !isAudioQuestion.value || audioIgnoreState.value !== 'idle') return

    audioIgnoreState.value = 'saving'
    error.value = null
    try {
        authStore.user = await exercisesApi.ignoreAudioLanguage(currentExercise.value.exercise_id)
        ignoredExerciseId.value = currentExercise.value.exercise_id
        audioIgnoreState.value = 'undo'
        scheduleAudioReplacement()
    } catch {
        audioIgnoreState.value = 'idle'
        error.value = t.value.quizIgnoreAudioError
    }
}

async function undoIgnoredAudioLanguage() {
    if (
        !currentExercise.value ||
        !isAudioQuestion.value ||
        (audioIgnoreState.value !== 'undo' && audioIgnoreState.value !== 'undo-failed')
    )
        return

    clearAudioIgnoreTimer()
    audioIgnoreState.value = 'saving'
    error.value = null
    try {
        authStore.user = await settingsApi.removeIgnoredAudioLanguage(currentExercise.value.language)
        await loadNextQuestion(true)
    } catch {
        audioIgnoreState.value = 'undo-failed'
        error.value = t.value.quizUndoAudioError
    }
}

function continueAfterIgnoredAudio() {
    if (audioIgnoreState.value !== 'undo-failed') return
    clearAudioIgnoreTimer()
    audioIgnoreState.value = 'saving'
    void loadNextQuestion(true)
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

function chooseCharacter(index: number) {
    if (
        !currentExercise.value ||
        !isCharacterQuestion.value ||
        state.value !== 'question' ||
        isSubmitting.value ||
        selectedCharacterIndices.value.includes(index)
    ) {
        return
    }

    const character = currentExercise.value.options[index]
    if (character == null) return

    clearCharacterLanguageWarning()
    selectedCharacterIndices.value = [...selectedCharacterIndices.value, index]
    if (selectedCharacterIndices.value.length === currentExercise.value.options.length) {
        void submitAnswer(getCharacterAnswer())
    } else {
        void nextTick(() => quizRootRef.value?.focus())
    }
}

function removeLastCharacter() {
    if (!isCharacterQuestion.value || state.value !== 'question' || isSubmitting.value) return
    clearCharacterLanguageWarning()
    selectedCharacterIndices.value = selectedCharacterIndices.value.slice(0, -1)
    void nextTick(() => quizRootRef.value?.focus())
}

function getCharacterAnswer(): string {
    const options = currentExercise.value?.options ?? []
    return selectedCharacterIndices.value.map((index) => options[index] ?? '').join('')
}

function displayCharacter(character?: string): string {
    if (character === ' ') return '⎵'
    if (character === '\t') return '⇥'
    return character ?? ''
}

function findAvailableCharacterIndex(key: string): number | null {
    const options = currentExercise.value?.options ?? []
    const normalizedKey = key.toLocaleLowerCase()
    const index = options.findIndex(
        (character, optionIndex) =>
            !selectedCharacterIndices.value.includes(optionIndex) && character.toLocaleLowerCase() === normalizedKey
    )
    return index >= 0 ? index : null
}

function getWritingSystem(character: string): 'latin' | 'cyrillic' | 'other' | null {
    if (!/\p{L}/u.test(character)) return null
    if (/\p{Script=Latin}/u.test(character)) return 'latin'
    if (/\p{Script=Cyrillic}/u.test(character)) return 'cyrillic'
    return 'other'
}

function isDifferentAnswerWritingSystem(character: string): boolean {
    const typedWritingSystem = getWritingSystem(character)
    if (!typedWritingSystem) return false

    const answerWritingSystems = new Set(
        (currentExercise.value?.options ?? [])
            .map((option) => getWritingSystem(option))
            .filter((writingSystem): writingSystem is 'latin' | 'cyrillic' | 'other' => writingSystem != null)
    )

    return answerWritingSystems.size === 1 && !answerWritingSystems.has(typedWritingSystem)
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

function clearCharacterLanguageWarning() {
    showCharacterLanguageWarning.value = false
    if (characterWarningTimeoutId.value != null) {
        window.clearTimeout(characterWarningTimeoutId.value)
        characterWarningTimeoutId.value = null
    }
}

function showInvalidCharacterLanguageWarning() {
    clearCharacterLanguageWarning()
    showCharacterLanguageWarning.value = true
    characterWarningTimeoutId.value = window.setTimeout(() => {
        showCharacterLanguageWarning.value = false
        characterWarningTimeoutId.value = null
    }, 2200)
}

function advanceFromFeedback() {
    clearFeedbackAdvance()

    if (isCollectionPractice.value) {
        if (!isMatchQuestion.value) {
            practiceVocabularyIndex.value++
        }

        if (practiceVocabularyIndex.value >= quizSize.value) {
            void showResults()
            return
        }

        void loadNextQuestion()
        return
    }

    if (exerciseIds.value.length >= quizSize.value) {
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
    if (state.value === 'question' && isAudioQuestion.value && audioIgnoreState.value !== 'idle') {
        return
    }

    if (state.value === 'question' && currentExercise.value && isCharacterQuestion.value) {
        if (event.key === 'Escape') {
            event.preventDefault()
            void skipAnswer()
            return
        }

        if (event.key === 'Backspace') {
            event.preventDefault()
            removeLastCharacter()
            return
        }

        if (!event.repeat && Array.from(event.key).length === 1) {
            const characterIndex = findAvailableCharacterIndex(event.key)
            if (characterIndex != null) {
                event.preventDefault()
                chooseCharacter(characterIndex)
            } else if (isDifferentAnswerWritingSystem(event.key)) {
                event.preventDefault()
                showInvalidCharacterLanguageWarning()
            }
        }
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

        if (
            state.value === 'question' &&
            !isChoiceQuestion.value &&
            !isCharacterQuestion.value &&
            !isMatchQuestion.value
        ) {
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
        void closeQuiz()
    }
}

function handleQuizBodyClick(event: MouseEvent) {
    if (state.value !== 'feedback' || event.button !== 0) {
        return
    }

    advanceFromFeedback()
}

async function closeQuiz() {
    clearChoiceSubmit()
    clearMatchResolve()
    clearFeedbackAdvance()

    const exerciseId = currentExercise.value?.exercise_id
    if (exerciseId && state.value === 'question') {
        try {
            await exercisesApi.ignoreExercise(exerciseId)
            ignoredExerciseId.value = exerciseId
        } catch {
            // Closing the quiz should not trap the user if the exercise was already handled elsewhere.
        }
    }

    if (isCollectionPractice.value && collectionId.value) {
        void router.push({ name: 'collection-detail', params: { id: collectionId.value } })
        return
    }

    void router.push({ name: 'translation' })
}

function ignoreCurrentExerciseOnPageExit() {
    const exerciseId = currentExercise.value?.exercise_id
    if (exerciseId && exerciseId !== ignoredExerciseId.value && state.value === 'question') {
        ignoredExerciseId.value = exerciseId
        exercisesApi.ignoreExerciseOnPageExit(exerciseId)
    }
}

const scoredResults = computed(() =>
    isCollectionPractice.value
        ? results.value.filter((exercise) => targetExerciseIds.value.includes(exercise.id))
        : results.value
)
const correctResults = computed(() => scoredResults.value.filter((e) => e.status === 'completed'))
const wrongResults = computed(() => scoredResults.value.filter((e) => e.status === 'failed'))
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
    if (!vocabularies?.length) return '–'
    const first = vocabularies[0]
    if (!first?.translation) return '–'
    const orig = first.translation.original
    const trans = first.translation.translation
    if (!orig && !trans) return '–'
    const origFlag = getFlag(orig?.language)
    const transFlag = getFlag(trans?.language)
    let label = `${origFlag} ${orig?.word ?? ''} → ${trans?.word ?? ''} ${transFlag}`.trim()
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
    prefersReducedMotion.value = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.addEventListener('pagehide', ignoreCurrentExerciseOnPageExit)
    void startQuiz()
})

onBeforeUnmount(() => {
    window.removeEventListener('pagehide', ignoreCurrentExerciseOnPageExit)
    ignoreCurrentExerciseOnPageExit()
    clearChoiceSubmit()
    clearMatchResolve()
    clearFeedbackAdvance()
    clearCharacterLanguageWarning()
    clearAudioIgnoreTimer()
})
</script>

<template>
    <main
        ref="quizRootRef"
        class="min-h-full bg-background focus:outline-none"
        tabindex="-1"
        @keydown.capture="handleKeydown"
    >
        <h1 class="sr-only">{{ pageTitle }}</h1>
        <div class="border-b border-border px-4 py-3 sm:px-6">
            <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                    <span class="block text-sm font-medium text-muted-foreground">{{ pageTitle }}</span>
                    <span
                        v-if="isCollectionPractice && practiceRound"
                        class="block truncate text-xs text-muted-foreground/80"
                    >
                        {{ practiceRound.collection_title }}
                    </span>
                </div>
                <span
                    v-if="state === 'question' || state === 'feedback'"
                    class="text-sm tabular-nums text-muted-foreground"
                >
                    {{
                        isCollectionPractice && isMatchQuestion
                            ? t.collectionPracticeMatching
                            : `${questionNumber} / ${quizSize}`
                    }}
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

        <div
            class="flex min-h-[calc(100dvh-83px)] flex-col items-center justify-start px-4 py-6 sm:px-6 sm:py-8 lg:justify-center lg:py-12"
            @click="handleQuizBodyClick"
        >
            <div :class="quizContentClass">
                <template v-if="state === 'loading'">
                    <div v-if="error" class="space-y-4 text-center">
                        <p :class="emptyState === 'mastered' ? 'text-success' : 'text-destructive'">
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
                                :correct-text="t.quizResultCorrect"
                                :invalid-text="t.quizResultWrong"
                                @choose="chooseMatchCard"
                            />

                            <p class="text-center text-sm text-muted-foreground">{{ matchResolvedCount }} / 5</p>
                        </template>

                        <template v-else>
                            <div class="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                                <span class="text-base">{{ getFlag(currentExercise?.language) }}</span>
                                <template v-if="!isDescriptionQuestion">
                                    <span aria-hidden="true">→</span>
                                    <span class="text-base">{{ getFlag(currentExercise?.answer_language) }}</span>
                                </template>
                            </div>

                            <p class="text-center text-sm text-muted-foreground">
                                {{ questionHint }}
                            </p>

                            <div v-if="isAudioQuestion" class="flex flex-col items-center gap-3 text-center">
                                <PronunciationButton
                                    v-if="audioIgnoreState === 'idle'"
                                    :word-id="currentExercise?.audio_word_id"
                                    word=""
                                    prominent
                                    :listen-label="t.quizAudioListen"
                                    :pause-label="t.quizAudioPause"
                                    :loading-label="t.quizAudioLoading"
                                    :error-label="t.quizAudioError"
                                />
                                <div
                                    v-else
                                    class="flex h-11 items-center text-sm font-medium text-muted-foreground"
                                    aria-live="polite"
                                >
                                    {{
                                        audioIgnoreState === 'saving'
                                            ? t.quizIgnoringAudioLanguage
                                            : t.quizTypeAudioHint
                                    }}
                                </div>
                            </div>

                            <div v-else-if="isDescriptionQuestion" class="text-center">
                                <p class="break-words text-xl font-semibold leading-relaxed tracking-tight sm:text-2xl">
                                    {{ currentExercise?.description }}
                                </p>
                            </div>

                            <div v-else class="text-center">
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
                                    :disabled="isAnswerDisabled"
                                    :class="
                                        selectedChoiceIndex === index
                                            ? 'quiz-choice-button--selected'
                                            : 'quiz-choice-button--idle'
                                    "
                                    class="quiz-choice-button flex min-h-16 w-full items-center gap-3 rounded-lg border px-3.5 py-3 text-left transition-[background-color,border-color,color,box-shadow,transform] duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-90 sm:min-h-24 sm:items-center sm:gap-4 sm:px-4 sm:py-4"
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

                            <div v-else-if="isCharacterQuestion" class="space-y-7">
                                <div
                                    class="flex flex-wrap justify-center gap-1.5 sm:gap-2"
                                    role="group"
                                    :aria-label="t.quizCharactersAnswerLabel"
                                >
                                    <button
                                        v-for="(_, position) in currentExercise?.options ?? []"
                                        :key="position"
                                        type="button"
                                        :disabled="
                                            isSubmitting ||
                                            position >= selectedCharacterIndices.length ||
                                            position !== selectedCharacterIndices.length - 1
                                        "
                                        :aria-label="`${t.quizCharacterSlot} ${position + 1}`"
                                        class="flex h-11 w-9 items-center justify-center border-b-2 border-foreground/40 text-xl font-semibold leading-none transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-100 sm:h-12 sm:w-10"
                                        :class="
                                            position < selectedCharacterIndices.length
                                                ? 'border-primary text-foreground'
                                                : 'text-transparent'
                                        "
                                        @click="removeLastCharacter"
                                    >
                                        {{
                                            displayCharacter(
                                                currentExercise?.options[selectedCharacterIndices[position] ?? -1]
                                            ) || '·'
                                        }}
                                    </button>
                                </div>

                                <div class="space-y-3">
                                    <p class="text-center text-sm text-muted-foreground">
                                        {{ t.quizCharactersHint }}
                                    </p>
                                    <div
                                        v-if="showCharacterLanguageWarning"
                                        class="mx-auto flex w-fit max-w-full items-center gap-2 rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-left text-sm font-medium text-warning"
                                        role="status"
                                        aria-live="polite"
                                    >
                                        <TriangleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
                                        <span>{{ t.quizCharactersLanguageWarning }}</span>
                                    </div>
                                    <div
                                        class="flex flex-wrap justify-center gap-2"
                                        role="group"
                                        :aria-label="t.quizCharactersAvailableLabel"
                                    >
                                        <button
                                            v-for="(character, index) in currentExercise?.options ?? []"
                                            :key="`${character}-${index}`"
                                            type="button"
                                            :disabled="isSubmitting || selectedCharacterIndices.includes(index)"
                                            class="inline-flex h-12 min-w-12 items-center justify-center rounded-md border border-input bg-background px-3 text-lg font-semibold shadow-sm transition-[background-color,border-color,color,transform] duration-150 ease-out hover:border-primary/60 hover:bg-accent active:scale-95 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:border-transparent disabled:bg-muted/40 disabled:text-transparent disabled:shadow-none"
                                            @click="chooseCharacter(index)"
                                        >
                                            {{ displayCharacter(character) }}
                                        </button>
                                    </div>
                                </div>

                                <Button
                                    class="w-full"
                                    size="lg"
                                    type="button"
                                    variant="outline"
                                    :disabled="isAnswerDisabled"
                                    @click="skipAnswer"
                                >
                                    {{ t.quizSkip }}
                                </Button>
                            </div>

                            <form v-else class="space-y-3" @submit.prevent="submitAnswer(answer)">
                                <label for="quiz-answer" class="sr-only">{{ t.quizAnswerPlaceholder }}</label>
                                <input
                                    id="quiz-answer"
                                    ref="answerInputRef"
                                    v-model="answer"
                                    name="answer"
                                    :placeholder="t.quizAnswerPlaceholder"
                                    :disabled="isAnswerDisabled"
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
                                        :disabled="isAnswerDisabled"
                                        @click="skipAnswer"
                                    >
                                        {{ t.quizSkip }}
                                    </Button>
                                    <Button
                                        class="w-full"
                                        size="lg"
                                        type="submit"
                                        :disabled="isAnswerDisabled || !answer.trim()"
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

                            <div v-if="isAudioQuestion" class="space-y-2">
                                <Button
                                    v-if="audioIgnoreState === 'idle' || audioIgnoreState === 'saving'"
                                    class="w-full"
                                    type="button"
                                    variant="ghost"
                                    :disabled="audioIgnoreState === 'saving'"
                                    @click="ignoreCurrentAudioLanguage"
                                >
                                    {{
                                        audioIgnoreState === 'saving'
                                            ? t.quizIgnoringAudioLanguage
                                            : formatQuizText(t.quizIgnoreAudioLanguage, {
                                                  language: audioSpokenLanguageName,
                                              })
                                    }}
                                </Button>

                                <button
                                    v-else-if="audioIgnoreState === 'undo'"
                                    type="button"
                                    class="relative flex min-h-11 w-full items-center justify-center overflow-hidden rounded-md border border-success/35 bg-success/10 px-4 py-2 text-sm font-medium text-success transition-colors hover:bg-success/15 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                    @click="undoIgnoredAudioLanguage"
                                >
                                    <span>
                                        {{
                                            formatQuizText(t.quizUndoAudioLanguage, {
                                                language: audioSpokenLanguageName,
                                            })
                                        }}
                                    </span>
                                    <span v-if="prefersReducedMotion" class="ml-2 tabular-nums" aria-live="polite">
                                        {{
                                            formatQuizText(t.quizUndoAudioLanguageCountdown, {
                                                seconds: audioIgnoreCountdown,
                                            })
                                        }}
                                    </span>
                                    <span
                                        v-else
                                        class="quiz-audio-undo-progress absolute inset-x-0 bottom-0 h-0.5 origin-left bg-success"
                                        aria-hidden="true"
                                    ></span>
                                </button>

                                <div v-else class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                                    <Button type="button" variant="outline" @click="undoIgnoredAudioLanguage">
                                        {{ t.quizRetryUndo }}
                                    </Button>
                                    <Button type="button" @click="continueAfterIgnoredAudio">
                                        {{ t.quizNextQuestion }}
                                    </Button>
                                </div>
                            </div>
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
                            <p v-if="isCollectionPractice" class="text-sm font-medium text-muted-foreground">
                                {{ t.collectionPracticeKnowledgeUnchanged }}
                            </p>
                            <p
                                v-else-if="feedbackPointDeltas.length > 0 && !matchCompleteResult"
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
                                    <p
                                        v-if="!isCollectionPractice"
                                        class="mt-1 text-xs font-medium text-emerald-700 dark:text-emerald-300"
                                    >
                                        {{ matchFinalPointSummaries.correct }}
                                    </p>
                                </div>
                                <div class="rounded-md border border-amber-500/25 bg-amber-500/10 px-3 py-2">
                                    <p class="font-semibold text-amber-700 dark:text-amber-300">
                                        {{ matchFinalCounts.almost }}
                                    </p>
                                    <p class="text-muted-foreground">{{ t.exerciseResultAlmost }}</p>
                                    <p
                                        v-if="!isCollectionPractice"
                                        class="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300"
                                    >
                                        {{ matchFinalPointSummaries.almost }}
                                    </p>
                                </div>
                                <div class="rounded-md border border-rose-500/25 bg-rose-500/10 px-3 py-2">
                                    <p class="font-semibold text-rose-700 dark:text-rose-300">
                                        {{ matchFinalCounts.wrong }}
                                    </p>
                                    <p class="text-muted-foreground">{{ t.exerciseResultWrong }}</p>
                                    <p
                                        v-if="!isCollectionPractice"
                                        class="mt-1 text-xs font-medium text-rose-700 dark:text-rose-300"
                                    >
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
                            <p v-if="!isCollectionPractice" class="text-sm text-muted-foreground">
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
                            {{ formatNumber(score) }} / {{ formatNumber(quizSize) }}
                        </p>
                        <p class="text-center text-sm font-medium text-muted-foreground">
                            {{
                                isCollectionPractice
                                    ? t.collectionPracticeKnowledgeUnchanged
                                    : `${t.quizPoints}: ${quizPointsSummary}`
                            }}
                        </p>

                        <div class="grid gap-8 sm:grid-cols-2">
                            <div class="space-y-2">
                                <p class="text-base font-medium text-success sm:text-lg">✓ {{ t.quizCorrect }}</p>
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
                                        –
                                    </li>
                                </ul>
                            </div>
                            <div class="space-y-2">
                                <p class="text-base font-medium text-destructive sm:text-lg">✗ {{ t.quizWrong }}</p>
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
                                        –
                                    </li>
                                </ul>
                            </div>
                        </div>

                        <div class="flex flex-col gap-3 pt-2 sm:flex-row">
                            <Button class="w-full sm:flex-1" size="lg" @click="startQuiz">{{
                                isCollectionPractice ? t.collectionPracticeAgain : t.quizMore
                            }}</Button>
                            <Button class="w-full sm:flex-1" size="lg" variant="outline" @click="closeQuiz">{{
                                isCollectionPractice ? t.collectionPracticeStop : t.quizEnough
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
    background: hsl(var(--card));
    border-color: hsl(var(--border));
    color: hsl(var(--foreground));
    box-shadow: 0 4px 14px -14px hsl(var(--overlay) / 0.35);
}

.quiz-choice-button--idle:hover:not(:disabled) {
    background: hsl(var(--accent) / 0.55);
    border-color: hsl(var(--primary) / 0.45);
    box-shadow: 0 8px 18px -16px hsl(var(--primary) / 0.6);
    transform: translateY(-1px);
}

.quiz-choice-button--selected {
    background: hsl(var(--primary));
    border-color: hsl(var(--primary));
    color: hsl(var(--primary-foreground));
    box-shadow:
        inset 0 1px 0 hsl(var(--primary-foreground) / 0.16),
        0 12px 24px -20px hsl(var(--primary));
}

.quiz-choice-button:active:not(:disabled) {
    transform: translateY(1px);
    box-shadow: inset 0 1px 2px hsl(var(--overlay) / 0.12);
}

.quiz-choice-index {
    background: hsl(var(--secondary));
    color: hsl(var(--secondary-foreground));
    box-shadow: inset 0 0 0 1px hsl(var(--border));
}

.quiz-choice-button--selected .quiz-choice-index {
    background: hsl(var(--primary-foreground) / 0.14);
    color: hsl(var(--primary-foreground));
    box-shadow: inset 0 0 0 1px hsl(var(--primary-foreground) / 0.18);
}

.quiz-inline-spinner {
    animation: quiz-spin 0.7s linear infinite;
}

.quiz-audio-undo-progress {
    animation: quiz-audio-undo-deplete 5s linear forwards;
}

@keyframes quiz-spin {
    to {
        transform: rotate(360deg);
    }
}

@keyframes quiz-audio-undo-deplete {
    from {
        transform: scaleX(1);
    }
    to {
        transform: scaleX(0);
    }
}

@media (prefers-reduced-motion: reduce) {
    .quiz-choice-button {
        transition: none;
    }

    .quiz-inline-spinner {
        animation: none;
    }

    .quiz-audio-undo-progress {
        animation: none;
    }

    .quiz-choice-button--idle:hover:not(:disabled),
    .quiz-choice-button:active:not(:disabled) {
        transform: none;
    }
}
</style>
