<script setup lang="ts">
import { computed } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const settingsStore = useSettingsStore()

const getLanguageLabel = (code?: string) => {
    if (!code) return 'Not set'

    const option = settingsStore.languageOptions.find((lang) => lang.code === code)
    if (!option) return code.toUpperCase()

    return `${option.emoji} ${option.name}`
}

const fields = computed(() => {
    return [
        {
            key: 'native_language',
            label: 'Native Language',
            value: getLanguageLabel(props.settings?.native_language),
            explanation:
                'This is your main language. We use it in quizzes to explain vocabulary words and crossword tasks.',
        },
        {
            key: 'main_learning_language',
            label: 'Main Learning Language',
            value: getLanguageLabel(props.settings?.main_learning_language),
            explanation: 'This is the language you are focusing on in your daily learning flow.',
        },
    ]
})
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>Languages</CardTitle>
            <CardDescription>Language preferences used in translation and learning.</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div v-for="field in fields" :key="field.key" class="rounded-lg border border-border p-4 space-y-2">
                <p class="text-sm font-semibold text-foreground">{{ field.label }}</p>
                <p class="text-sm text-foreground">{{ field.value }}</p>
                <p class="text-xs text-muted-foreground">{{ field.explanation }}</p>
            </div>
        </CardContent>
    </Card>
</template>
