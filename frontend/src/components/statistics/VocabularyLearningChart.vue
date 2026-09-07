<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import type { VocabularyLearningDistribution } from '@/api/exercises.ts'
import { useI18n } from '@/composables/useI18n'
import { createPieSegments } from '@/lib/pieChart'

const props = defineProps<{
    distribution: VocabularyLearningDistribution
    loading: boolean
    unavailable: boolean
}>()
const { t, locale } = useI18n()
const id = useId()
const root = ref<HTMLElement | null>(null)
const activeKey = ref<string | null>(null)
const tabKey = ref<string | null>(null)
const interaction = ref<'pointer' | 'keyboard' | 'touch'>('pointer')
const number = computed(() => new Intl.NumberFormat(locale.value))
const percent = computed(() => new Intl.NumberFormat(locale.value, { style: 'percent', maximumFractionDigits: 1 }))
const groups = computed(
    () =>
        [
            { key: 'not_started', label: t.value.statisticsLearningNotStarted, range: '0%' },
            { key: 'beginning', label: t.value.statisticsLearningBeginning, range: '1–34%' },
            { key: 'developing', label: t.value.statisticsLearningDeveloping, range: '35–69%' },
            { key: 'confident', label: t.value.statisticsLearningConfident, range: '70–99%' },
            { key: 'mastered', label: t.value.statisticsLearningMastered, range: '100%' },
        ] as const
)
const segments = computed(() => {
    const geometry = createPieSegments(groups.value.map((group) => props.distribution[group.key]))
    return groups.value.map((group, index) => ({ ...group, ...geometry[index]! }))
})
const visible = computed(() => segments.value.filter((segment) => segment.count > 0))
const total = computed(() => segments.value.reduce((sum, segment) => sum + segment.count, 0))
const active = computed(() => segments.value.find((segment) => segment.key === activeKey.value))
const entryKey = computed(() =>
    visible.value.some((segment) => segment.key === tabKey.value) ? tabKey.value : visible.value[0]?.key
)
const shareLabel = (share: number) =>
    share > 0 && share < 0.001
        ? `<${percent.value.format(0.001)}`
        : share < 1 && share > 0.999
          ? `>${percent.value.format(0.999)}`
          : percent.value.format(share)
const accessibleLabel = (segment: (typeof segments.value)[number]) =>
    `${segment.label}, ${segment.range}: ${number.value.format(segment.count)}; ${shareLabel(segment.share)} ${t.value.statisticsLearningShare}`
const summary = computed(() => segments.value.map(accessibleLabel).join('; '))

// Anchor to the slice, not the cursor: movement within a slice stays calm.
// Clamp horizontally to keep the panel inside even a narrow mobile chart.
const tooltipStyle = computed(() => {
    if (interaction.value === 'touch') return {}
    const anchor = active.value?.anchor ?? { x: 140, y: 140 }
    return {
        left: `clamp(0px, calc(${(anchor.x / 280) * 100}% - 108px), calc(100% - 216px))`,
        top: `${(anchor.y / 280) * 100}%`,
        transform: `translateY(${anchor.y < 140 ? '16px' : 'calc(-100% - 16px)'})`,
    }
})
const dismiss = () => {
    activeKey.value = null
}
const show = (key: string, mode: typeof interaction.value) => {
    interaction.value = mode
    activeKey.value = key
    tabKey.value = key
}
const leave = () => {
    if (interaction.value === 'pointer') dismiss()
}
const focus = (event: FocusEvent, key: string) => {
    if ((event.target as SVGElement).matches(':focus-visible')) show(key, 'keyboard')
}
const blur = (event: FocusEvent) => {
    if (interaction.value === 'keyboard' && !root.value?.contains(event.relatedTarget as Node | null)) dismiss()
}
const navigate = (event: KeyboardEvent, key: string) => {
    const index = visible.value.findIndex((segment) => segment.key === key)
    let next: number
    switch (event.key) {
        case 'ArrowRight':
        case 'ArrowDown':
            next = (index + 1) % visible.value.length
            break
        case 'ArrowLeft':
        case 'ArrowUp':
            next = (index + visible.value.length - 1) % visible.value.length
            break
        case 'Home':
            next = 0
            break
        case 'End':
            next = visible.value.length - 1
            break
        case 'Enter':
        case ' ':
            event.preventDefault()
            show(key, 'keyboard')
            return
        default:
            return
    }
    event.preventDefault()
    const target = visible.value[next]!
    show(target.key, 'keyboard')
    root.value?.querySelector<SVGElement>(`[data-slice="${target.key}"]`)?.focus()
}
const stepTouch = (direction: number) => {
    const index = segments.value.findIndex((segment) => segment.key === activeKey.value)
    show(segments.value[(index + direction + segments.value.length) % segments.value.length]!.key, 'touch')
}
const outside = (event: PointerEvent) => {
    if (!root.value?.contains(event.target as Node)) dismiss()
}
watch(() => [props.distribution, props.loading, props.unavailable], dismiss, { deep: true })
onMounted(() => document.addEventListener('pointerdown', outside))
onBeforeUnmount(() => {
    document.removeEventListener('pointerdown', outside)
})
</script>

