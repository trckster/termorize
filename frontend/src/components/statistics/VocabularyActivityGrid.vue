<script setup lang="ts">
import { computed } from 'vue'
import type { VocabularyDailyActivity } from '@/api/exercises.ts'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

const props = defineProps<{
    activity: VocabularyDailyActivity[]
    locale: string
    vocabularyLabel: string
    lessLabel: string
    moreLabel: string
}>()

const monthFormatter = computed(() => new Intl.DateTimeFormat(props.locale, { month: 'long', year: 'numeric' }))
const dateFormatter = computed(
    () => new Intl.DateTimeFormat(props.locale, { weekday: 'short', day: 'numeric', month: 'long', year: 'numeric' })
)

const months = computed(() => {
    const grouped = new Map<string, VocabularyDailyActivity[]>()
    for (const day of props.activity) {
        const monthKey = day.date.slice(0, 7)
        grouped.set(monthKey, [...(grouped.get(monthKey) ?? []), day])
    }

    return Array.from(grouped.entries())
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, days]) => {
            const firstDate = new Date(`${key}-01T00:00:00`)
            const mondayOffset = (firstDate.getDay() + 6) % 7
            return {
                key,
                label: monthFormatter.value.format(firstDate),
                days: [...Array.from({ length: mondayOffset }, () => null), ...days],
            }
        })
})

const maximum = computed(() => Math.max(0, ...props.activity.map((day) => day.count)))
const intensity = (count: number) => {
    if (count === 0 || maximum.value === 0) return 0
    return Math.max(1, Math.ceil((count / maximum.value) * 4))
}

const intensityClasses = [
    'border-border/70 bg-muted/65',
    'border-primary/10 bg-primary/20',
    'border-primary/15 bg-primary/40',
    'border-primary/20 bg-primary/65',
    'border-primary/30 bg-primary/90',
]
const intensityClass = (count: number) => intensityClasses[intensity(count)]

const formatDate = (date: string) => dateFormatter.value.format(new Date(`${date}T00:00:00`))
</script>

<template>
    <div class="overflow-x-auto pb-2">
        <div class="grid w-full grid-cols-2 gap-3 md:flex md:min-w-[990px] md:justify-between md:gap-6">
            <section
                v-for="month in months"
                :key="month.key"
                class="min-w-0 max-md:[&:nth-last-child(n+3)]:hidden md:w-[145px] md:shrink-0"
                :aria-label="month.label"
            >
                <h3 class="mb-3 text-center text-sm font-medium capitalize text-foreground">{{ month.label }}</h3>
                <div
                    class="grid grid-flow-col auto-cols-4 grid-rows-7 justify-center gap-1 min-[360px]:auto-cols-[18px] md:auto-cols-5 md:gap-[5px]"
                >
                    <template v-for="(day, index) in month.days" :key="day?.date ?? `blank-${index}`">
                        <span
                            v-if="!day"
                            aria-hidden="true"
                            class="h-4 w-4 min-[360px]:h-[18px] min-[360px]:w-[18px] md:h-5 md:w-5"
                        />
                        <Tooltip v-else :delay-duration="100">
                            <TooltipTrigger as-child>
                                <button
                                    type="button"
                                    class="h-4 w-4 rounded-[4px] border outline-none transition-[transform,box-shadow] duration-200 hover:scale-110 focus-visible:scale-110 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background min-[360px]:h-[18px] min-[360px]:w-[18px] md:h-5 md:w-5 md:rounded-[5px]"
                                    :class="intensityClass(day.count)"
                                    :aria-label="`${formatDate(day.date)}: ${day.count} ${vocabularyLabel}`"
                                />
                            </TooltipTrigger>
                            <TooltipContent side="top" class="text-xs">
                                <p class="font-medium">{{ day.count }} {{ vocabularyLabel }}</p>
                                <p class="text-muted-foreground">{{ formatDate(day.date) }}</p>
                            </TooltipContent>
                        </Tooltip>
                    </template>
                </div>
            </section>
        </div>
    </div>

    <div class="mt-4 flex items-center justify-end gap-2 text-xs text-muted-foreground" aria-hidden="true">
        <span>{{ lessLabel }}</span>
        <span
            v-for="level in 5"
            :key="level"
            class="h-3 w-3 rounded-[3px] border"
            :class="intensityClasses[level - 1]"
        />
        <span>{{ moreLabel }}</span>
    </div>
</template>
