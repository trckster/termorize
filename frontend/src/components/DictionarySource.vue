<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { LoaderCircle } from 'lucide-vue-next'
import { adminApi, type Dictionary, type DictionaryImportJob } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { formatDate, formatNumber } from '@/lib/utils'

const props = defineProps<{ dictionary: Dictionary }>()
const { t } = useI18n()
const latest = ref<DictionaryImportJob | null>(null)
const jobs = ref<DictionaryImportJob[]>([])
const page = ref(1)
const totalPages = ref(0)
const loading = ref(true)
const loadError = ref(false)
const startError = ref('')
const starting = ref(false)
let disposed = false
let version = 0
let timer: ReturnType<typeof setTimeout> | undefined
const active = computed(() => latest.value && ['queued', 'downloading', 'importing'].includes(latest.value.status))
const retryable = computed(() => latest.value && ['failed', 'interrupted'].includes(latest.value.status))
const previous = computed(() => jobs.value.filter((job) => job.id !== latest.value?.id))
const statusLabel = (job: DictionaryImportJob) => {
    const labels = {
        queued: t.value.dictionaryQueued,
        downloading: t.value.dictionaryDownloading,
        importing: t.value.dictionaryImporting,
        succeeded: job.failed ? t.value.dictionaryCompletedWithErrors : t.value.dictionaryCompleted,
        failed: t.value.dictionaryFailed,
        interrupted: t.value.dictionaryInterrupted,
    }
    return labels[job.status]
}
const counts = (job: DictionaryImportJob) => [
    { label: t.value.dictionaryProcessed, value: job.processed },
    { label: t.value.dictionaryInserted, value: job.inserted },
    { label: t.value.dictionaryClassified, value: job.classified },
    { label: t.value.dictionarySkipped, value: job.skipped },
    { label: t.value.dictionaryFailedRecords, value: job.failed },
]
const load = async (nextPage = page.value, quiet = false) => {
    const currentVersion = ++version
    clearTimeout(timer)
    if (!quiet) loading.value = true
    try {
        const current = await adminApi.getDictionaryImports(props.dictionary.id, 1)
        const history = nextPage === 1 ? current : await adminApi.getDictionaryImports(props.dictionary.id, nextPage)
        if (disposed || currentVersion !== version) return
        latest.value = current.data[0] ?? null
        jobs.value = history.data
        page.value = nextPage
        totalPages.value = history.pagination.total_pages
        loadError.value = false
    } catch {
        if (!disposed && currentVersion === version) loadError.value = true
    } finally {
        if (!disposed && currentVersion === version) {
            loading.value = false
            timer = setTimeout(() => void load(page.value, true), 5000)
        }
    }
}
const start = async () => {
    if (starting.value || active.value) return
    starting.value = true
    startError.value = ''
    ++version
    clearTimeout(timer)
    try {
        const job = await adminApi.startDictionaryImport(props.dictionary.id)
        if (disposed) return
        latest.value = job
    } catch (error) {
        if (disposed) return
        startError.value =
            (error as { status?: number }).status === 409
                ? t.value.dictionaryAlreadyActive
                : t.value.dictionaryStartError
    } finally {
        if (!disposed) {
            starting.value = false
            await load(1, true)
        }
    }
}
onMounted(() => void load())
onBeforeUnmount(() => {
    disposed = true
    ++version
    clearTimeout(timer)
})
</script>

