<template>
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="max-w-6xl mx-auto">
            <div class="mb-6 flex flex-col gap-4 sm:mb-8 sm:flex-row sm:items-center sm:justify-between">
                <h1 class="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl sm:font-bold">
                    {{ t.collectionsTitle }}
                </h1>
                <div
                    class="relative grid grid-cols-2 overflow-hidden rounded-md border border-input bg-background sm:flex sm:items-center sm:overflow-visible sm:border-0 sm:bg-transparent"
                >
                    <Dialog v-model:open="isGenerateDialogOpen">
                        <DialogTrigger as-child>
                            <Button
                                variant="outline"
                                class="min-h-11 w-full rounded-none border-0 px-3 sm:w-auto sm:rounded-md sm:border"
                            >
                                <Sparkles class="h-4 w-4 mr-2" />
                                {{ t.collectionsGenerateButton }}
                            </Button>
                        </DialogTrigger>
                        <DialogContent class="sm:max-w-md">
                            <DialogHeader>
                                <DialogTitle>{{ t.collectionsGenerateDialogTitle }}</DialogTitle>
                                <DialogDescription>{{
                                    isAdmin
                                        ? t.collectionsGenerateDialogDescriptionAdmin
                                        : t.collectionsGenerateDialogDescription
                                }}</DialogDescription>
                            </DialogHeader>
                            <form @submit.prevent="handleGenerate" class="space-y-4 py-4">
                                <div class="space-y-2">
                                    <label for="collection-prompt" class="text-sm font-medium">{{
                                        t.collectionsGeneratePromptLabel
                                    }}</label>
                                    <textarea
                                        id="collection-prompt"
                                        v-model="generatePrompt"
                                        rows="3"
                                        :placeholder="t.collectionsGeneratePromptPlaceholder"
                                        maxlength="500"
                                        class="w-full resize-y rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                        @keydown.enter.shift.prevent="handleGenerate"
                                    ></textarea>
                                </div>
                                <p class="text-xs text-muted-foreground">
                                    {{ t.collectionsGenerateDisclaimer }}
                                </p>
                                <DialogFooter
                                    class="flex-col items-center justify-center gap-2 pt-2 sm:flex-col sm:justify-center"
                                >
                                    <Button type="submit" :disabled="isGenerating || !isGenerateValid">
                                        <Loader2 v-if="isGenerating" class="mr-2 h-4 w-4 animate-spin" />
                                        {{ isGenerating ? t.collectionsGenerating : t.collectionsGenerateSubmit }}
                                    </Button>
                                    <span class="text-xs text-muted-foreground/70">Shift + Enter</span>
                                </DialogFooter>
                            </form>
                        </DialogContent>
                    </Dialog>

                    <span
                        aria-hidden="true"
                        class="pointer-events-none absolute left-1/2 top-1/2 z-10 h-14 w-px -translate-x-1/2 -translate-y-1/2 rotate-[18deg] bg-border sm:hidden"
                    />

                    <Button
                        class="min-h-11 w-full rounded-none px-3 sm:ml-2 sm:w-auto sm:rounded-md"
                        :disabled="isCreating"
                        @click="handleCreate"
                    >
                        <Loader2 v-if="isCreating" class="mr-2 h-4 w-4 animate-spin" />
                        <Plus v-else class="h-4 w-4 mr-2" />
                        {{ t.collectionsCreateButton }}
                    </Button>
                </div>
            </div>

            <div class="mb-6 flex flex-col gap-3">
                <div class="relative max-w-md">
                    <input
                        id="collection-search"
                        v-model="searchInput"
                        name="collection-search"
                        type="text"
                        :placeholder="t.collectionsSearchPlaceholder"
                        :aria-label="t.collectionsSearchPlaceholder"
                        class="w-full rounded-md border border-border bg-background px-3 py-2 pr-9 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary sm:text-sm"
                    />
                    <span
                        v-if="isLoading"
                        class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                    >
                        <Loader2 class="h-4 w-4 animate-spin" />
                    </span>
                </div>

                <Select v-if="settingsStore.languageOptions.length > 0" v-model="selectedLanguage">
                    <SelectTrigger class="w-full max-w-xs" :aria-label="t.collectionsFilterByLanguage">
                        <SelectValue :placeholder="t.collectionsFilterByLanguage" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem :value="ALL_LANGUAGES">{{ t.collectionsAllLanguages }}</SelectItem>
                        <SelectItem v-for="lang in settingsStore.languageOptions" :key="lang.code" :value="lang.code">
                            {{ lang.emoji }} {{ lang.name }}
                        </SelectItem>
                    </SelectContent>
                </Select>
            </div>

            <div
                v-if="errorMessage"
                class="mb-6 rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive"
            >
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                    <span>{{ errorMessage }}</span>
                    <Button variant="outline" size="sm" @click="fetchCollections(currentPage)">{{
                        t.commonRetry
                    }}</Button>
                </div>
            </div>

            <div v-if="collections.length > 0" class="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div v-for="collection in collections" :key="collection.id" class="group relative">
                    <Card class="h-full transition-colors group-hover:bg-accent/50">
                        <router-link
                            :to="`/collections/${collection.id}`"
                            class="block h-full rounded-xl focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                        >
                            <CardHeader>
                                <div class="flex items-start justify-between gap-2">
                                    <CardTitle class="min-w-0 break-words text-lg">{{ collection.title }}</CardTitle>
                                    <span v-if="collection.is_admin">
                                        <span
                                            v-if="!collection.is_published && isAdmin"
                                            class="shrink-0 rounded bg-warning/15 px-2 py-0.5 text-xs font-medium text-warning"
                                        >
                                            {{ t.collectionsDraftBadge }}
                                        </span>
                                        <span
                                            v-else-if="isAdmin"
                                            class="shrink-0 rounded border border-primary/30 bg-primary/30 px-2 py-0.5 text-xs font-medium text-primary"
                                        >
                                            {{ t.collectionsGlobalBadge }}
                                        </span>
                                    </span>
                                    <span
                                        v-else-if="collection.is_owner"
                                        class="shrink-0 rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground"
                                    >
                                        {{ t.collectionsPrivateBadge }}
                                    </span>
                                    <span
                                        v-else-if="collection.owner_username"
                                        class="shrink-0 rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground"
                                    >
                                        @{{ collection.owner_username }}
                                    </span>
                                </div>
                            </CardHeader>
                            <CardContent class="space-y-3 pb-14">
                                <div v-if="collection.languages.length > 0" class="flex flex-wrap gap-1 text-xl">
                                    <span
                                        v-for="lang in collection.languages"
                                        :key="lang"
                                        role="img"
                                        :aria-label="getLanguageName(lang)"
                                        >{{ settingsStore.getFlag(lang) }}</span
                                    >
                                </div>
                                <p class="text-sm text-muted-foreground">
                                    {{ formatNumber(collection.translation_count) }}
                                    {{ t.collectionsTranslationsLabel }}
                                    <template v-if="collection.user_add_count > 0">
                                        · {{ saves(collection.user_add_count) }}
                                    </template>
                                </p>
                            </CardContent>
                        </router-link>
                        <Tooltip>
                            <TooltipTrigger as-child>
                                <span class="absolute bottom-2 right-2 inline-flex">
                                    <Button
                                        type="button"
                                        size="icon"
                                        variant="ghost"
                                        class="rounded-full border"
                                        :class="
                                            collection.vocabulary_count === 0
                                                ? 'border-border/60 bg-muted/20 text-muted-foreground'
                                                : 'border-primary/30 bg-primary/10 text-primary hover:bg-primary/15'
                                        "
                                        :disabled="collection.vocabulary_count === 0"
                                        :aria-label="
                                            collection.vocabulary_count === 0
                                                ? t.collectionPracticeUnavailable
                                                : `${t.collectionPracticeStart}: ${collection.title}`
                                        "
                                        @click="startPractice(collection.id)"
                                    >
                                        <Play class="h-4 w-4 translate-x-px" />
                                    </Button>
                                </span>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                                {{
                                    collection.vocabulary_count === 0
                                        ? t.collectionPracticeUnavailable
                                        : t.collectionPracticeStart
                                }}
                            </TooltipContent>
                        </Tooltip>
                    </Card>
                </div>
            </div>

            <div
                v-else-if="!isLoading && !errorMessage"
                class="mb-8 flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 px-6 text-center"
            >
                <h2 class="text-xl font-semibold text-foreground">
                    {{ hasActiveFilters ? t.collectionsNoResultsTitle : t.collectionsEmptyTitle }}
                </h2>
                <p class="mt-2 max-w-md text-sm text-muted-foreground">
                    {{ hasActiveFilters ? t.collectionsNoResultsDescription : t.collectionsEmptyDescription }}
                </p>
                <Button v-if="!hasActiveFilters" class="mt-5" :disabled="isCreating" @click="handleCreate">
                    <Loader2 v-if="isCreating" class="mr-2 h-4 w-4 animate-spin" />
                    <Plus v-else class="mr-2 h-4 w-4" />
                    {{ t.collectionsCreateButton }}
                </Button>
            </div>

            <div v-if="paginationData.total > 0" class="space-y-3">
                <p class="text-center text-sm text-muted-foreground">
                    {{ t.collectionsTotalCount }}: {{ formatNumber(paginationData.total) }}
                </p>

                <Pagination
                    v-slot="{ page }"
                    :total="paginationData.total"
                    :items-per-page="paginationData.page_size"
                    :sibling-count="1"
                    show-edges
                    :default-page="1"
                    :page="currentPage"
                    @update:page="handlePageChange"
                >
                    <PaginationContent v-slot="{ items }" class="flex justify-center gap-1">
                        <template v-for="(item, index) in items">
                            <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                                <Button class="h-11 w-11 p-0" :variant="item.value === page ? 'default' : 'outline'">
                                    {{ item.value }}
                                </Button>
                            </PaginationItem>
                            <PaginationEllipsis v-else :key="item.type + index" :index="index" />
                        </template>
                    </PaginationContent>
                </Pagination>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import { collectionsApi, type CollectionSummary } from '@/api/collections.ts'
