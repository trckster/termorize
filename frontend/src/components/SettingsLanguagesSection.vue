<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UserSettings } from '@/api/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { useI18n } from '@/composables/useI18n'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{
    settings?: UserSettings
}>()

const settingsStore = useSettingsStore()
const { t } = useI18n()

const supportedSystemLanguages = ['en', 'ru']

const getSystemLanguageValue = (language?: string) => (language === 'ru' ? 'ru' : 'en')

const systemLanguage = ref(getSystemLanguageValue(props.settings?.system_language))
const mainLearningLanguage = ref(props.settings?.main_learning_language || '')
const ignoredAudioLanguages = ref<string[]>([...(props.settings?.ignored_audio_languages ?? [])])
const ignoredDescriptionLanguages = ref<string[]>([...(props.settings?.ignored_description_languages ?? [])])

const hasLanguageSettingsChanged = computed(() => {
    if (!props.settings) return false

    return (
        systemLanguage.value !== props.settings.system_language ||
        mainLearningLanguage.value !== props.settings.main_learning_language ||
        ignoredAudioLanguages.value.join(',') !== (props.settings.ignored_audio_languages ?? []).join(',') ||
        ignoredDescriptionLanguages.value.join(',') !== (props.settings.ignored_description_languages ?? []).join(',')
    )
})

watch(
    () => props.settings,
    (nextSettings) => {
        systemLanguage.value = getSystemLanguageValue(nextSettings?.system_language)
        mainLearningLanguage.value = nextSettings?.main_learning_language || ''
        ignoredAudioLanguages.value = [...(nextSettings?.ignored_audio_languages ?? [])]
        ignoredDescriptionLanguages.value = [...(nextSettings?.ignored_description_languages ?? [])]
    },
    { immediate: true }
)

const setAudioLanguageIgnored = (language: string, ignored: boolean) => {
    const selected = new Set(ignoredAudioLanguages.value)
    if (ignored) selected.add(language)
    else selected.delete(language)
    ignoredAudioLanguages.value = settingsStore.languageOptions
        .map((option) => option.code)
        .filter((code) => selected.has(code))
}

const setDescriptionLanguageIgnored = (language: string, ignored: boolean) => {
    const selected = new Set(ignoredDescriptionLanguages.value)
    if (ignored) selected.add(language)
    else selected.delete(language)
    ignoredDescriptionLanguages.value = settingsStore.languageOptions
        .map((option) => option.code)
        .filter((code) => selected.has(code))
}
const changes = computed<Partial<UserSettings> | null>(() =>
    hasLanguageSettingsChanged.value
        ? {
              system_language: getSystemLanguageValue(systemLanguage.value),
              main_learning_language: mainLearningLanguage.value,
              ignored_audio_languages: [...ignoredAudioLanguages.value],
              ignored_description_languages: [...ignoredDescriptionLanguages.value],
          }
        : null
)
defineExpose({ changes })
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>{{ t.settingsLanguagesTitle }}</CardTitle>
            <CardDescription>{{ t.settingsLanguagesDescription }}</CardDescription>
        </CardHeader>
        <CardContent>
            <div class="space-y-4">
                <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div class="space-y-2 sm:rounded-lg sm:p-4">
                        <p class="text-sm font-semibold text-foreground">{{ t.settingsSystemLanguageTitle }}</p>
                        <LanguageSelector
                            v-model="systemLanguage"
                            :allowed-values="supportedSystemLanguages"
                            :placeholder="t.settingsSystemLanguagePlaceholder"
                            :aria-label="t.settingsSystemLanguageTitle"
                            name="system-language"
                        />
                        <p class="text-xs text-muted-foreground">
                            {{ t.settingsSystemLanguageNote }}
                        </p>
                    </div>

                    <div class="space-y-2 sm:rounded-lg sm:p-4">
                        <p class="text-sm font-semibold text-foreground">{{ t.settingsMainLearningLanguageTitle }}</p>
                        <LanguageSelector
                            v-model="mainLearningLanguage"
                            :placeholder="t.settingsMainLearningLanguagePlaceholder"
                            :aria-label="t.settingsMainLearningLanguageTitle"
                            name="main-learning-language"
                        />
                        <p class="text-xs text-muted-foreground">
                            {{ t.settingsMainLearningLanguageNote }}
                        </p>
                    </div>
                </div>

                <div class="space-y-3 border-t border-border pt-4 sm:mx-4">
                    <div class="space-y-1">
                        <p class="text-sm font-semibold text-foreground">{{ t.settingsIgnoredAudioLanguagesTitle }}</p>
                        <p class="text-xs leading-5 text-muted-foreground">
                            {{ t.settingsIgnoredAudioLanguagesNote }}
                        </p>
                    </div>
                    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                        <label
                            v-for="language in settingsStore.languageOptions"
                            :key="language.code"
                            class="flex min-h-11 cursor-pointer items-center gap-3 rounded-md border border-border px-3 py-2 text-sm transition-colors hover:bg-accent/50"
                        >
                            <input
                                type="checkbox"
                                class="h-4 w-4 rounded border-input accent-primary"
                                :checked="ignoredAudioLanguages.includes(language.code)"
                                @change="
                                    setAudioLanguageIgnored(language.code, ($event.target as HTMLInputElement).checked)
                                "
                            />
                            <span aria-hidden="true" class="text-base">{{ language.emoji }}</span>
                            <span class="font-medium text-foreground">
                                {{ settingsStore.getLanguageName(language.code, systemLanguage) }}
                            </span>
                        </label>
                    </div>
                </div>

                <div class="space-y-3 border-t border-border pt-4 sm:mx-4">
                    <div class="space-y-1">
                        <p class="text-sm font-semibold text-foreground">
                            {{ t.settingsIgnoredDescriptionLanguagesTitle }}
                        </p>
                        <p class="text-xs leading-5 text-muted-foreground">
                            {{ t.settingsIgnoredDescriptionLanguagesNote }}
                        </p>
                    </div>
                    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
                        <label
                            v-for="language in settingsStore.languageOptions"
                            :key="language.code"
                            class="flex min-h-11 cursor-pointer items-center gap-3 rounded-md border border-border px-3 py-2 text-sm transition-colors hover:bg-accent/50"
                        >
                            <input
                                type="checkbox"
                                class="h-4 w-4 rounded border-input accent-primary"
                                :checked="ignoredDescriptionLanguages.includes(language.code)"
                                @change="
                                    setDescriptionLanguageIgnored(
                                        language.code,
                                        ($event.target as HTMLInputElement).checked
                                    )
                                "
                            />
                            <span aria-hidden="true" class="text-base">{{ language.emoji }}</span>
                            <span class="font-medium text-foreground">
                                {{ settingsStore.getLanguageName(language.code, systemLanguage) }}
                            </span>
                        </label>
                    </div>
                </div>
            </div>
        </CardContent>
    </Card>
</template>
