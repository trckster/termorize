<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { ExerciseMatchCard, MatchPairResult } from '@/api/exercises.ts'

type MatchCardVisualState = 'idle' | 'selected' | 'green' | 'yellow' | 'red'
type MatchVocabularyState = {
    result: MatchPairResult | null
}
type MatchCardLayout = {
    x: number
    y: number
    rotation: number
}
type PlacedCard = {
    x: number
    y: number
    width: number
    height: number
}
type PlacementCandidate = PlacedCard & {
    renderX: number
    renderY: number
}

const BOARD_INSET = 6
const CARD_GAP = 4
const ROTATION_PADDING = 4
const PLACEMENT_ATTEMPTS = 6000

const props = defineProps<{
    cards: ExerciseMatchCard[]
    selectedCardIds: string[]
    vocabularyStates: Record<string, MatchVocabularyState>
    cardWrongAttempts: Record<string, number>
    disabled: boolean
    isSubmitting: boolean
    checkingText: string
    boardLabel: string
}>()

const emit = defineEmits<{
    choose: [card: ExerciseMatchCard]
}>()

const boardRef = ref<HTMLElement | null>(null)
const boardSize = ref(0)
const cardWidth = ref(112)
const cardLayouts = ref<Record<string, MatchCardLayout>>({})
let resizeObserver: ResizeObserver | null = null

const boardStyle = computed(() => ({
    '--match-card-width': `${cardWidth.value}px`,
}))

function clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, value))
}

function createSeededRandom(seedInput: string): () => number {
    let seed = 2166136261
    for (let index = 0; index < seedInput.length; index++) {
        seed ^= seedInput.charCodeAt(index)
        seed = Math.imul(seed, 16777619)
    }

    return () => {
        seed += 0x6d2b79f5
        let value = seed
        value = Math.imul(value ^ (value >>> 15), value | 1)
        value ^= value + Math.imul(value ^ (value >>> 7), value | 61)
        return ((value ^ (value >>> 14)) >>> 0) / 4294967296
    }
}

function estimateCardHeight(card: ExerciseMatchCard, width: number): number {
    const contentWidth = Math.max(1, width - 24)
    const averageCharacterWidth = 8
    const charactersPerLine = clamp(Math.floor(contentWidth / averageCharacterWidth), 6, 18)
    const lines = clamp(Math.ceil(card.word.length / charactersPerLine), 1, 5)
    return Math.max(64, 34 + lines * 19)
}

function isInsideCircle(x: number, y: number, width: number, height: number, size: number): boolean {
    const radius = size / 2 - BOARD_INSET
    const corners: Array<[number, number]> = [
        [x - width / 2, y - height / 2],
        [x + width / 2, y - height / 2],
        [x - width / 2, y + height / 2],
        [x + width / 2, y + height / 2],
    ]

    return corners.every(([cornerX, cornerY]) => {
        const dx = cornerX - size / 2
        const dy = cornerY - size / 2
        return dx * dx + dy * dy <= radius * radius
    })
}

function overlapsPlacedCards(candidate: PlacedCard, placed: PlacedCard[]): boolean {
    return placed.some((item) => {
        const horizontalClearance = (candidate.width + item.width) / 2 + CARD_GAP
        const verticalClearance = (candidate.height + item.height) / 2 + CARD_GAP
        return (
            Math.abs(candidate.x - item.x) < horizontalClearance && Math.abs(candidate.y - item.y) < verticalClearance
        )
    })
}

function getCircleOverflow(candidate: PlacedCard, size: number): number {
    const radius = size / 2 - BOARD_INSET
    const maxDistanceSquared = radius * radius
    const corners: Array<[number, number]> = [
        [candidate.x - candidate.width / 2, candidate.y - candidate.height / 2],
        [candidate.x + candidate.width / 2, candidate.y - candidate.height / 2],
        [candidate.x - candidate.width / 2, candidate.y + candidate.height / 2],
        [candidate.x + candidate.width / 2, candidate.y + candidate.height / 2],
    ]

    return corners.reduce((overflow, [cornerX, cornerY]) => {
        const dx = cornerX - size / 2
        const dy = cornerY - size / 2
        return overflow + Math.max(0, dx * dx + dy * dy - maxDistanceSquared)
    }, 0)
}

function getOverlapPenalty(candidate: PlacedCard, placed: PlacedCard[]): number {
    return placed.reduce((penalty, item) => {
        const overlapX = (candidate.width + item.width) / 2 + CARD_GAP - Math.abs(candidate.x - item.x)
        const overlapY = (candidate.height + item.height) / 2 + CARD_GAP - Math.abs(candidate.y - item.y)
        if (overlapX <= 0 || overlapY <= 0) return penalty
        return penalty + overlapX * overlapY
    }, 0)
}

