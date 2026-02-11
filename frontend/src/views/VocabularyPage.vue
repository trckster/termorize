<template>
    <Header />
    <main class="px-6 py-8">
        <div class="max-w-6xl mx-auto">
            <h1 class="text-3xl font-bold mb-8 text-foreground">Saved Words</h1>

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
                                <span class="text-xl">{{ languageToEmoji(item.translation.word_1.language) }}</span>
                                <span class="text-lg">{{ item.translation.word_1.word }}</span>
                                <span class="text-muted-foreground">-</span>
                                <span class="text-lg">{{ item.translation.word_2.word }}</span>
                                <span class="text-xl">{{ languageToEmoji(item.translation.word_2.language) }}</span>
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
                                                >{{ item.translation.word_1.word }}</span
                                            >
                                            -
                                            <span class="font-medium text-foreground">{{
                                                item.translation.word_2.word
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
import Header from '@/components/Header.vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary.ts'
import { onMounted, ref } from 'vue'
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
import { languageToEmoji, formatRelativeTime, formatDate } from '@/lib/utils.ts'
import { Progress } from '@/components/ui/progress'
import { Trash2, Loader2 } from 'lucide-vue-next'

const vocabulary = ref<VocabularyItem[]>([])
const currentPage = ref(1)
const paginationData = ref<PaginationData>({
    page: 1,
    page_size: 20,
    total: 0,
    total_pages: 0,
})
const deletingId = ref<string | null>(null)

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
