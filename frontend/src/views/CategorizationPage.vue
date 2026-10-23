<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { adminApi, type CategorizationStats } from '@/api/admin'
import type { Word, VocabularyItem } from '@/api/vocabulary'
import type { PaginationData } from '@/api/pagination'
import type { PartOfSpeech } from '@/lib/partOfSpeech'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import AdminWordCategory from '@/components/AdminWordCategory.vue'

type List = 'unknown' | 'mismatches' | 'words'
const lists: List[] = ['unknown', 'words', 'mismatches']
const { t } = useI18n()
const active = ref<List>('unknown')
const words = reactive<Record<'unknown' | 'words', Word[]>>({ unknown: [], words: [] })
const pairs = ref<VocabularyItem[]>([])
const search = ref('')
const loading = reactive({ unknown: false, mismatches: false, words: false })
const errors = reactive({ unknown: false, mismatches: false, words: false })
const initialPagination = (): PaginationData => ({ page: 1, page_size: 20, total: 0, total_pages: 0 })
const pages = reactive({ unknown: initialPagination(), mismatches: initialPagination(), words: initialPagination() })
const pagination = computed(() => pages[active.value])
const saving = ref(false)
const saveMessage = ref('')
const saveFailed = ref(false)
const stats = ref<CategorizationStats | null>(null)
const statsError = ref(false)
const restarting = ref(false)
const restartMessage = ref('')
const restartFailed = ref(false)
const sequence = { unknown: 0, mismatches: 0, words: 0, stats: 0 }
let disposed = false
let pollTimer: ReturnType<typeof setTimeout> | undefined
let searchTimer: ReturnType<typeof setTimeout> | undefined
const canRestart = computed(
    () =>
        !saving.value &&
        !restarting.value &&
        !statsError.value &&
        stats.value?.worker &&
        !stats.value.worker.active &&
        stats.value.pending + stats.value.unknown > 0
)
const listLabels = computed(() => ({
    unknown: t.value.categorizationUnknown,
    words: t.value.categorizationWords,
    mismatches: t.value.categorizationMismatches,
}))
const countLabels = computed(() => ({
    total: t.value.categorizationTotal,
    categorized: t.value.categorizationCategorized,
    unknown: t.value.categorizationUnknownCount,
    pending: t.value.categorizationPending,
}))

async function load(list: List, page = pages[list].page) {
    const request = ++sequence[list]
    loading[list] = true
    errors[list] = false
    try {
        const response =
            list === 'unknown'
                ? await adminApi.getUnknownWords(page)
                : list === 'words'
                  ? await adminApi.getCategoryWords(page, search.value)
                  : await adminApi.getMismatchedVocabulary(page)
        if (disposed || request !== sequence[list]) return
        if (page > Math.max(1, response.pagination.total_pages)) {
            await load(list, Math.max(1, response.pagination.total_pages))
            return
        }
        if (list === 'mismatches') pairs.value = response.data as VocabularyItem[]
        else words[list] = response.data as Word[]
        pages[list] = response.pagination
    } catch {
        if (!disposed && request === sequence[list]) errors[list] = true
    } finally {
        if (!disposed && request === sequence[list]) loading[list] = false
    }
}

async function loadStats() {
    clearTimeout(pollTimer)
    const request = ++sequence.stats
    try {
        const response = await adminApi.getCategorizationStats()
        if (disposed || request !== sequence.stats) return
        stats.value = response
        statsError.value = false
    } catch {
        if (!disposed && request === sequence.stats) statsError.value = true
    } finally {
        if (!disposed && request === sequence.stats) {
            const busy = stats.value?.worker?.active || (stats.value?.pending ?? 0) > 0
            pollTimer = setTimeout(() => void loadStats(), busy || statsError.value ? 2000 : 10000)
        }
    }
}

async function refresh() {
    await Promise.all([loadStats(), ...lists.map((list) => load(list))])
}