function getPlacementScore(candidate: PlacedCard, placed: PlacedCard[], size: number): number {
    return getOverlapPenalty(candidate, placed) + getCircleOverflow(candidate, size) * 10
}

function fallbackPosition(
    index: number,
    total: number,
    size: number,
    width: number,
    height: number
): { x: number; y: number } {
    const radius = size / 2
    const diagonal = Math.sqrt(width * width + height * height) / 2
    const maxRadius = Math.max(0, radius - diagonal - 8)
    const ringRadius = maxRadius * (index % 3 === 0 ? 0.42 : index % 3 === 1 ? 0.72 : 0.92)
    const angle = -Math.PI / 2 + (index / total) * Math.PI * 2

    return {
        x: radius + Math.cos(angle) * ringRadius,
        y: radius + Math.sin(angle) * ringRadius,
    }
}

function generateMatchCardLayouts() {
    const size = boardSize.value
    if (size <= 0 || props.cards.length === 0) {
        cardLayouts.value = {}
        return
    }

    const width = Math.round(clamp(size * 0.2, 60, 140))
    cardWidth.value = width

    const random = createSeededRandom(props.cards.map((card) => card.id).join('|'))
    const layouts: Record<string, MatchCardLayout> = {}
    const placed: PlacedCard[] = []

    props.cards.forEach((card, index) => {
        const height = estimateCardHeight(card, width)
        const placementWidth = width + ROTATION_PADDING
        const placementHeight = height + ROTATION_PADDING
        const radius = size / 2
        const diagonal = Math.sqrt(placementWidth * placementWidth + placementHeight * placementHeight) / 2
        const maxRadius = Math.max(0, radius - diagonal - 8)
        const fallback = fallbackPosition(index, props.cards.length, size, placementWidth, placementHeight)
        let selected: PlacementCandidate = {
            x: fallback.x,
            y: fallback.y,
            renderX: fallback.x,
            renderY: fallback.y,
            width: placementWidth,
            height: placementHeight,
        }
        let selectedScore = getPlacementScore(selected, placed, size)

        const considerCandidate = (x: number, y: number): boolean => {
            const candidate = {
                x,
                y,
                renderX: x,
                renderY: y,
                width: placementWidth,
                height: placementHeight,
            }
            const score = getPlacementScore(candidate, placed, size)

            if (score < selectedScore) {
                selected = candidate
                selectedScore = score
            }

            if (
                isInsideCircle(candidate.x, candidate.y, candidate.width, candidate.height, size) &&
                !overlapsPlacedCards(candidate, placed)
            ) {
                selected = candidate
                selectedScore = 0
                return true
            }

            return false
        }

        for (let attempt = 0; attempt < PLACEMENT_ATTEMPTS; attempt++) {
            const angle = random() * Math.PI * 2
            const distance = Math.sqrt(random()) * maxRadius
            if (considerCandidate(radius + Math.cos(angle) * distance, radius + Math.sin(angle) * distance)) {
                break
            }
        }

        if (selectedScore > 0) {
            const ringCount = 6
            const stepsPerRing = Math.max(18, props.cards.length * 5)
            const angleOffset = index * 2.399963229728653

            for (let ring = 0; ring < ringCount && selectedScore > 0; ring++) {
                const distance = maxRadius * ((ring + 1) / ringCount)

                for (let step = 0; step < stepsPerRing; step++) {
                    const angle = angleOffset + (step / stepsPerRing) * Math.PI * 2
                    if (considerCandidate(radius + Math.cos(angle) * distance, radius + Math.sin(angle) * distance)) {
                        break
                    }
                }
            }
        }

        placed.push({ x: selected.x, y: selected.y, width: placementWidth, height: placementHeight })
        layouts[card.id] = {
            x: Math.round(selected.renderX),
            y: Math.round(selected.renderY),
            rotation: Math.round((random() * 10 - 5) * 10) / 10,
        }
    })

    cardLayouts.value = layouts
}

function updateBoardSize() {
    const rect = boardRef.value?.getBoundingClientRect()
    const size = rect ? Math.min(rect.width, rect.height) : 0
    if (size > 0 && Math.abs(size - boardSize.value) > 1) {
        boardSize.value = size
        generateMatchCardLayouts()
    }
}

function getMatchCardVisualState(card: ExerciseMatchCard): MatchCardVisualState {
    const state = props.vocabularyStates[card.vocabulary_id]
    if (!state) return 'idle'
    if (state.result === 'wrong') return 'red'
    if (state.result === 'correct' || state.result === 'almost') return 'green'
    if (props.selectedCardIds.includes(card.id)) return 'selected'
    if ((props.cardWrongAttempts[card.id] ?? 0) > 0) return 'yellow'
    return 'idle'
}

