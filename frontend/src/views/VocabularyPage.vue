<template>
    <Header />
    <main class="px-6 py-8">
        <div class="max-w-4xl mx-auto">
            <h1 class="text-3xl font-bold mb-8 text-foreground">Saved Words</h1>
            <div class="space-y-3">
                <div
                    v-for="word in savedWords"
                    :key="word.id"
                    class="p-4 rounded-lg border border-border bg-card hover:bg-accent/50 transition-colors cursor-pointer"
                >
                    <div class="flex items-start justify-between">
                        <div>
                            <h3 class="font-semibold text-foreground">{{ word.word }}</h3>
                            <p class="text-sm text-muted-foreground mt-1">{{ word.definition }}</p>
                        </div>
                        <span class="text-xs font-medium text-primary bg-primary/10 px-2.5 py-1 rounded">
                            {{ word.language }}
                        </span>
                    </div>
                    <p v-if="word.example" class="text-sm text-muted-foreground mt-2 italic">"{{ word.example }}"</p>
                </div>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import Header from '@/components/Header.vue'

interface SavedWord {
    id: number
    word: string
    definition: string
    language: string
    example?: string
}

const savedWords: SavedWord[] = [
    {
        id: 1,
        word: 'Serendipity',
        definition: 'The occurrence of events by chance in a happy or beneficial way',
        language: 'English',
        example: 'Finding that old photo was pure serendipity.',
    },
    {
        id: 2,
        word: 'Eloquent',
        definition: 'Fluent or persuasive in speaking or writing',
        language: 'English',
        example: 'The speaker gave an eloquent speech about climate change.',
    },
    {
        id: 3,
        word: 'Ephemeral',
        definition: 'Lasting for a very short time; transitory',
        language: 'English',
        example: 'The beauty of cherry blossoms is ephemeral.',
    },
    {
        id: 4,
        word: 'Ubiquitous',
        definition: 'Present, appearing, or found everywhere',
        language: 'English',
        example: 'Smartphones have become ubiquitous in modern society.',
    },
    {
        id: 5,
        word: 'Perspicacious',
        definition: 'Having a ready insight into and understanding of things',
        language: 'English',
        example: 'Her perspicacious analysis of the market trends proved accurate.',
    },
]
</script>
