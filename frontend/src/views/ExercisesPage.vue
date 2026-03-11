<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { exercisesApi, type ExerciseStatistics } from '@/api/exercises.ts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Activity, AlertCircle, Ban, CheckCircle2, CircleDashed } from 'lucide-vue-next'

const statistics = ref<ExerciseStatistics>({
    in_progress: 0,
    done: 0,
    failed: 0,
    ignored: 0,
})
const isLoading = ref(true)
const errorMessage = ref('')

const totalExercises = computed(() => {
    return statistics.value.in_progress + statistics.value.done + statistics.value.failed + statistics.value.ignored
})

const statisticCards = computed(() => [
    {
        key: 'in_progress',
        label: 'In Progress',
        description: 'Active exercises waiting for completion.',
        value: statistics.value.in_progress,
        icon: Activity,
        accentClass: 'text-sky-600 bg-sky-500/10 border-sky-500/20',
    },
    {
        key: 'done',
        label: 'Done',
        description: 'Exercises answered successfully.',
        value: statistics.value.done,
        icon: CheckCircle2,
        accentClass: 'text-emerald-600 bg-emerald-500/10 border-emerald-500/20',
    },
    {
        key: 'failed',
        label: 'Failed',
        description: 'Exercises answered incorrectly.',
        value: statistics.value.failed,
        icon: AlertCircle,
        accentClass: 'text-rose-600 bg-rose-500/10 border-rose-500/20',
    },
    {
        key: 'ignored',
        label: 'Ignored',
        description: 'Exercises that expired without an answer.',
        value: statistics.value.ignored,
        icon: Ban,
        accentClass: 'text-amber-600 bg-amber-500/10 border-amber-500/20',
    },
])

const fetchStatistics = async () => {
    isLoading.value = true
    errorMessage.value = ''

    try {
        statistics.value = await exercisesApi.getStatistics()
    } catch {
        errorMessage.value = 'Failed to load exercise statistics. Please try again.'
    } finally {
        isLoading.value = false
    }
}

onMounted(() => {
    void fetchStatistics()
})
</script>

<template>
    <main class="px-6 py-8">
        <div class="mx-auto max-w-6xl space-y-6">
            <section class="rounded-3xl border border-border bg-gradient-to-br from-card via-card to-muted/40 p-6 shadow-sm">
                <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                    <div class="space-y-2">
                        <p class="text-sm font-medium uppercase tracking-[0.2em] text-muted-foreground">Exercises</p>
                        <h1 class="text-3xl font-bold text-foreground">Your exercise overview</h1>
                        <p class="max-w-2xl text-sm text-muted-foreground">
                            Track how your practice is going across all generated exercises.
                        </p>
                    </div>

                    <div class="flex items-center gap-4 rounded-2xl border border-border bg-background/80 px-5 py-4 backdrop-blur">
                        <div class="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10 text-primary">
                            <CircleDashed class="h-6 w-6" />
                        </div>
                        <div>
                            <p class="text-sm text-muted-foreground">Tracked exercises</p>
                            <p class="text-3xl font-semibold text-foreground">{{ totalExercises }}</p>
                        </div>
                    </div>
                </div>
            </section>

            <div v-if="errorMessage" class="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                {{ errorMessage }}
            </div>

            <section class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                <Card v-for="item in statisticCards" :key="item.key" class="overflow-hidden border-border/70">
                    <CardHeader class="space-y-4 pb-4">
                        <div class="flex items-start justify-between gap-3">
                            <div>
                                <CardDescription>{{ item.label }}</CardDescription>
                                <CardTitle class="mt-2 text-4xl">
                                    {{ isLoading ? '-' : item.value }}
                                </CardTitle>
                            </div>
                            <div
                                class="flex h-11 w-11 items-center justify-center rounded-2xl border"
                                :class="item.accentClass"
                            >
                                <component :is="item.icon" class="h-5 w-5" />
                            </div>
                        </div>
                    </CardHeader>
                    <CardContent>
                        <p class="text-sm text-muted-foreground">{{ item.description }}</p>
                    </CardContent>
                </Card>
            </section>
        </div>
    </main>
</template>
