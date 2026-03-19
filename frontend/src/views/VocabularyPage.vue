<template>
    <main class="px-6 py-8">
        <div class="max-w-6xl mx-auto">
            <div class="flex justify-between items-center mb-8">
                <h1 class="text-3xl font-bold text-foreground">{{ t.vocabularyTitle }}</h1>
                <Dialog v-model:open="isAddDialogOpen">
                    <DialogTrigger as-child>
                        <Button>
                            <Plus class="h-4 w-4 mr-2" />
                            {{ t.vocabularyAddButton }}
                        </Button>
                    </DialogTrigger>
                    <DialogContent class="sm:max-w-md">
                        <DialogHeader>
                            <DialogTitle>{{ t.vocabularyDialogTitle }}</DialogTitle>
                            <DialogDescription>
                                {{ t.vocabularyDialogDescription }}
                            </DialogDescription>
                        </DialogHeader>
                        <form @submit.prevent="handleAdd" class="space-y-4 py-4">
                            <div class="grid grid-cols-2 gap-4">
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">{{ t.vocabularyLanguage1 }}</label>
                                    <LanguageSelector
                                        v-model="newTranslation.language1"
                                        :placeholder="t.vocabularySelectLanguagePlaceholder"
                                        :disabled-values="[newTranslation.language2]"
                                        aria-label="Language 1"
                                    />
                                </div>
                                <div class="space-y-2">
                                    <label for="vocab-word1" class="text-sm font-medium">{{ t.vocabularyWord1 }}</label>
                                    <input
                                        id="vocab-word1"
                                        v-model="newTranslation.word1"
                                        type="text"
                                        :placeholder="t.vocabularyWord1Placeholder"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                            </div>
                            <div class="grid grid-cols-2 gap-4">
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">{{ t.vocabularyLanguage2 }}</label>
                                    <LanguageSelector
                                        v-model="newTranslation.language2"
                                        :placeholder="t.vocabularySelectLanguagePlaceholder"
                                        :disabled-values="[newTranslation.language1]"
                                        aria-label="Language 2"
                                    />
                                </div>
                                <div class="space-y-2">
                                    <label for="vocab-word2" class="text-sm font-medium">{{ t.vocabularyWord2 }}</label>
                                    <input
                                        id="vocab-word2"
                                        v-model="newTranslation.word2"
                                        type="text"
                                        :placeholder="t.vocabularyWord2Placeholder"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                            </div>
                            <DialogFooter class="justify-center sm:justify-center pt-4">
                                <Button type="submit" :disabled="isAdding || !isFormValid">
                                    <Loader2 v-if="isAdding" class="mr-2 h-4 w-4 animate-spin" />
                                    {{ isAdding ? t.adding : t.vocabularyAddButton }}
                                </Button>
                            </DialogFooter>
                        </form>
                    </DialogContent>
                </Dialog>
            </div>

            <div v-if="vocabulary.length > 0" class="space-y-2 mb-8">
                <div
                    v-for="item in vocabulary"
                    :key="item.id"
                    class="p-4 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors cursor-pointer group"
                >
                    <div class="grid grid-cols-1 md:grid-cols-12 gap-4 items-center">
                        <!-- Part 1: Words -->
                        <div class="md:col-span-4">
                            <h3 class="font-semibold text-foreground flex items-center gap-2">
                                <span
                                    class="text-xl"
                                    role="img"
                                    :aria-label="getLanguageName(item.translation.original.language)"
                                >{{ settingsStore.getFlag(item.translation.original.language) }}</span>
                                <span class="text-lg">{{ item.translation.original.word }}</span>
                                <span class="text-muted-foreground">-</span>
                                <span class="text-lg">{{ item.translation.translation.word }}</span>
                                <span
                                    class="text-xl"
                                    role="img"
                                    :aria-label="getLanguageName(item.translation.translation.language)"
                                >{{ settingsStore.getFlag(item.translation.translation.language) }}</span>
                            </h3>
                        </div>

                        <!-- Part 2: Progress -->
                        <div class="md:col-span-5 flex flex-col gap-3">
                            <div v-if="item.progress && item.progress.length > 0">
                                <div v-for="(prog, idx) in item.progress" :key="idx" class="w-full">
                                    <Progress :model-value="prog.knowledge" class="h-2" />
                                    <div class="flex justify-between items-center mt-1">
                                        <span class="text-xs text-muted-foreground capitalize">{{ prog.type }}</span>
                                        <span class="text-xs font-medium">{{ Math.round(prog.knowledge) }}%</span>
                                    </div>
                                </div>
                            </div>
                            <div v-else class="text-sm text-muted-foreground text-center py-2">
                                {{ t.vocabularyNoProgress }}
                            </div>
                        </div>

                        <!-- Part 3: Date and Delete -->
                        <div class="md:col-span-3 flex justify-end items-center gap-2">
                            <Tooltip>
                                <TooltipTrigger as-child>
                                    <span
                                        class="text-xs font-medium text-muted-foreground bg-secondary/50 px-2.5 py-1 rounded whitespace-nowrap"
                                    >
                                        {{ formatRelativeTime(item.created_at) }}
                                    </span>
                                </TooltipTrigger>
                                <TooltipContent>
                                    <p>{{ t.vocabularyCreatedAt }} {{ formatDate(item.created_at) }}</p>
                                    <p v-if="item.mastered_at">{{ t.vocabularyMasteredAt }} {{ formatDate(item.mastered_at) }}</p>
                                </TooltipContent>
                            </Tooltip>
                            <Dialog>
                                <DialogTrigger as-child>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        class="text-muted-foreground opacity-100 md:opacity-0 md:group-hover:opacity-100 focus:opacity-100 transition-opacity hover:text-destructive hover:bg-destructive/10"
                                        :aria-label="`Delete ${item.translation.original.word} — ${item.translation.translation.word}`"
                                        @click.stop
                                    >
                                        <Trash2 class="h-4 w-4" />
                                    </Button>
                                </DialogTrigger>
                                <DialogContent class="sm:max-w-md">
                                    <DialogHeader>
                                        <DialogTitle>{{ t.vocabularyDeleteDialogTitle }}</DialogTitle>
                                        <DialogDescription>
                                            {{ t.vocabularyDeleteConfirmPrefix }}<span
                                                class="font-medium text-foreground"
                                                >{{ item.translation.original.word }}</span
                                            >
                                            -
                                            <span class="font-medium text-foreground">{{
                                                item.translation.translation.word
                                            }}</span>{{ t.vocabularyDeleteConfirmSuffix }}
                                        </DialogDescription>
                                    </DialogHeader>
                                    <DialogFooter class="flex gap-2 sm:justify-end">
                                        <DialogClose as-child>
                                            <Button type="button" variant="secondary"> {{ t.cancel }} </Button>
                                        </DialogClose>
                                        <DialogClose as-child>
                                            <Button
                                                type="button"
                                                variant="destructive"
                                                @click="handleDelete(item.id)"
                                                :disabled="deletingId === item.id"
                                            >
                                                <Loader2
                                                    v-if="deletingId === item.id"
                                                    class="mr-2 h-4 w-4 animate-spin"
                                                />
                                                {{ deletingId === item.id ? t.deleting : t.delete }}
                                            </Button>
                                        </DialogClose>
                                    </DialogFooter>
                                </DialogContent>
                            </Dialog>
                        </div>
                    </div>
                </div>
            </div>

            <div
                v-else-if="!isLoadingVocabulary"
                class="mb-8 flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 px-6 text-center"
            >
                <h2 class="text-xl font-semibold text-foreground">{{ t.vocabularyEmptyTitle }}</h2>
                <p class="mt-2 max-w-md text-sm text-muted-foreground">
                    {{ t.vocabularyEmptyDescription }}
                </p>
                <Button class="mt-5" @click="isAddDialogOpen = true">
                    <Plus class="mr-2 h-4 w-4" />
                    {{ t.vocabularyAddButton }}
                </Button>
            </div>

            <Pagination
                v-if="paginationData.total > 0"
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
    </main>
