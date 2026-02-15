<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import { settingsApi } from '@/api/settings.ts'
import { useToast } from '@/composables/useToast.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { Button } from '@/components/ui/button'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const authStore = useAuthStore()
const { addToast } = useToast()

const nativeLanguage = ref(props.settings?.native_language || '')
const mainLearningLanguage = ref(props.settings?.main_learning_language || '')
const isSaving = ref(false)

const hasLanguageSettingsChanged = computed(() => {
    if (!props.settings) return false

    return (
        nativeLanguage.value !== props.settings.native_language ||
        mainLearningLanguage.value !== props.settings.main_learning_language
    )
})

const saveLanguageSettings = async () => {
    if (!props.settings || !hasLanguageSettingsChanged.value || isSaving.value) return

    isSaving.value = true

    try {
        authStore.user = await settingsApi.updateSettings({
            ...props.settings,
            native_language: nativeLanguage.value,
            main_learning_language: mainLearningLanguage.value,
        })

        addToast({
            title: 'Saved',
            description: 'Settings were saved successfully.',
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        console.error('Failed to save settings:', error)
        addToast({
            title: 'Error',
            description: 'Failed to save settings. Please try again.',
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSaving.value = false
    }
}

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
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">Native Language</p>
                    <LanguageSelector v-model="nativeLanguage" placeholder="Select native language" />
                    <p class="text-xs text-muted-foreground">
                        This is your main language. We use it in quizzes to explain vocabulary words and crossword
                        tasks.
                    </p>
                </div>

                <div class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">Main Learning Language</p>
                    <LanguageSelector v-model="mainLearningLanguage" placeholder="Select learning language" />
                    <p class="text-xs text-muted-foreground">
                        This is the language you are focusing on in your daily learning flow.
                    </p>
                </div>
            </div>

            <div class="px-4" v-if="hasLanguageSettingsChanged">
                <Button :disabled="isSaving" @click="saveLanguageSettings">
                    {{ isSaving ? 'Saving...' : 'Save' }}
                </Button>
            </div>
        </CardContent>
    </Card>
</template>