function getMatchCardClass(card: ExerciseMatchCard): string {
    const visualState = getMatchCardVisualState(card)
    if (visualState === 'green') return 'quiz-match-card--green'
    if (visualState === 'yellow') return 'quiz-match-card--yellow'
    if (visualState === 'red') return 'quiz-match-card--red'
    if (visualState === 'selected') return 'quiz-match-card--selected'
    return 'quiz-match-card--idle'
}

function isMatchCardResolved(card: ExerciseMatchCard): boolean {
    const result = props.vocabularyStates[card.vocabulary_id]?.result
    return result === 'correct' || result === 'almost' || result === 'wrong'
}

watch(
    () => props.cards.map((card) => card.id).join('|'),
    async () => {
        await nextTick()
        updateBoardSize()
    }
)

onMounted(async () => {
    await nextTick()
    updateBoardSize()

    if (boardRef.value && typeof ResizeObserver !== 'undefined') {
        resizeObserver = new ResizeObserver(() => updateBoardSize())
        resizeObserver.observe(boardRef.value)
    }
})

onBeforeUnmount(() => {
    resizeObserver?.disconnect()
})
</script>

<template>
    <div
        ref="boardRef"
        class="quiz-match-board relative mx-auto aspect-square w-full max-w-[680px]"
        role="group"
        :aria-label="boardLabel"
        :style="boardStyle"
    >
        <button
            v-for="card in cards"
            :key="card.id"
            type="button"
            :disabled="disabled || isMatchCardResolved(card)"
            :class="getMatchCardClass(card)"
            :style="{
                left: `${cardLayouts[card.id]?.x ?? boardSize / 2}px`,
                top: `${cardLayouts[card.id]?.y ?? boardSize / 2}px`,
                transform: `translate(-50%, -50%) rotate(${cardLayouts[card.id]?.rotation ?? 0}deg)`,
            }"
            class="quiz-match-card absolute flex min-h-14 items-center justify-center rounded-md border px-3 py-2 text-center text-sm font-semibold leading-tight shadow-sm transition-[background-color,border-color,box-shadow,filter,transform] duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-default sm:min-h-16 sm:px-3.5 sm:py-2.5"
            @click="emit('choose', card)"
        >
            <span class="w-full min-w-0 break-words">{{ card.word }}</span>
        </button>

        <div
            v-if="isSubmitting"
            class="absolute inset-x-0 bottom-8 mx-auto flex w-fit items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-sm text-muted-foreground shadow-sm"
        >
            <span
                class="quiz-inline-spinner h-4 w-4 rounded-full border-2 border-muted-foreground/35 border-t-muted-foreground"
                aria-hidden="true"
            ></span>
            {{ checkingText }}
        </div>
    </div>
</template>

<style scoped>
.quiz-match-card {
    width: var(--match-card-width);
    color: hsl(var(--foreground));
    background: hsl(var(--background));
    border-color: hsl(var(--border));
}

.quiz-match-card--idle:hover:not(:disabled) {
    border-color: color-mix(in oklab, hsl(var(--primary)) 42%, hsl(var(--border)));
    box-shadow: 0 14px 28px -24px hsl(var(--primary));
    filter: brightness(1.02);
}

.quiz-match-card--selected {
    border-color: color-mix(in oklab, hsl(var(--primary)) 65%, hsl(var(--border)));
    background: color-mix(in oklab, hsl(var(--primary)) 12%, hsl(var(--background)));
    box-shadow:
        0 0 0 2px color-mix(in oklab, hsl(var(--primary)) 18%, transparent),
        0 16px 28px -24px hsl(var(--primary));
}

.quiz-match-card--green {
    border-color: color-mix(in oklab, hsl(var(--success)) 55%, hsl(var(--border)));
    background: color-mix(in oklab, hsl(var(--success)) 14%, hsl(var(--background)));
    color: color-mix(in oklab, hsl(var(--success)) 76%, hsl(var(--foreground)));
}

.quiz-match-card--yellow {
    border-color: color-mix(in oklab, hsl(var(--warning)) 60%, hsl(var(--border)));
    background: color-mix(in oklab, hsl(var(--warning)) 16%, hsl(var(--background)));
    color: color-mix(in oklab, hsl(var(--warning)) 76%, hsl(var(--foreground)));
}

.quiz-match-card--red {
    border-color: color-mix(in oklab, hsl(var(--destructive)) 58%, hsl(var(--border)));
    background: color-mix(in oklab, hsl(var(--destructive)) 14%, hsl(var(--background)));
    color: color-mix(in oklab, hsl(var(--destructive)) 78%, hsl(var(--foreground)));
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
    .quiz-match-card {
        transition: none;
    }

    .quiz-inline-spinner {
        animation: none;
    }
}
</style>