</template>

<script setup lang="ts">
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary.ts'
import { onMounted, ref, computed, watch } from 'vue'
import { useAuthStore } from '@/stores/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { useI18n } from '@/composables/useI18n'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Pagination, PaginationContent, PaginationItem, PaginationEllipsis } from '@/components/ui/pagination'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import type { PaginationData } from '@/api/pagination.ts'
import { formatRelativeTime, formatDate } from '@/lib/utils.ts'
import { Progress } from '@/components/ui/progress'
import { Trash2, Loader2, Plus } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast.ts'

const { t } = useI18n()

const vocabulary = ref<VocabularyItem[]>([])
const currentPage = ref(1)
const paginationData = ref<PaginationData>({
    page: 1,
    page_size: 20,
    total: 0,
    total_pages: 0,
})
const deletingId = ref<string | null>(null)
const isAddDialogOpen = ref(false)
const isAdding = ref(false)
const isLoadingVocabulary = ref(false)

const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const getLanguageName = (code: string) =>
    settingsStore.languageOptions.find((l) => l.code === code)?.name || code.toUpperCase()

type NewTranslationForm = {
    word1: string
    word2: string
    language1: string
    language2: string
}

const defaultNewTranslation = (): NewTranslationForm => ({
    word1: '',
    word2: '',
    language1: authStore.user?.settings.translation_source_language || 'en',
    language2:
        authStore.user?.settings.translation_target_language === authStore.user?.settings.translation_source_language
            ? authStore.user?.settings.translation_source_language === 'en'
                ? 'ru'
                : 'en'
            : authStore.user?.settings.translation_target_language || 'ru',
})

