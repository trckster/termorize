<template>
    <Header />
    <main class="px-6 py-8">
        <div class="max-w-6xl mx-auto">
            <h1 class="text-3xl font-bold mb-8 text-foreground">Saved Words</h1>
            <Pagination
                v-slot="{ page }"
                :items-per-page="paginationData.page_size"
                :total="paginationData.total"
                :default-page="1"
            >
                <PaginationContent v-slot="{ items }">
                    <PaginationPrevious />
                    <template v-for="(item, index) in items" :key="index">
                        <PaginationItem
                            v-if="item.type === 'page'"
                            :value="item.value"
                            :is-active="item.value === page"
                        >
                            {{ item.value }}
                        </PaginationItem>
                    </template>
                    <PaginationEllipsis :index="4" />
                    <PaginationNext />
                </PaginationContent>
            </Pagination>
            <div class="space-y-2">
                <div
                    v-for="item in vocabulary"
                    :key="item.id"
                    class="p-4 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors cursor-pointer"
                >
                    <div class="flex items-start justify-between">
                        <div>
                            <h3 class="font-semibold text-foreground">{{ item.translation.word_1.word }}</h3>
                            <p class="text-sm text-muted-foreground mt-1">{{ item.translation.word_2.word }}</p>
                        </div>
                        <span class="text-xs font-medium text-primary bg-primary/10 px-2.5 py-1 rounded">
                            {{ item.translation.word_1.language }}
                        </span>
                    </div>
                </div>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary.ts'
import { onMounted, ref } from 'vue'
import { Pagination, PaginationContent, PaginationItem, PaginationPrevious } from '@/components/ui/pagination'
import type { PaginationData } from '@/api/pagination.ts'

const vocabulary = ref<VocabularyItem[]>([])
const paginationData = ref<PaginationData>({
    page: 1,
    page_size: 50,
    total: 0,
    total_pages: 0,
})

onMounted(async () => {
    const response = await vocabularyApi.getVocabulary(1, 100)
    vocabulary.value = response.data
    paginationData.value = response.pagination
})
</script>