<template>
    <section
        ref="root"
        class="learning-card flex min-w-0 flex-col rounded-xl border border-border bg-card"
        :aria-labelledby="`${id}-title`"
        :aria-busy="loading"
        @keydown.esc.stop="dismiss"
    >
        <header class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 px-5 pt-5">
            <h2 :id="`${id}-title`" class="text-base font-semibold text-foreground">{{ t.statisticsLearningTitle }}</h2>
            <span v-if="!loading && !unavailable" class="text-xs tabular-nums text-muted-foreground"
                >{{ t.statisticsLearningWords }} · {{ number.format(total) }}</span
            >
        </header>
        <div v-if="loading" class="flex flex-1 items-center justify-center p-6" aria-hidden="true">
            <div
                class="aspect-square w-full max-w-[248px] animate-pulse rounded-full bg-muted motion-reduce:animate-none"
            />
        </div>
        <p v-else-if="unavailable" class="m-auto px-5 py-12 text-center text-sm text-muted-foreground">
            {{ t.statisticsLearningUnavailable }}
        </p>
        <div v-else class="flex flex-1 flex-col justify-center px-5 pb-5 pt-3">
            <p :id="`${id}-instructions`" class="sr-only">{{ t.statisticsLearningKeyboard }} {{ summary }}</p>
            <div class="relative mx-auto w-full max-w-[300px]">
                <svg
                    viewBox="0 0 280 280"
                    class="block aspect-square w-full overflow-visible"
                    role="group"
                    :aria-label="t.statisticsLearningTitle"
                    :aria-describedby="`${id}-instructions`"
                    @pointerleave="leave"
                >
                    <circle v-if="total === 0" cx="140" cy="140" r="124" class="fill-muted" />
                    <path
                        v-for="segment in visible"
                        :key="segment.key"
                        :data-slice="segment.key"
                        :d="segment.path"
                        :fill="`var(--learning-${segment.key})`"
                        :tabindex="entryKey === segment.key ? 0 : -1"
                        role="button"
                        :aria-label="accessibleLabel(segment)"
                        :aria-describedby="activeKey === segment.key ? `${id}-details` : undefined"
                        :class="{
                            'is-active': activeKey === segment.key,
                            'is-muted': activeKey && activeKey !== segment.key,
                        }"
                        class="learning-segment"
                        @pointerenter="
                            (event) => {
                                if (event.pointerType !== 'touch') show(segment.key, 'pointer')
                            }
                        "
                        @pointerleave="leave"
                        @pointerdown="
                            (event) => {
                                if (event.pointerType === 'touch') show(segment.key, 'touch')
                            }
                        "
                        @focus="focus($event, segment.key)"
                        @blur="blur"
                        @keydown="navigate($event, segment.key)"
                        @click="
                            (event) => {
                                if (event.detail === 0) show(segment.key, 'keyboard')
                            }
                        "
                    />
                    <!-- A thin outer contour sharpens the silhouette without hiding tiny slices. -->
                    <circle
                        v-if="total"
                        cx="140"
                        cy="140"
                        r="124"
                        fill="none"
                        class="pointer-events-none stroke-foreground/10"
                        stroke-width="0.75"
                        aria-hidden="true"
                    />
                </svg>
                <Transition name="learning-detail">
                    <div
                        v-if="active"
                        :id="`${id}-details`"
                        :role="interaction === 'touch' ? 'group' : 'tooltip'"
                        :aria-label="active.label"
                        :style="tooltipStyle"
                        :class="interaction === 'touch' ? 'relative mx-auto mt-2' : 'pointer-events-none absolute'"
                        class="learning-tooltip z-20 w-[216px] rounded-xl border border-border bg-popover p-3.5 text-popover-foreground"
                    >
                        <div class="flex items-center gap-2">
                            <span
                                class="h-2.5 w-2.5 shrink-0 rounded-full"
                                :style="{ backgroundColor: `var(--learning-${active.key})` }"
                            />
                            <span class="text-sm font-semibold">{{ active.label }}</span>
                        </div>
                        <p class="mt-1 text-xs text-muted-foreground">
                            {{ t.statisticsLearningRange }} · {{ active.range }}
                        </p>
                        <div class="mt-3 flex items-end justify-between gap-2 border-t border-border pt-3">
                            <div>
                                <p class="text-2xl font-semibold leading-none tracking-tight tabular-nums">
                                    {{ number.format(active.count) }}
                                </p>
                                <p class="mt-1 text-xs text-muted-foreground">{{ t.statisticsLearningWords }}</p>
                            </div>
                            <div class="text-right">
                                <p class="text-sm font-semibold tabular-nums">{{ shareLabel(active.share) }}</p>
                                <p class="text-xs text-muted-foreground">{{ t.statisticsLearningShare }}</p>
                            </div>
                        </div>
                        <div
                            v-if="interaction === 'touch'"
                            class="-mx-1.5 -mb-1.5 mt-2 flex justify-between border-t border-border pt-1"
                        >
                            <button
                                type="button"
                                class="flex h-11 w-11 items-center justify-center rounded-lg hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                :aria-label="t.statisticsLearningPrevious"
                                @click="stepTouch(-1)"
                            >
                                <ChevronLeft class="h-4 w-4" />
                            </button>
                            <button
                                type="button"
                                class="flex h-11 w-11 items-center justify-center rounded-lg hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                :aria-label="t.statisticsLearningNext"
                                @click="stepTouch(1)"
                            >
                                <ChevronRight class="h-4 w-4" />
                            </button>
                        </div>
                    </div>
                </Transition>
            </div>
            <div v-if="total === 0" class="mt-3 text-center text-sm">
                <p class="font-medium text-foreground">{{ t.statisticsLearningEmpty }}</p>
                <p class="mt-1 leading-6 text-muted-foreground">{{ t.statisticsLearningEmptyDescription }}</p>
            </div>
        </div>
    </section>
