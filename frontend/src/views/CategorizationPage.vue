<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminApi } from '@/api/admin'
import type { Word, VocabularyItem } from '@/api/vocabulary'
import type { PaginationData } from '@/api/pagination'
import type { PartOfSpeech } from '@/lib/partOfSpeech'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import AdminWordCategory from '@/components/AdminWordCategory.vue'

type List = 'unknown' | 'mismatches'
const { t } = useI18n()
const active = ref<List>('unknown')
const words = ref<Word[]>([])
const pairs = ref<VocabularyItem[]>([])
const loading = reactive({ unknown: false, mismatches: false })
const errors = reactive({ unknown: false, mismatches: false })
const initialPagination = (): PaginationData => ({ page: 1, page_size: 20, total: 0, total_pages: 0 })
const pages = reactive({ unknown: initialPagination(), mismatches: initialPagination() })
const pagination = computed(() => pages[active.value])
const saving = ref(false)
const saveMessage = ref('')
const saveFailed = ref(false)
const sequence = { unknown: 0, mismatches: 0 }

async function load(list: List, page = pages[list].page) {
    const request = ++sequence[list]
    loading[list] = true
    errors[list] = false
    try {
        const response =
            list === 'unknown' ? await adminApi.getUnknownWords(page) : await adminApi.getMismatchedVocabulary(page)
        if (request !== sequence[list]) return
        if (page > Math.max(1, response.pagination.total_pages)) {
            await load(list, Math.max(1, response.pagination.total_pages))
            return
        }
        if (list === 'unknown') words.value = response.data as Word[]
        else pairs.value = response.data as VocabularyItem[]
        pages[list] = response.pagination
    } catch {
        if (request === sequence[list]) errors[list] = true
    } finally {
        if (request === sequence[list]) loading[list] = false
    }
}

async function save(word: Word, category: PartOfSpeech) {
    saving.value = true
    saveMessage.value = ''
    saveFailed.value = false
    try {
        await adminApi.setWordPartOfSpeech(word.id, category)
        saveMessage.value = t.value.categorizationSaved
    } catch (error) {
        saveFailed.value = true
        saveMessage.value =
            (error as { status?: number }).status === 409
                ? t.value.categorizationConflict
                : t.value.categorizationSaveError
    } finally {
        await Promise.all([load('unknown'), load('mismatches')])
        saving.value = false
    }
}

onMounted(() => void Promise.all([load('unknown'), load('mismatches')]))
</script>

<template>
    <main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 sm:py-8">
        <header class="mb-6 max-w-3xl">
            <h2 class="text-xl font-semibold">{{ t.navCategorization }}</h2>
            <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ t.categorizationScope }}</p>
            <p class="mt-1 text-sm leading-6 text-muted-foreground">{{ t.categorizationMismatchNote }}</p>
        </header>

        <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
            <div class="flex flex-wrap gap-2" :aria-label="t.navCategorization">
                <Button
                    v-for="list in ['unknown', 'mismatches'] as const"
                    :key="list"
                    :variant="active === list ? 'secondary' : 'ghost'"
                    :aria-pressed="active === list"
                    @click="active = list"
                >
                    {{ list === 'unknown' ? t.categorizationUnknown : t.categorizationMismatches }}
                    <span class="ml-2 tabular-nums text-muted-foreground">{{ pages[list].total }}</span>
                </Button>
            </div>
            <Button variant="outline" :disabled="loading[active] || saving" @click="load(active)">{{
                t.categorizationRefresh
            }}</Button>
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
            <ul v-if="active === 'unknown' && words.length" class="divide-y divide-border border-y border-border">
                <li v-for="word in words" :key="word.id" class="py-4">
                    <AdminWordCategory
                        :word="word"
                        :busy="saving"
                        :input-id="`unknown-${word.id}`"
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
                            :busy="saving"
                            :input-id="`original-${pair.id}`"
                            @save="save"
                        />
                        <AdminWordCategory
                            :word="pair.translation.translation"
                            :busy="saving"
                            :input-id="`translated-${pair.id}`"
                            @save="save"
                        />
                    </div>
                </li>
            </ul>
            <p v-else class="py-8 text-sm text-muted-foreground">
                {{ active === 'unknown' ? t.categorizationNoUnknown : t.categorizationNoMismatches }}
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
