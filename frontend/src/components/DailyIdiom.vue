<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Loader2 } from 'lucide-vue-next'
import { dailyIdiomApi, type DailyIdiom } from '@/api/dailyIdiom'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { localDateInTimezone } from '@/lib/localDate'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const { t } = useI18n()
const language = ref(auth.user?.settings.main_learning_language || 'en')
const timezone = computed(() => auth.user?.settings.time_zone || 'UTC')
const requested = ref(false)
const loading = ref(false)
const failed = ref(false)
const daily = ref<DailyIdiom | null>(null)
const description = ref('')
let requestVersion = 0
let requestedDate = ''
let refreshTimer: ReturnType<typeof setInterval> | undefined

const currentDate = () => localDateInTimezone(new Date(), timezone.value)

async function load() {
    requested.value = true
    loading.value = true
    failed.value = false
    daily.value = null
    description.value = ''
    requestedDate = currentDate()
    const version = ++requestVersion
    try {
        const result = await dailyIdiomApi.get(language.value)
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
    if (document.visibilityState === 'hidden' || !requested.value || currentDate() === requestedDate) return
    void load()
}

watch([language, timezone], () => {
    if (requested.value) void load()
})
watch(
    () => auth.user?.settings.main_learning_language,
    (next, previous) => {
        if (next && language.value === (previous || 'en')) language.value = next
    }
)

onMounted(() => {
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
    <section aria-labelledby="daily-idiom-title" class="mt-8 border-t border-border pt-6 sm:mt-10">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h2 id="daily-idiom-title" class="text-lg font-semibold">{{ t.dailyIdiomTitle }}</h2>
            <div class="w-full sm:w-52">
                <LanguageSelector v-model="language" :aria-label="t.dailyIdiomLanguage" />
            </div>
        </div>
        <div class="mt-4" aria-live="polite" :aria-busy="loading">
            <Button v-if="!requested" variant="outline" class="min-h-11" @click="load">
                {{ t.dailyIdiomShow }}
            </Button>
            <div v-else class="space-y-3">
                <p v-if="daily?.idiom" :lang="daily.language" class="break-words text-xl font-medium leading-7">
                    {{ daily.idiom.word }}
                </p>
                <p v-if="description" :lang="daily?.language" class="max-w-prose break-words leading-7">
                    {{ description }}
                </p>
                <p v-if="loading" role="status" class="flex items-center gap-2 text-sm text-muted-foreground">
                    <Loader2 class="size-4 motion-safe:animate-spin" aria-hidden="true" />
                    {{ t.dailyIdiomLoading }}
                </p>
                <div v-else-if="failed" class="flex flex-wrap items-center gap-3">
                    <p role="alert" class="text-sm text-destructive">{{ t.dailyIdiomError }}</p>
                    <Button variant="outline" class="min-h-11" @click="load">{{ t.commonRetry }}</Button>
                </div>
                <p v-else-if="daily && !daily.idiom" class="text-sm text-muted-foreground">{{ t.dailyIdiomEmpty }}</p>
            </div>
        </div>
    </section>
</template>