const newTranslation = ref<NewTranslationForm>(defaultNewTranslation())

const isFormValid = computed(() => {
    return newTranslation.value.word1.trim().length > 0 && newTranslation.value.word2.trim().length > 0
})

const resetForm = () => {
    newTranslation.value = defaultNewTranslation()
}

watch(isAddDialogOpen, (isOpen) => {
    if (isOpen) {
        resetForm()
    }
})

const { addToast } = useToast()

const handleAdd = async () => {
    if (!isFormValid.value) return

    isAdding.value = true
    try {
        await vocabularyApi.addVocabulary(
            newTranslation.value.word1.trim(),
            newTranslation.value.word2.trim(),
            newTranslation.value.language1,
            newTranslation.value.language2
        )
        isAddDialogOpen.value = false
        resetForm()
        await fetchVocabulary(currentPage.value)

        addToast({
            title: t.value.vocabularyToastSuccessTitle,
            description: t.value.vocabularyToastSuccessDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        const apiError = error as { status?: number }
        if (apiError.status === 409) {
            addToast({
                title: t.value.vocabularyToastAlreadyExistsTitle,
                description: t.value.vocabularyToastAlreadyExistsDescription,
                duration: 3000,
            })
            return
        }
        addToast({
            title: t.value.vocabularyToastErrorTitle,
            description: t.value.vocabularyToastErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isAdding.value = false
    }
}

const fetchVocabulary = async (page: number) => {
    isLoadingVocabulary.value = true
    currentPage.value = page
    try {
        const response = await vocabularyApi.getVocabulary(page, paginationData.value.page_size)
        vocabulary.value = response.data
        paginationData.value = response.pagination
    } finally {
        isLoadingVocabulary.value = false
    }
}

const handlePageChange = async (page: number) => {
    await fetchVocabulary(page)
}

const handleDelete = async (id: string) => {
    deletingId.value = id
    try {
        await vocabularyApi.deleteVocabulary(id)
        await fetchVocabulary(currentPage.value)
    } catch (error) {
        console.error('Failed to delete vocabulary item:', error)
    } finally {
        deletingId.value = null
    }
}

onMounted(async () => {
    await fetchVocabulary(1)
})
</script>