import type { PaginationData } from '@/api/pagination.ts'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast.ts'
import { formatNumber } from '@/lib/utils.ts'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Pagination, PaginationContent, PaginationItem, PaginationEllipsis } from '@/components/ui/pagination'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import { Loader2, Play, Plus, Sparkles } from 'lucide-vue-next'

const { t, saves } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { addToast } = useToast()

const startPractice = (collectionId: string) => {
    void router.push({ name: 'collection-practice', params: { collectionId } })
}

const collections = ref<CollectionSummary[]>([])
const currentPage = ref(1)
const paginationData = ref<PaginationData>({ page: 1, page_size: 24, total: 0, total_pages: 0 })
const isLoading = ref(false)
const errorMessage = ref('')
const ALL_LANGUAGES = 'all'

const searchInput = ref('')
const search = ref('')
const selectedLanguage = ref<string>(ALL_LANGUAGES)
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

const isCreating = ref(false)

const isGenerateDialogOpen = ref(false)
const isGenerating = ref(false)
const generatePrompt = ref('')

const isAdmin = computed(() => !!authStore.user?.is_admin)
const isGenerateValid = computed(() => generatePrompt.value.trim().length > 0)
const hasActiveFilters = computed(() => !!search.value || selectedLanguage.value !== ALL_LANGUAGES)

