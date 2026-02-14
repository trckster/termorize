<template>
    <main class="px-6 py-8">
        <div class="max-w-6xl mx-auto">
            <div class="flex justify-between items-center mb-8">
                <h1 class="text-3xl font-bold text-foreground">Saved Words</h1>
                <Dialog v-model:open="isAddDialogOpen">
                    <DialogTrigger as-child>
                        <Button class="bg-green-600 hover:bg-green-700 text-white">
                            <Plus class="h-4 w-4 mr-2" />
                            Add Translation
                        </Button>
                    </DialogTrigger>
                    <DialogContent class="sm:max-w-md">
                        <DialogHeader>
                            <DialogTitle>Add Translation</DialogTitle>
                            <DialogDescription>
                                Enter two words and their languages to add a new translation.
                            </DialogDescription>
                        </DialogHeader>
                        <form @submit.prevent="handleAdd" class="space-y-4 py-4">
                            <div class="grid grid-cols-2 gap-4">
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">Language 1</label>
                                    <select
                                        v-model="newTranslation.language1"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    >
                                        <option
                                            v-for="lang in availableLanguages"
                                            :key="lang.code"
                                            :value="lang.code"
                                            :disabled="lang.code === newTranslation.language2"
                                        >
                                            {{ lang.name }}
                                        </option>
                                    </select>
                                </div>
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">Word 1</label>
                                    <input
                                        v-model="newTranslation.word1"
                                        type="text"
                                        placeholder="Enter word"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                            </div>
                            <div class="grid grid-cols-2 gap-4">
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">Language 2</label>
                                    <select
                                        v-model="newTranslation.language2"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    >
                                        <option
                                            v-for="lang in availableLanguages"
                                            :key="lang.code"
                                            :value="lang.code"
                                            :disabled="lang.code === newTranslation.language1"
                                        >
                                            {{ lang.name }}
                                        </option>
                                    </select>
                                </div>
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">Word 2</label>
                                    <input
                                        v-model="newTranslation.word2"
                                        type="text"
                                        placeholder="Enter translation"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                            </div>
                            <DialogFooter class="justify-center sm:justify-center pt-4">
                                <Button
                                    type="submit"
                                    class="bg-green-600 hover:bg-green-700 text-white"
                                    :disabled="isAdding || !isFormValid"
                                >
                                    <Loader2 v-if="isAdding" class="mr-2 h-4 w-4 animate-spin" />
                                    {{ isAdding ? 'Adding...' : 'Add Translation' }}
                                </Button>
                            </DialogFooter>
                        </form>
                    </DialogContent>
                </Dialog>
            </div>

            <div class="space-y-2 mb-8">
                <div
                    v-for="item in vocabulary"
                    :key="item.id"
                    class="p-4 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors cursor-pointer group"
                >
                    <div class="grid grid-cols-1 md:grid-cols-12 gap-4 items-center">
                        <!-- Part 1: Words -->
                        <div class="md:col-span-4">
                            <h3 class="font-semibold text-foreground flex items-center gap-2">
                                <span class="text-xl">{{
                                    settingsStore.getFlag(item.translation.original.language)
                                }}</span>
                                <span class="text-lg">{{ item.translation.original.word }}</span>
                                <span class="text-muted-foreground">-</span>
                                <span class="text-lg">{{ item.translation.translation.word }}</span>
                                <span class="text-xl">{{
                                    settingsStore.getFlag(item.translation.translation.language)
                                }}</span>
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
                                No progress recorded
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
                                    <p>Created: {{ formatDate(item.created_at) }}</p>
                                    <p v-if="item.mastered_at">Mastered: {{ formatDate(item.mastered_at) }}</p>
                                </TooltipContent>
                            </Tooltip>
                            <Dialog>
                                <DialogTrigger as-child>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        class="h-8 w-8 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity hover:text-destructive hover:bg-destructive/10"
                                        @click.stop
                                    >
                                        <Trash2 class="h-4 w-4" />
                                    </Button>
                                </DialogTrigger>
                                <DialogContent class="sm:max-w-md">
                                    <DialogHeader>
                                        <DialogTitle>Delete Vocabulary Item</DialogTitle>
                                        <DialogDescription>
                                            Are you sure you want to delete "<span
                                                class="font-medium text-foreground"
                                                >{{ item.translation.original.word }}</span
                                            >
                                            -
                                            <span class="font-medium text-foreground">{{
                                                item.translation.translation.word
                                            }}</span
                                            >"? This action cannot be undone.
                                        </DialogDescription>
                                    </DialogHeader>
                                    <DialogFooter class="flex gap-2 sm:justify-end">
                                        <DialogClose as-child>
                                            <Button type="button" variant="secondary"> Cancel </Button>
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
                                                {{ deletingId === item.id ? 'Deleting...' : 'Delete' }}
                                            </Button>
                                        </DialogClose>
                                    </DialogFooter>
                                </DialogContent>
                            </Dialog>
                        </div>
                    </div>
                </div>
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
                    <PaginationFirst />
                    <PaginationPrevious />

                    <template v-for="(item, index) in items">
                        <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value" as-child>
                            <Button class="w-9 h-9 p-0" :variant="item.value === page ? 'default' : 'outline'">
                                {{ item.value }}
                            </Button>
                        </PaginationItem>
                        <PaginationEllipsis v-else :key="item.type + index" :index="index" />
                    </template>

                    <PaginationNext />
                    <PaginationLast />
                </PaginationContent>
            </Pagination>
        </div>
    </main>
</template>

<script setup lang="ts">
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary.ts'
import { onMounted, ref, computed } from 'vue'
import { useSettingsStore } from '@/stores/settings.ts'
import {
    Pagination,
    PaginationContent,
    PaginationItem,
    PaginationPrevious,
    PaginationNext,
    PaginationEllipsis,
    PaginationFirst,
    PaginationLast,
} from '@/components/ui/pagination'
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

const settingsStore = useSettingsStore()
const availableLanguages = computed(() => settingsStore.languageOptions)

const newTranslation = ref({
    word1: '',
    word2: '',
    language1: 'en',
    language2: 'ru',
})

const isFormValid = computed(() => {
    return newTranslation.value.word1.trim() && newTranslation.value.word2.trim()
})

const resetForm = () => {
    newTranslation.value = {
        word1: '',
        word2: '',
        language1: 'en',
        language2: 'ru',
    }
}

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
            title: 'Success!',
            description: 'Translation added successfully.',
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        console.error('Failed to add translation:', error)
        addToast({
            title: 'Error',
            description: 'Failed to add translation. Please try again.',
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isAdding.value = false
    }
}

const fetchVocabulary = async (page: number) => {
    currentPage.value = page
    const response = await vocabularyApi.getVocabulary(page, paginationData.value.page_size)
    vocabulary.value = response.data
    paginationData.value = response.pagination
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
