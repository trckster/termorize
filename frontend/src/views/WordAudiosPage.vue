<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Database, LoaderCircle, RotateCcw, Search } from 'lucide-vue-next'
import { adminApi, type AdminWordPronunciation } from '@/api/admin.ts'
import type { PaginationData } from '@/api/pagination.ts'
import AdminAudioButton from '@/components/AdminAudioButton.vue'
import { Button } from '@/components/ui/button'
import { Pagination, PaginationContent, PaginationEllipsis, PaginationItem } from '@/components/ui/pagination'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { formatDate, formatNumber, formatRelativeTime } from '@/lib/utils.ts'

const { t } = useI18n()
const { addToast } = useToast()
const settingsStore = useSettingsStore()

const audios = ref<AdminWordPronunciation[]>([])
const currentPage = ref(1)
const pagination = ref<PaginationData>({ page: 1, page_size: 20, total: 0, total_pages: 0 })
const searchInput = ref('')
const search = ref('')
const isLoading = ref(true)
const hasError = ref(false)
const regeneratingId = ref<string | null>(null)
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

const hasSearch = computed(() => search.value.length > 0)

const getLanguageLabel = (code: string) => {
    const language = settingsStore.languageOptions.find((option) => option.code === code)
    return language ? `${language.emoji} ${language.name}` : `${settingsStore.getFlag(code)} ${code.toUpperCase()}`
}