const getLanguageName = (code: string) =>
    settingsStore.languageOptions.find((l) => l.code === code)?.name || code.toUpperCase()

watch(isGenerateDialogOpen, (isOpen) => {
    if (isOpen) {
        generatePrompt.value = ''
    }
})

watch(searchInput, (value) => {
    if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer)
    }

    searchDebounceTimer = setTimeout(() => {
        const nextSearch = value.trim()
        if (nextSearch === search.value) {
            return
        }
        search.value = nextSearch
    }, 350)
})

watch(search, async () => {
    await fetchCollections(1)
})

watch(selectedLanguage, async () => {
    await fetchCollections(1)
})

const fetchCollections = async (page: number) => {
    isLoading.value = true
    currentPage.value = page
    errorMessage.value = ''

    try {
        const response = await collectionsApi.getCollections(
            page,
            paginationData.value.page_size,
            search.value || undefined,
            selectedLanguage.value === ALL_LANGUAGES ? undefined : [selectedLanguage.value]
        )
        collections.value = response.data
        paginationData.value = response.pagination
    } catch {
        collections.value = []
        errorMessage.value = t.value.collectionsLoadErrorDescription
    } finally {
        isLoading.value = false
    }
}

const handlePageChange = async (page: number) => {
    await fetchCollections(page)
}

const handleCreate = async () => {
    if (isCreating.value) return

    isCreating.value = true
    try {
        const title = `Collection #${Math.floor(Math.random() * 999) + 1}`
        const collection = await collectionsApi.createCollection(title)
        router.push(`/collections/${collection.id}`)
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionsCreateErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isCreating.value = false
    }
}

const handleGenerate = async () => {
    if (isGenerating.value || !isGenerateValid.value) return

    isGenerating.value = true
    try {
        const collection = await collectionsApi.generate(generatePrompt.value.trim())
        isGenerateDialogOpen.value = false
        generatePrompt.value = ''

        addToast({
            title: isAdmin.value
                ? t.value.collectionsGenerateSuccessTitleAdmin
                : t.value.collectionsGenerateSuccessTitle,
            description: isAdmin.value
                ? t.value.collectionsGenerateSuccessDescriptionAdmin
                : t.value.collectionsGenerateSuccessDescription,
            variant: 'success',
            duration: 3000,
        })

        router.push(`/collections/${collection.id}`)
    } catch (error) {
        const apiError = error as { status?: number }
        addToast({
            title: t.value.toastErrorTitle,
            description:
                apiError.status === 503
                    ? t.value.collectionsGenerateUnavailableDescription
                    : t.value.collectionsGenerateErrorDescription,
            variant: 'destructive',
            duration: 6000,
        })
    } finally {
        isGenerating.value = false
    }
}

onMounted(async () => {
    await fetchCollections(1)
})

onBeforeUnmount(() => {
    if (searchDebounceTimer) {
        clearTimeout(searchDebounceTimer)
    }
})
</script>
