<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { exercisesApi, type ExerciseStatistics } from '@/api/exercises.ts'
import { Activity, AlertCircle, Ban, CheckCircle2 } from 'lucide-vue-next'
import { useI18n } from '@/composables/useI18n'
import { formatNumber } from '@/lib/utils.ts'
import WeeklyExerciseChart from '@/components/statistics/WeeklyExerciseChart.vue'
import VocabularyActivityGrid from '@/components/statistics/VocabularyActivityGrid.vue'

const { t, locale } = useI18n()

const statistics = ref<ExerciseStatistics>({
    in_progress: 0,
    done: 0,
    failed: 0,
    ignored: 0,
    exercise_activity: [],
    vocabulary_activity: [],
})
const isLoading = ref(true)
const errorMessage = ref('')

const totalExercises = computed(() => {
    return statistics.value.in_progress + statistics.value.done + statistics.value.failed + statistics.value.ignored
})

const statisticCards = computed(() => [
    {
        key: 'in_progress',
        label: t.value.exerciseStatInProgress,
        description: t.value.exerciseStatInProgressDesc,
        value: statistics.value.in_progress,
        icon: Activity,
        accentClass: 'text-info',
    },
    {
        key: 'done',
        label: t.value.exerciseStatDone,
        description: t.value.exerciseStatDoneDesc,
        value: statistics.value.done,
        icon: CheckCircle2,
        accentClass: 'text-success',
    },
    {
        key: 'failed',
        label: t.value.exerciseStatFailed,
        description: t.value.exerciseStatFailedDesc,
        value: statistics.value.failed,
        icon: AlertCircle,
        accentClass: 'text-destructive',
    },
    {
        key: 'ignored',
        label: t.value.exerciseStatIgnored,
        description: t.value.exerciseStatIgnoredDesc,
        value: statistics.value.ignored,
        icon: Ban,
        accentClass: 'text-warning',
    },
])

const fetchStatistics = async () => {
    isLoading.value = true
    errorMessage.value = ''

    try {
        statistics.value = await exercisesApi.getStatistics()
    } catch {
        errorMessage.value = t.value.statisticsErrorMessage
    } finally {
        isLoading.value = false
    }
}

onMounted(() => {
    void fetchStatistics()
})
</script>

<template>
    <main class="px-4 py-6 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-6xl space-y-8">
            <header class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <h1 class="text-2xl font-semibold tracking-tight text-foreground">{{ t.statisticsHeading }}</h1>
                    <p class="mt-1.5 max-w-2xl text-sm leading-6 text-muted-foreground">
                        {{ t.statisticsDescription }}
                    </p>
                </div>

                <dl class="shrink-0 sm:text-right">
                    <dt class="text-xs font-medium text-muted-foreground">{{ t.statisticsTracked }}</dt>
                    <dd class="mt-1 text-2xl font-semibold tabular-nums text-foreground">
                        <span
                            v-if="isLoading"
                            class="inline-block h-7 w-12 animate-pulse rounded bg-muted motion-reduce:animate-none"
                        />
                        <template v-else>{{ formatNumber(totalExercises) }}</template>
                    </dd>
                </dl>
            </header>

            <div
                v-if="errorMessage"
                role="alert"
                class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive"
            >
                <span>{{ errorMessage }}</span>
                <button
                    type="button"
                    class="font-medium underline underline-offset-4 hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    :disabled="isLoading"
                    @click="fetchStatistics"
                >
                    {{ t.commonRetry }}
                </button>
            </div>

            <section
                :aria-label="t.statisticsLabel"
                class="grid overflow-hidden rounded-xl border border-border bg-card sm:grid-cols-2 xl:grid-cols-4 xl:divide-x xl:divide-y-0"
            >
                <div
                    v-for="item in statisticCards"
                    :key="item.key"
                    class="flex gap-3 border-b border-border p-4 last:border-b-0 sm:[&:nth-child(3)]:border-b-0 xl:border-b-0 xl:p-5"
                >
                    <component :is="item.icon" class="mt-0.5 h-4 w-4 shrink-0" :class="item.accentClass" />
                    <div class="min-w-0">
                        <div class="flex items-baseline gap-2">
                            <p class="text-sm font-medium text-foreground">{{ item.label }}</p>
                            <span
                                v-if="isLoading"
                                class="inline-block h-6 w-8 animate-pulse rounded bg-muted motion-reduce:animate-none"
                            />
                            <p v-else class="text-xl font-semibold tabular-nums text-foreground">
                                {{ formatNumber(item.value) }}
                            </p>
                        </div>
                        <p class="mt-1 text-xs leading-5 text-muted-foreground">{{ item.description }}</p>
                    </div>
                </div>
            </section>

            <section class="overflow-hidden rounded-xl border border-border bg-card">
                <header class="flex flex-col gap-4 px-4 py-4 sm:flex-row sm:items-start sm:justify-between sm:px-5">
                    <div>
                        <h2 class="text-base font-semibold text-foreground">{{ t.statisticsWeeklyTitle }}</h2>
                        <p class="mt-1 max-w-2xl text-sm leading-5 text-muted-foreground">
                            {{ t.statisticsWeeklyDescription }}
                        </p>
                    </div>
                    <div class="flex shrink-0 items-center gap-4 text-xs font-medium text-muted-foreground">
                        <span class="flex items-center gap-2">
                            <i class="h-0.5 w-5 rounded-full bg-success" />{{ t.statisticsCompleted }}
                        </span>
                        <span class="flex items-center gap-2">
                            <i class="h-0.5 w-5 rounded-full bg-destructive" />{{ t.statisticsFailed }}
                        </span>
                    </div>
                </header>

                <div class="overflow-x-auto border-t border-border px-2 py-4 sm:px-4">
                    <div
                        v-if="isLoading"
                        class="h-[250px] min-w-[560px] animate-pulse rounded-lg bg-muted/55 motion-reduce:animate-none"
                    />
                    <WeeklyExerciseChart
                        v-else
                        :activity="statistics.exercise_activity"
                        :locale="locale"
                        :completed-label="t.statisticsCompleted"
                        :failed-label="t.statisticsFailed"
                        :tasks-label="t.statisticsTasks"
                    />
                </div>
            </section>

            <section class="overflow-hidden rounded-xl border border-border bg-card">
                <header class="px-4 py-4 sm:px-5">
                    <h2 class="text-base font-semibold text-foreground">{{ t.statisticsVocabularyTitle }}</h2>
                    <p class="mt-1 max-w-2xl text-sm leading-5 text-muted-foreground">
                        {{ t.statisticsVocabularyDescription }}
                    </p>
                </header>

                <div class="border-t border-border px-4 py-5 sm:px-5">
                    <div
                        v-if="isLoading"
                        class="h-[160px] animate-pulse rounded-lg bg-muted/55 motion-reduce:animate-none"
                    />
                    <VocabularyActivityGrid
                        v-else
                        :activity="statistics.vocabulary_activity"
                        :locale="locale"
                        :vocabulary-label="t.statisticsVocabularyAdded"
                        :less-label="t.statisticsLess"
                        :more-label="t.statisticsMore"
                    />
                </div>
            </section>
        </div>
    </main>
</template>