async function restart() {
    if (!canRestart.value) return
    restarting.value = true
    restartMessage.value = ''
    restartFailed.value = false
    ++sequence.stats
    clearTimeout(pollTimer)
    try {
        await adminApi.restartCategorization()
        restartMessage.value = t.value.categorizationStarted
    } catch (error) {
        restartFailed.value = true
        restartMessage.value =
            (error as { status?: number }).status === 409
                ? t.value.categorizationAlreadyActive
                : t.value.categorizationRestartError
    } finally {
        await refresh()
        restarting.value = false
    }
}

async function save(word: Word, category: PartOfSpeech) {
    saving.value = true
    saveMessage.value = ''
    saveFailed.value = false
    try {
        await adminApi.setWordPartOfSpeech(word.id, category)
        saveMessage.value = t.value.categorizationSaved
        await refresh()
    } catch {
        saveFailed.value = true
        saveMessage.value = t.value.categorizationSaveError
    } finally {
        saving.value = false
    }
}

watch(search, () => {
    ++sequence.words
    loading.words = true
    clearTimeout(searchTimer)
    searchTimer = setTimeout(() => void load('words', 1), 300)
})

onMounted(() => void refresh())
onBeforeUnmount(() => {
    disposed = true
    clearTimeout(pollTimer)
    clearTimeout(searchTimer)
})
</script>