<template>
    <section class="py-6 first:pt-0" :aria-labelledby="`dictionary-${dictionary.id}`">
        <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0">
                <h3 :id="`dictionary-${dictionary.id}`" class="text-lg font-semibold">{{ dictionary.name }}</h3>
                <a
                    :href="dictionary.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="mt-1 inline-block text-sm text-muted-foreground underline underline-offset-4 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                    {{ dictionary.edition }}
                </a>
            </div>
            <Button variant="outline" :disabled="loading || loadError || starting || !!active" @click="start">
                <LoaderCircle
                    v-if="starting || active"
                    class="mr-2 size-4 motion-safe:animate-spin"
                    aria-hidden="true"
                />
                {{
                    active
                        ? t.dictionaryInProgress
                        : retryable
                          ? t.dictionaryRetry
                          : latest
                            ? t.dictionaryUpdate
                            : t.dictionaryImport
                }}
            </Button>
        </div>
        <p v-if="startError" role="alert" class="mt-3 text-sm text-foreground">{{ startError }}</p>
        <div v-if="loadError" role="alert" class="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <span class="text-foreground">{{ t.dictionaryLoadError }}</span>
            <Button variant="outline" size="sm" @click="load()">{{ t.dictionaryRefresh }}</Button>
        </div>
        <p v-if="loading && !latest" role="status" class="mt-4 text-sm text-muted-foreground">
            {{ t.dictionaryLoading }}
        </p>
        <template v-else-if="latest">
            <div class="mt-4 flex flex-wrap items-baseline gap-x-3 gap-y-1 text-sm">
                <span role="status" class="font-medium text-foreground">{{ statusLabel(latest) }}</span>
                <time :datetime="latest.created_at" class="text-muted-foreground">{{
                    formatDate(latest.created_at)
                }}</time>
            </div>
            <p v-if="latest.status === 'downloading'" class="mt-2 text-sm tabular-nums text-muted-foreground">
                {{ formatNumber(Math.floor(latest.downloaded_bytes / 1048576)) }} MB
                <template v-if="latest.total_bytes">
                    / {{ formatNumber(Math.ceil(latest.total_bytes / 1048576)) }} MB</template
                >
                · {{ t.dictionaryDownloaded }}
            </p>
            <dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-5">
                <div v-for="count in counts(latest)" :key="count.label">
                    <dt class="text-xs leading-5 text-muted-foreground">{{ count.label }}</dt>
                    <dd class="text-base font-medium tabular-nums">{{ formatNumber(count.value) }}</dd>
                </div>
            </dl>
            <p v-if="latest.error" class="mt-3 whitespace-pre-wrap break-words text-sm text-foreground">
                {{ latest.error }}
            </p>
            <details v-if="latest.record_errors.length" class="mt-3 text-sm">
                <summary
                    class="cursor-pointer rounded-sm py-2 text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                    {{ t.dictionaryRecordErrors }}
                </summary>
                <ul class="mt-1 list-inside list-disc space-y-1 break-words text-foreground">
                    <li v-for="error in latest.record_errors" :key="error">{{ error }}</li>
                </ul>
            </details>
        </template>
        <p v-else-if="!loadError" class="mt-4 text-sm text-muted-foreground">{{ t.dictionaryNoImports }}</p>
        <details class="mt-4 text-sm">
            <summary
                class="cursor-pointer rounded-sm py-2 text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
                {{ t.dictionarySourceDetails }}
            </summary>
            <p class="mt-2 break-words leading-6">{{ dictionary.license }}</p>
            <p class="mt-2 break-words leading-6 text-muted-foreground">{{ dictionary.attribution }}</p>
        </details>
        <details v-if="previous.length || totalPages > 1" class="mt-1 text-sm">
            <summary
                class="cursor-pointer rounded-sm py-2 text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
                {{ t.dictionaryHistory }}
            </summary>
            <ol class="divide-y divide-border">
                <li v-for="job in previous" :key="job.id" class="py-4">
                    <p class="flex flex-wrap gap-x-3 gap-y-1">
                        <span class="font-medium">{{ statusLabel(job) }}</span>
                        <time :datetime="job.created_at" class="text-muted-foreground">{{
                            formatDate(job.created_at)
                        }}</time>
                    </p>
                    <dl class="mt-3 grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-5">
                        <div v-for="count in counts(job)" :key="count.label">
                            <dt class="text-xs text-muted-foreground">{{ count.label }}</dt>
                            <dd class="tabular-nums">{{ formatNumber(count.value) }}</dd>
                        </div>
                    </dl>
                    <p v-if="job.error" class="mt-2 break-words text-foreground">{{ job.error }}</p>
                    <ul v-if="job.record_errors.length" class="mt-2 list-inside list-disc break-words text-foreground">
                        <li v-for="error in job.record_errors" :key="error">{{ error }}</li>
                    </ul>
                </li>
            </ol>
            <div v-if="totalPages > 1" class="mt-3 flex flex-wrap items-center gap-3">
                <Button variant="outline" size="sm" :disabled="loading || page <= 1" @click="load(page - 1)">{{
                    t.descriptionsPrevious
                }}</Button>
                <span class="text-muted-foreground">{{
                    t.descriptionsPage.replace('{page}', String(page)).replace('{total}', String(totalPages))
                }}</span>
                <Button variant="outline" size="sm" :disabled="loading || page >= totalPages" @click="load(page + 1)">{{
                    t.descriptionsNext
                }}</Button>
            </div>
        </details>
    </section>
</template>
