<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import { settingsApi } from '@/api/settings.ts'
import { useToast } from '@/composables/useToast.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const authStore = useAuthStore()
const { addToast } = useToast()
const { t } = useI18n()

const supportedSystemLanguages = ['en', 'ru']

const getSystemLanguageValue = (language?: string) => (language === 'ru' ? 'ru' : 'en')

const systemLanguage = ref(getSystemLanguageValue(props.settings?.system_language))
const mainLearningLanguage = ref(props.settings?.main_learning_language || '')
const isSaving = ref(false)

const hasLanguageSettingsChanged = computed(() => {
    if (!props.settings) return false

    return (
        systemLanguage.value !== props.settings.system_language ||
        mainLearningLanguage.value !== props.settings.main_learning_language
    )
})

const saveLanguageSettings = async () => {
    if (!props.settings || !hasLanguageSettingsChanged.value || isSaving.value) return

    isSaving.value = true

    try {
        authStore.user = await settingsApi.updateSettings({
            ...props.settings,
            system_language: getSystemLanguageValue(systemLanguage.value),
            main_learning_language: mainLearningLanguage.value,
        })

        addToast({
            title: t.value.toastSavedTitle,
            description: t.value.toastSavedDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        console.error('Failed to save settings:', error)
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.toastSaveErrorDescription,
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
        systemLanguage.value = getSystemLanguageValue(nextSettings?.system_language)
        mainLearningLanguage.value = nextSettings?.main_learning_language || ''
    },
    { immediate: true }
)
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>{{ t.settingsLanguagesTitle }}</CardTitle>
            <CardDescription>{{ t.settingsLanguagesDescription }}</CardDescription>
        </CardHeader>
        <CardContent class="space-y-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">{{ t.settingsSystemLanguageTitle }}</p>
                    <LanguageSelector
                        v-model="systemLanguage"
                        :allowed-values="supportedSystemLanguages"
                        :placeholder="t.settingsSystemLanguagePlaceholder"
                    />
                    <p class="text-xs text-muted-foreground">
                        {{ t.settingsSystemLanguageNote }}
                    </p>
                </div>

                <div class="space-y-2 rounded-lg p-4">
                    <p class="text-sm font-semibold text-foreground">{{ t.settingsMainLearningLanguageTitle }}</p>
                    <LanguageSelector v-model="mainLearningLanguage" :placeholder="t.settingsMainLearningLanguagePlaceholder" />
                    <p class="text-xs text-muted-foreground">
                        {{ t.settingsMainLearningLanguageNote }}
                    </p>
                </div>
            </div>

            <div class="px-4" v-if="hasLanguageSettingsChanged">
                <Button :disabled="isSaving" @click="saveLanguageSettings">
                    {{ isSaving ? t.saving : t.save }}
                </Button>
            </div>
        </CardContent>
    </Card>
</template>