<template>
    <main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 sm:py-8">
        <header class="mb-6 max-w-3xl">
            <h2 class="text-xl font-semibold">{{ t.navCategorization }}</h2>
            <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t.categorizationScope }}</p>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t.categorizationMismatchNote }}</p>
        </header>

        <section :aria-label="t.categorizationStats" class="mb-6 border-y border-border py-4">
            <div class="flex flex-wrap items-start justify-between gap-4">
                <dl v-if="stats" class="flex flex-wrap gap-x-8 gap-y-3">
                    <div v-for="key in ['pending', 'unknown', 'categorized', 'total'] as const" :key="key">
                        <dt class="text-xs text-muted-foreground">{{ countLabels[key] }}</dt>
                        <dd class="mt-1 text-lg font-medium tabular-nums">{{ stats[key].toLocaleString() }}</dd>
                    </div>
                </dl>
                <p v-else class="text-sm text-muted-foreground">{{ t.categorizationLoading }}</p>
                <Button :disabled="!canRestart" @click="restart">
                    {{ restarting ? t.categorizationRestarting : t.categorizationRestart }}
                </Button>
            </div>
            <p v-if="stats?.worker" class="mt-3 flex flex-wrap gap-x-3 gap-y-1 text-sm text-muted-foreground">
                <span :class="stats.worker.active ? 'text-foreground' : ''">{{
                    stats.worker.active ? t.categorizationRunning : t.categorizationIdle
                }}</span>
                <span class="tabular-nums">{{
                    t.categorizationProcessed.replace('{count}', stats.worker.processed.toLocaleString())
                }}</span>
                <span class="tabular-nums" :class="stats.worker.failed ? 'text-destructive' : ''">{{
                    t.categorizationFailed.replace('{count}', stats.worker.failed.toLocaleString())
                }}</span>
            </p>
            <p v-else-if="stats" class="mt-3 text-sm text-muted-foreground">{{ t.categorizationUnavailable }}</p>
            <p class="mt-2 max-w-3xl text-xs leading-5 text-muted-foreground">{{ t.categorizationRestartNote }}</p>
            <p v-if="stats?.worker?.scan_failed" role="alert" class="mt-2 text-sm text-destructive">
                {{ t.categorizationScanFailed }}
            </p>
            <p v-if="statsError" role="alert" class="mt-2 text-sm text-destructive">{{ t.categorizationStatsError }}</p>
            <p
                v-if="restartMessage"
                role="status"
                class="mt-2 text-sm"
                :class="restartFailed ? 'text-destructive' : 'text-foreground'"
            >
                {{ restartMessage }}
            </p>
        </section>

        <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
            <div class="flex flex-wrap gap-2" :aria-label="t.navCategorization">
                <Button
                    v-for="list in lists"
                    :key="list"
                    :variant="active === list ? 'secondary' : 'ghost'"
                    :aria-pressed="active === list"
                    @click="active = list"
                >
                    {{ listLabels[list] }}
                    <span class="ml-2 tabular-nums text-muted-foreground">{{
                        list === 'unknown' && stats
                            ? stats.pending + stats.unknown
                            : list === 'words' && stats
                              ? stats.total
                              : pages[list].total
                    }}</span>
                </Button>
            </div>
            <Button variant="outline" :disabled="loading[active] || saving || restarting" @click="refresh">{{
                t.categorizationRefresh
            }}</Button>
        </div>

        <div v-if="active === 'words'" class="mb-5 max-w-lg">
            <label for="category-word-search" class="mb-2 block text-sm font-medium">{{
                t.categorizationSearch
            }}</label>
            <input
                id="category-word-search"
                v-model="search"
                type="search"
                :placeholder="t.categorizationSearch"
                class="h-10 w-full rounded-md border border-input bg-background px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
        </div>

        <p
            v-if="saveMessage"
            role="status"
            class="mb-4 text-sm"
            :class="saveFailed ? 'text-destructive' : 'text-foreground'"
        >
            {{ saveMessage }}
        </p>
        <div v-if="errors[active]" role="alert" class="flex flex-wrap items-center gap-3 py-6 text-sm text-destructive">
            {{ t.categorizationLoadError }}
            <Button variant="outline" @click="load(active)">{{ t.commonRetry }}</Button>
        </div>
        <p v-else-if="loading[active]" role="status" class="py-8 text-sm text-muted-foreground">
            {{ t.categorizationLoading }}
        </p>
        <template v-else>
            <ul
                v-if="active !== 'mismatches' && words[active].length"
                class="divide-y divide-border border-y border-border"
            >
                <li v-for="word in words[active as 'unknown' | 'words']" :key="word.id" class="py-4">
                    <AdminWordCategory
                        :word="word"
                        :busy="saving || restarting"
                        :input-id="`${active}-${word.id}`"
                        class="max-w-lg"
                        @save="save"
                    />
                </li>
            </ul>
            <ul
                v-else-if="active === 'mismatches' && pairs.length"
                class="divide-y divide-border border-y border-border"
            >
                <li v-for="pair in pairs" :key="pair.id" class="py-4">
                    <p class="mb-3 break-all text-xs text-muted-foreground">
                        {{ t.categorizationRecord }} {{ pair.id }}
                    </p>
                    <div class="grid min-w-0 gap-5 md:grid-cols-2 md:gap-10">
                        <AdminWordCategory
                            :word="pair.translation.original"
                            :busy="saving || restarting"
                            :input-id="`original-${pair.id}`"
                            @save="save"
                        />
                        <AdminWordCategory
                            :word="pair.translation.translation"
                            :busy="saving || restarting"
                            :input-id="`translated-${pair.id}`"
                            @save="save"
                        />
                    </div>
                </li>
            </ul>
            <p v-else class="py-8 text-sm text-muted-foreground">
                {{
                    active === 'unknown'
                        ? t.categorizationNoUnknown
                        : active === 'words'
                          ? t.categorizationNoWords
                          : t.categorizationNoMismatches
                }}
            </p>

            <nav
                v-if="pagination.total_pages > 1"
                :aria-label="t.categorizationPages"
                class="mt-5 flex flex-wrap items-center justify-between gap-3"
            >
                <p class="text-sm text-muted-foreground">
                    {{
                        t.descriptionsPage
                            .replace('{page}', String(pagination.page))
                            .replace('{total}', String(pagination.total_pages))
                    }}
                </p>
                <div class="flex gap-2">
                    <Button
                        variant="outline"
                        :disabled="pagination.page <= 1 || saving"
                        @click="load(active, pagination.page - 1)"
                        >{{ t.descriptionsPrevious }}</Button
                    >
                    <Button
                        variant="outline"
                        :disabled="pagination.page >= pagination.total_pages || saving"
                        @click="load(active, pagination.page + 1)"
                        >{{ t.descriptionsNext }}</Button
                    >
                </div>
            </nav>
        </template>
    </main>
</template>
