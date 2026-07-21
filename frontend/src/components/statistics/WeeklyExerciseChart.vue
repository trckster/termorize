<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ExerciseDailyActivity } from '@/api/exercises.ts'

const props = defineProps<{
    activity: ExerciseDailyActivity[]
    locale: string
    completedLabel: string
    failedLabel: string
    tasksLabel: string
}>()

const hoveredIndex = ref<number | null>(null)

const chart = {
    width: 760,
    height: 250,
    left: 48,
    right: 16,
    top: 18,
    bottom: 48,
}

const plotWidth = chart.width - chart.left - chart.right
const plotHeight = chart.height - chart.top - chart.bottom

const yMax = computed(() => {
    const maximum = Math.max(0, ...props.activity.flatMap((day) => [day.completed, day.failed]))
    return Math.max(4, Math.ceil(maximum / 4) * 4)
})

const yTicks = computed(() => Array.from({ length: 5 }, (_, index) => (yMax.value / 4) * index).reverse())
const groupWidth = computed(() => plotWidth / Math.max(props.activity.length, 1))
const barWidth = computed(() => Math.min(22, Math.max(9, groupWidth.value * 0.24)))

const xCenter = (index: number) => chart.left + groupWidth.value * (index + 0.5)
const barHeight = (value: number) => (value / yMax.value) * plotHeight
const yPosition = (value: number) => chart.top + plotHeight - barHeight(value)

const dateFormatter = computed(() => new Intl.DateTimeFormat(props.locale, { day: 'numeric', month: 'short' }))
const longDateFormatter = computed(
    () => new Intl.DateTimeFormat(props.locale, { weekday: 'short', day: 'numeric', month: 'long' })
)
const formatDate = (date: string, long = false) =>
    (long ? longDateFormatter.value : dateFormatter.value).format(new Date(`${date}T00:00:00`))

const activeDay = computed(() => (hoveredIndex.value === null ? null : props.activity[hoveredIndex.value]))
</script>

<template>
    <div class="relative" @mouseleave="hoveredIndex = null">
        <div
            v-if="activeDay"
            class="pointer-events-none absolute left-1/2 top-1 z-10 -translate-x-1/2 rounded-lg border border-border bg-popover/95 px-3 py-2 text-xs shadow-md"
        >
            <p class="mb-1.5 font-medium text-popover-foreground">{{ formatDate(activeDay.date, true) }}</p>
            <div class="flex gap-3 text-muted-foreground">
                <span
                    ><i class="mr-1 inline-block h-2 w-2 rounded-full bg-success" />{{ activeDay.completed }}
                    {{ completedLabel.toLowerCase() }}</span
                >
                <span
                    ><i class="mr-1 inline-block h-2 w-2 rounded-full bg-destructive" />{{ activeDay.failed }}
                    {{ failedLabel.toLowerCase() }}</span
                >
            </div>
        </div>

        <svg
            class="h-auto w-full min-w-[560px]"
            :viewBox="`0 0 ${chart.width} ${chart.height}`"
            role="img"
            :aria-label="`${completedLabel} and ${failedLabel}, ${tasksLabel}`"
        >
            <g v-for="tick in yTicks" :key="tick">
                <line
                    :x1="chart.left"
                    :x2="chart.width - chart.right"
                    :y1="yPosition(tick)"
                    :y2="yPosition(tick)"
                    class="stroke-border/70"
                    stroke-dasharray="3 5"
                />
                <text
                    :x="chart.left - 12"
                    :y="yPosition(tick) + 4"
                    text-anchor="end"
                    class="fill-muted-foreground text-[11px]"
                >
                    {{ tick }}
                </text>
            </g>

            <g v-for="(day, index) in activity" :key="day.date">
                <rect
                    :x="xCenter(index) - barWidth - 2"
                    :y="yPosition(day.completed)"
                    :width="barWidth"
                    :height="Math.max(barHeight(day.completed), day.completed > 0 ? 2 : 0)"
                    :rx="barWidth / 3"
                    class="fill-success/80 transition-opacity duration-200"
                    :class="hoveredIndex !== null && hoveredIndex !== index ? 'opacity-35' : ''"
                />
                <rect
                    :x="xCenter(index) + 2"
                    :y="yPosition(day.failed)"
                    :width="barWidth"
                    :height="Math.max(barHeight(day.failed), day.failed > 0 ? 2 : 0)"
                    :rx="barWidth / 3"
                    class="fill-destructive/75 transition-opacity duration-200"
                    :class="hoveredIndex !== null && hoveredIndex !== index ? 'opacity-35' : ''"
                />
                <text
                    :x="xCenter(index)"
                    :y="chart.height - 18"
                    text-anchor="middle"
                    class="fill-muted-foreground text-[11px]"
                >
                    {{ formatDate(day.date) }}
                </text>
                <rect
                    :x="xCenter(index) - groupWidth / 2"
                    :y="chart.top"
                    :width="groupWidth"
                    :height="plotHeight + 32"
                    fill="transparent"
                    tabindex="0"
                    class="outline-none focus-visible:stroke-ring focus-visible:stroke-2"
                    @mouseenter="hoveredIndex = index"
                    @focus="hoveredIndex = index"
                    @blur="hoveredIndex = null"
                >
                    <title>
                        {{
                            `${formatDate(day.date, true)}: ${day.completed} ${completedLabel.toLowerCase()}, ${day.failed} ${failedLabel.toLowerCase()}`
                        }}
                    </title>
                </rect>
            </g>
        </svg>
    </div>
</template>
