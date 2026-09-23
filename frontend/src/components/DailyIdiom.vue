<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import { dailyIdiomApi, type DailyIdiom } from '@/api/dailyIdiom'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { localDateInTimezone } from '@/lib/localDate'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const { t } = useI18n()
const mainLearningLanguage = computed(() => auth.user?.settings.main_learning_language)
const timezone = computed(() => auth.user?.settings.time_zone || 'UTC')
const loading = ref(true)
const failed = ref(false)
const daily = ref<DailyIdiom | null>(null)
const description = ref('')
let requestVersion = 0
let requestedDate = ''
let refreshTimer: ReturnType<typeof setInterval> | undefined

const currentDate = () => localDateInTimezone(new Date(), timezone.value)

async function load() {
    loading.value = true
    failed.value = false
    daily.value = null
    description.value = ''
    requestedDate = currentDate()
    const version = ++requestVersion
    try {
        const result = await dailyIdiomApi.get()
        if (version !== requestVersion) return
        daily.value = result
        if (result.idiom) {
            const content = await dailyIdiomApi.describe(result.idiom.id)
            if (version !== requestVersion) return
            description.value = content.description
        }
    } catch {
        if (version === requestVersion) failed.value = true
    } finally {
        if (version === requestVersion) {
            loading.value = false
            refreshIfDateChanged()
        }
    }
}

function refreshIfDateChanged() {
    if (document.visibilityState === 'hidden' || currentDate() === requestedDate) return
    void load()
}

watch([mainLearningLanguage, timezone], () => {
    void load()
})

onMounted(() => {
    void load()
    refreshTimer = setInterval(refreshIfDateChanged, 15_000)
    document.addEventListener('visibilitychange', refreshIfDateChanged)
    window.addEventListener('focus', refreshIfDateChanged)
})
onBeforeUnmount(() => {
    requestVersion++
    clearInterval(refreshTimer)
    document.removeEventListener('visibilitychange', refreshIfDateChanged)
    window.removeEventListener('focus', refreshIfDateChanged)
})
</script>

<template>
    <section
        v-if="daily?.idiom || failed"
        aria-labelledby="daily-idiom-title"
        class="mx-auto mt-8 max-w-lg rounded-xl border border-border bg-card px-6 py-6 text-center text-card-foreground sm:mt-10 sm:px-8 sm:py-7"
    >
        <h2 id="daily-idiom-title" class="text-base font-medium text-muted-foreground">{{ t.dailyIdiomTitle }}</h2>
        <div class="mt-4 space-y-3" aria-live="polite" :aria-busy="loading">
            <p
                v-if="daily?.idiom"
                :lang="daily.language"
                class="break-words text-balance text-2xl font-semibold leading-tight sm:text-3xl"
            >
                {{ daily.idiom.word }}
            </p>
            <p
                v-if="description"
                :lang="daily?.language"
                class="mx-auto max-w-prose break-words text-pretty text-base leading-7"
            >
                {{ description }}
            </p>
            <p
                v-if="loading"
                role="status"
                class="flex items-center justify-center gap-2 text-sm text-muted-foreground"
            >
                <Loader2 class="size-4 motion-safe:animate-spin" aria-hidden="true" />
                {{ t.dailyIdiomLoading }}
            </p>
            <div v-else-if="failed" class="flex flex-col items-center gap-3">
                <p role="alert" class="text-sm text-foreground">{{ t.dailyIdiomError }}</p>
                <Button variant="outline" class="min-h-11" @click="load">{{ t.commonRetry }}</Button>
            </div>
        </div>
    </section>
</template>
