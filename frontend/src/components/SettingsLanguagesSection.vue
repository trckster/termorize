<script setup lang="ts">
import { ref, watch } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const nativeLanguage = ref(props.settings?.native_language || '')
const mainLearningLanguage = ref(props.settings?.main_learning_language || '')

watch(
    () => props.settings,
    (nextSettings) => {
        nativeLanguage.value = nextSettings?.native_language || ''
        mainLearningLanguage.value = nextSettings?.main_learning_language || ''
    },
    { immediate: true }
)
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Languages</CardTitle>
            <CardDescription>Language preferences used in translation and learning.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div class="space-y-2">
                <p class="text-sm font-semibold text-foreground">Native Language</p>
                <LanguageSelector v-model="nativeLanguage" placeholder="Select native language" />
                <p class="text-xs text-muted-foreground">
                    This is your main language. We use it in quizzes to explain vocabulary words and crossword tasks.
                </p>
            </div>

            <div class="space-y-2">
                <p class="text-sm font-semibold text-foreground">Main Learning Language</p>
                <LanguageSelector v-model="mainLearningLanguage" placeholder="Select learning language" />
                <p class="text-xs text-muted-foreground">
                    This is the language you are focusing on in your daily learning flow.
                </p>
            </div>
        </CardContent>
    </Card>
</template>