const formatBytes = (bytes: number) => {
    if (bytes < 1024) return `${formatNumber(bytes)} B`
    if (bytes < 1024 * 1024) return `${formatNumber(Math.round(bytes / 1024))} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const fetchAudios = async (page: number) => {
    isLoading.value = true
    hasError.value = false
    currentPage.value = page

    try {
        const response = await adminApi.getWordPronunciations(
            page,
            pagination.value.page_size,
            search.value || undefined
        )
        audios.value = response.data
        pagination.value = response.pagination
    } catch {
        audios.value = []
        hasError.value = true
    } finally {
        isLoading.value = false
    }
}

const regenerateAudio = async (audio: AdminWordPronunciation) => {
    regeneratingId.value = audio.id
    try {
        await adminApi.regenerateWordPronunciation(audio.id)
        await fetchAudios(currentPage.value)
        addToast({
            title: t.value.wordAudiosRegeneratedTitle,
            description: t.value.wordAudiosRegeneratedDescription.replace('{word}', audio.word),
            variant: 'success',
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.wordAudiosRegenerateError.replace('{word}', audio.word),
            variant: 'destructive',
        })
    } finally {
        regeneratingId.value = null
    }
}

watch(searchInput, (value) => {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
    searchDebounceTimer = setTimeout(() => {
        const nextSearch = value.trim()
        if (nextSearch !== search.value) search.value = nextSearch
    }, 350)
})

watch(search, () => void fetchAudios(1))

onMounted(() => void fetchAudios(1))
onBeforeUnmount(() => {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
})
</script>

<template>
    <main class="px-4 py-6 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-7xl space-y-6">
            <header class="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <p class="text-xs font-semibold uppercase tracking-[0.14em] text-primary">
                        {{ t.wordAudiosLabel }}
                    </p>
                    <h1 class="mt-2 text-2xl font-semibold tracking-tight text-foreground">
                        {{ t.wordAudiosHeading }}
                    </h1>
                    <p class="mt-1.5 max-w-2xl text-sm leading-6 text-muted-foreground">
                        {{ t.wordAudiosDescription }}
                    </p>
                </div>

                <dl class="flex items-center gap-3 sm:block sm:text-right">
                    <dt class="text-xs font-medium text-muted-foreground">{{ t.wordAudiosTotal }}</dt>
                    <dd class="text-2xl font-semibold tabular-nums text-foreground sm:mt-1">
                        <span
                            v-if="isLoading && pagination.total === 0"
                            class="inline-block h-7 w-12 animate-pulse rounded bg-muted motion-reduce:animate-none"
                        />
                        <template v-else>{{ formatNumber(pagination.total) }}</template>
                    </dd>
                </dl>
            </header>

            <div class="relative max-w-xl">
                <Search
                    class="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                    aria-hidden="true"
                />
                <input
                    v-model="searchInput"
                    type="search"
                    :placeholder="t.wordAudiosSearchPlaceholder"
                    :aria-label="t.wordAudiosSearchLabel"
                    class="h-11 w-full rounded-md border border-input bg-background pl-10 pr-4 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                />
            </div>

            <div
                v-if="hasError"
                role="alert"
                class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-destructive/25 bg-destructive/5 px-4 py-3 text-sm text-destructive"
            >
                <span>{{ t.wordAudiosLoadError }}</span>
                <button
                    type="button"
                    class="font-medium underline underline-offset-4 hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    @click="fetchAudios(currentPage)"
                >
                    {{ t.commonRetry }}
                </button>
            </div>

            <div
                v-else-if="isLoading"
                class="overflow-hidden rounded-xl border border-border bg-card"
                aria-label="Loading audios"
            >
                <div
                    v-for="index in 8"
                    :key="index"
                    class="flex items-center gap-4 border-b border-border px-4 py-4 last:border-0 sm:px-5"
                >
                    <span class="size-10 shrink-0 animate-pulse rounded-full bg-muted motion-reduce:animate-none" />
                    <div class="min-w-0 flex-1 space-y-2">
                        <span class="block h-4 w-32 animate-pulse rounded bg-muted motion-reduce:animate-none" />
                        <span
                            class="block h-3 w-48 max-w-full animate-pulse rounded bg-muted motion-reduce:animate-none"
                        />
                    </div>
                    <span class="hidden h-9 w-28 animate-pulse rounded bg-muted motion-reduce:animate-none sm:block" />
                </div>
            </div>

            <div
                v-else-if="audios.length === 0"
                class="flex min-h-72 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card/50 px-6 text-center"
            >
                <Database class="size-8 text-muted-foreground" aria-hidden="true" />
                <h2 class="mt-4 text-base font-semibold text-foreground">
                    {{ hasSearch ? t.wordAudiosNoResults : t.wordAudiosEmpty }}
                </h2>
                <p class="mt-1 max-w-md text-sm leading-6 text-muted-foreground">
                    {{ hasSearch ? t.wordAudiosNoResultsDescription : t.wordAudiosEmptyDescription }}
                </p>
            </div>

            <template v-else>
                <div class="space-y-3 md:hidden">
                    <article
                        v-for="audio in audios"
                        :key="audio.id"
                        class="rounded-xl border border-border bg-card p-4"
                    >
                        <div class="flex items-start gap-3">
                            <AdminAudioButton
                                :pronunciation-id="audio.id"
                                :word="audio.word"
                                :listen-label="t.wordAudiosListen"
                                :pause-label="t.wordAudiosPause"
                                :loading-label="t.wordAudiosAudioLoading"
                                :error-label="t.wordAudiosAudioError"
                            />
                            <div class="min-w-0 flex-1">
                                <h2 class="break-words font-semibold text-foreground">{{ audio.word }}</h2>
                                <p class="mt-0.5 text-xs text-muted-foreground">
                                    {{ getLanguageLabel(audio.language) }}
                                </p>
                            </div>
                            <Button
                                variant="outline"
                                size="icon-sm"
                                :disabled="regeneratingId !== null"
                                :aria-label="t.wordAudiosRegenerateLabel.replace('{word}', audio.word)"
                                :title="t.wordAudiosRegenerateLabel.replace('{word}', audio.word)"
                                @click="regenerateAudio(audio)"
                            >
                                <LoaderCircle v-if="regeneratingId === audio.id" class="motion-safe:animate-spin" />
                                <RotateCcw v-else />
                            </Button>
                        </div>

                        <dl class="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 border-t border-border pt-4 text-xs">
                            <div class="col-span-2">
                                <dt class="text-muted-foreground">{{ t.wordAudiosModel }}</dt>
                                <dd class="mt-1 break-all font-mono text-foreground">{{ audio.model }}</dd>
                            </div>
                            <div>
                                <dt class="text-muted-foreground">{{ t.wordAudiosVoice }}</dt>
                                <dd class="mt-1 text-foreground">{{ audio.voice }}</dd>
                            </div>
                            <div>
                                <dt class="text-muted-foreground">{{ t.wordAudiosSize }}</dt>
                                <dd class="mt-1 tabular-nums text-foreground">{{ formatBytes(audio.size_bytes) }}</dd>
                            </div>
                            <div class="col-span-2">
                                <dt class="text-muted-foreground">{{ t.wordAudiosGenerated }}</dt>
                                <dd class="mt-1 text-foreground">
                                    <time :datetime="audio.created_at" :title="formatDate(audio.created_at)">
                                        {{ formatRelativeTime(audio.created_at) }}
                                    </time>
                                </dd>
                            </div>
                        </dl>
                    </article>
                </div>

                <div class="hidden overflow-hidden rounded-xl border border-border bg-card md:block">
                    <div class="overflow-x-auto">
                        <table class="w-full min-w-[72rem] text-left text-sm">
                            <thead class="border-b border-border bg-muted/35 text-xs font-medium text-muted-foreground">
                                <tr>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosLanguage }}</th>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosWord }}</th>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosAudio }}</th>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosGenerator }}</th>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosGenerated }}</th>
                                    <th scope="col" class="px-5 py-3">{{ t.wordAudiosDetails }}</th>
                                    <th scope="col" class="px-5 py-3 text-right">{{ t.wordAudiosActions }}</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-border">
                                <tr v-for="audio in audios" :key="audio.id" class="align-middle hover:bg-muted/25">
                                    <td class="whitespace-nowrap px-5 py-3.5 text-muted-foreground">
                                        {{ getLanguageLabel(audio.language) }}
                                    </td>
                                    <td class="max-w-56 px-5 py-3.5 font-medium text-foreground">
                                        <span class="break-words">{{ audio.word }}</span>
                                    </td>
                                    <td class="px-5 py-3.5">
                                        <AdminAudioButton
                                            :pronunciation-id="audio.id"
                                            :word="audio.word"
                                            :listen-label="t.wordAudiosListen"
                                            :pause-label="t.wordAudiosPause"
                                            :loading-label="t.wordAudiosAudioLoading"
                                            :error-label="t.wordAudiosAudioError"
                                        />
                                    </td>
                                    <td class="max-w-72 px-5 py-3.5">
                                        <p class="break-all font-mono text-xs text-foreground">{{ audio.model }}</p>
                                        <p class="mt-1 text-xs text-muted-foreground">
                                            {{ t.wordAudiosVoice }}: {{ audio.voice }}
                                        </p>
                                    </td>
                                    <td class="whitespace-nowrap px-5 py-3.5 text-foreground">
                                        <time :datetime="audio.created_at" :title="formatDate(audio.created_at)">
                                            {{ formatRelativeTime(audio.created_at) }}
                                        </time>
                                    </td>
                                    <td class="whitespace-nowrap px-5 py-3.5 text-xs text-muted-foreground">
                                        <p>{{ formatBytes(audio.size_bytes) }} · {{ audio.mime_type }}</p>
                                        <p class="mt-1">
                                            {{
                                                audio.has_telegram_file
                                                    ? t.wordAudiosTelegramCached
                                                    : t.wordAudiosTelegramNotCached
                                            }}
                                        </p>
                                    </td>
                                    <td class="px-5 py-3.5 text-right">
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            :disabled="regeneratingId !== null"
                                            @click="regenerateAudio(audio)"
                                        >
                                            <LoaderCircle
                                                v-if="regeneratingId === audio.id"
                                                class="motion-safe:animate-spin"
                                            />
                                            <RotateCcw v-else />
                                            {{
                                                regeneratingId === audio.id
                                                    ? t.wordAudiosRegenerating
                                                    : t.wordAudiosRegenerate
                                            }}
                                        </Button>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <div v-if="pagination.total_pages > 1" class="pt-1">
                    <Pagination
                        v-slot="{ page }"
                        :total="pagination.total"
                        :items-per-page="pagination.page_size"
                        :sibling-count="1"
                        show-edges
                        :page="currentPage"
                        @update:page="fetchAudios"
                    >
                        <PaginationContent v-slot="{ items }" class="flex justify-center gap-1">
                            <template v-for="(item, index) in items">
                                <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                                    <Button class="size-11 p-0" :variant="item.value === page ? 'default' : 'outline'">
                                        {{ item.value }}
                                    </Button>
                                </PaginationItem>
                                <PaginationEllipsis v-else :key="item.type + index" :index="index" />
                            </template>
                        </PaginationContent>
                    </Pagination>
                </div>
            </template>
        </div>
    </main>
</template>