</template>

<style scoped>
.learning-card {
    --learning-not_started: #a3b4b9;
    --learning-beginning: #a0d7ed;
    --learning-developing: #44acd4;
    --learning-confident: #306eaf;
    --learning-mastered: #10a477;
}
:global(.dark .learning-card) {
    --learning-not_started: #63777e;
    --learning-beginning: #abdff0;
    --learning-developing: #51b8df;
    --learning-confident: #3f7fc5;
    --learning-mastered: #24c493;
}
.learning-segment {
    cursor: pointer;
    outline: none;
    transition:
        opacity 160ms ease-out,
        filter 160ms ease-out;
}
.learning-segment.is-muted {
    opacity: 0.72;
}
.learning-segment.is-active {
    filter: brightness(1.06);
}
.learning-segment:focus-visible {
    stroke: hsl(var(--foreground));
    stroke-width: 2;
    stroke-linejoin: round;
    paint-order: stroke fill;
}
.learning-tooltip {
    box-shadow:
        0 8px 24px -6px hsl(var(--overlay) / 0.22),
        0 2px 6px hsl(var(--overlay) / 0.08);
}
.learning-detail-enter-active,
.learning-detail-leave-active {
    transition: opacity 120ms ease-out;
}
.learning-detail-enter-from,
.learning-detail-leave-to {
    opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
    .learning-segment,
    .learning-detail-enter-active,
    .learning-detail-leave-active {
        transition: none;
    }
}
</style>
