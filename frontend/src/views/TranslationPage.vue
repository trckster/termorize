<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { settingsApi } from '@/api/settings.ts'
import { translationApi } from '@/api/translation.ts'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Kbd } from '@/components/ui/kbd'
import { Button } from '@/components/ui/button'
import { useToast } from '@/composables/useToast.ts'
import { usePhoneViewport } from '@/composables/usePhoneViewport.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { useI18n } from '@/composables/useI18n'

type LanguageSelectorInstance = {
    focusInput: () => Promise<void>
}

const authStore = useAuthStore()
const { isPhoneViewport } = usePhoneViewport()
const { t } = useI18n()

const getDistinctTargetLanguage = (sourceLanguage: string, targetLanguage: string) => {
    if (sourceLanguage !== targetLanguage) {
        return targetLanguage
    }

    return sourceLanguage === 'en' ? 'ru' : 'en'
}

const getInitialLanguages = () => {
    const sourceLanguage = authStore.user?.settings.translation_source_language || 'en'
    const targetLanguage = getDistinctTargetLanguage(
        sourceLanguage,
        authStore.user?.settings.translation_target_language || 'ru'
    )

    return {
        source: sourceLanguage,
        target: targetLanguage,
    }
}

const initialLanguages = getInitialLanguages()

const sourceText = ref('')
const translatedText = ref('')
const sourceTextareaRef = ref<HTMLTextAreaElement | null>(null)
const targetTextareaRef = ref<HTMLTextAreaElement | null>(null)
const sourceLanguageSelectorRef = ref<LanguageSelectorInstance | null>(null)
const targetLanguageSelectorRef = ref<LanguageSelectorInstance | null>(null)
const sourceLang = ref(initialLanguages.source)
const targetLang = ref(initialLanguages.target)
const translationId = ref<string | null>(null)
const isSavingVocabulary = ref(false)

const { addToast } = useToast()

let debounceTimer: ReturnType<typeof setTimeout> | null = null
let settingsSaveTimer: ReturnType<typeof setTimeout> | null = null
const activeField = ref<'source' | 'target' | null>(null)
const isLoadingSource = ref(false)
const isLoadingTarget = ref(false)
const translationSource = ref('')
const translationSourceLabel = computed(() => {
    if (translationSource.value === 'user') return t.value.translationSourceUser
    if (translationSource.value === 'dictionary') return t.value.translationSourceDictionary
    if (translationSource.value === 'google') return t.value.translationSourceGoogle
    return translationSource.value
})

const focusTextarea = async (field: 'source' | 'target') => {
    await nextTick()

    window.setTimeout(() => {
        if (field === 'source') {
            sourceTextareaRef.value?.focus()
            activeField.value = 'source'
            return
        }

        targetTextareaRef.value?.focus()
        activeField.value = 'target'
    }, 0)
}

const persistTranslationLanguages = async () => {
    const user = authStore.user
    if (!user) {
        return
    }

    const nextTargetLanguage = getDistinctTargetLanguage(sourceLang.value, targetLang.value)
    if (nextTargetLanguage !== targetLang.value) {
        targetLang.value = nextTargetLanguage
        return
    }

    const currentSettings = user.settings
    if (
        currentSettings.translation_source_language === sourceLang.value &&
        currentSettings.translation_target_language === targetLang.value
    ) {
        return
    }

    try {
        authStore.user = await settingsApi.updateSettings({
            ...currentSettings,
            translation_source_language: sourceLang.value,
            translation_target_language: targetLang.value,
        })
    } catch (error) {
        console.error('Failed to save translation languages:', error)
        addToast({
            title: t.value.translationToastLangErrorTitle,
            description: t.value.translationToastLangErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    }
}

const queuePersistTranslationLanguages = () => {
    if (settingsSaveTimer) {
        clearTimeout(settingsSaveTimer)
    }

    settingsSaveTimer = setTimeout(() => {
        void persistTranslationLanguages()
    }, 300)
}

const performTranslation = async (
    fromText: string,
    fromLang: string,
    toLang: string,
    updateTarget: (text: string) => void,
    setLoading: (loading: boolean) => void
) => {
    if (!fromText.trim()) {
        updateTarget('')
        translationSource.value = ''
        translationId.value = null
        return
    }

    setLoading(true)
    try {
        const result = await translationApi.translate({
            from_word: fromText,
            from_language: fromLang,
            to_language: toLang,
        })
        updateTarget(result.translation)
        translationSource.value = result.source
        translationId.value = result.id
    } catch (error) {
        console.error('Translation error:', error)
        translationSource.value = ''
        translationId.value = null
    } finally {
        setLoading(false)
    }
}

const debouncedTranslate = (
    fromText: string,
    fromLang: string,
    toLang: string,
    updateTarget: (text: string) => void,
    setLoading: (loading: boolean) => void
) => {
    if (debounceTimer) {
        clearTimeout(debounceTimer)
    }

    debounceTimer = setTimeout(() => {
        performTranslation(fromText, fromLang, toLang, updateTarget, setLoading)
    }, 500)
}

const queueSourceToTargetTranslation = (fromText: string) => {
    debouncedTranslate(
        fromText,
        sourceLang.value,
        targetLang.value,
        (text) => {
            translatedText.value = text
        },
        (loading) => {
            isLoadingTarget.value = loading
        }
    )
}

const queueTargetToSourceTranslation = (fromText: string) => {
    debouncedTranslate(
        fromText,
        targetLang.value,
        sourceLang.value,
        (text) => {
            sourceText.value = text
        },
        (loading) => {
            isLoadingSource.value = loading
        }
    )
}

const triggerActiveFieldTranslation = (requireText: boolean = false) => {
    if (activeField.value === 'source') {
        if (requireText && !sourceText.value.trim()) {
            return
        }

        translationId.value = null
        queueSourceToTargetTranslation(sourceText.value)
        return
    }

    if (activeField.value === 'target') {
        if (requireText && !translatedText.value.trim()) {
            return
        }

        translationId.value = null
        queueTargetToSourceTranslation(translatedText.value)
    }
}

watch(
    sourceText,
    (newValue) => {
        if (activeField.value !== 'source') return
        translationId.value = null
        queueSourceToTargetTranslation(newValue)
    },
    { immediate: false }
)

watch(
    translatedText,
    (newValue) => {
        if (activeField.value !== 'target') return
        translationId.value = null
        queueTargetToSourceTranslation(newValue)
    },
    { immediate: false }
)

watch(
    sourceLang,
    () => {
        void focusTextarea('source')
        triggerActiveFieldTranslation(true)
        queuePersistTranslationLanguages()
    },
    { immediate: false }
)

watch(
    targetLang,
    () => {
        void focusTextarea('target')
        triggerActiveFieldTranslation(true)
        queuePersistTranslationLanguages()
    },
    { immediate: false }
)

const handleSwapLanguages = () => {
    ;[sourceLang.value, targetLang.value] = [targetLang.value, sourceLang.value]
    ;[sourceText.value, translatedText.value] = [translatedText.value, sourceText.value]
}

const saveTranslationToVocabulary = async () => {
    if (!translationId.value) {
        addToast({
            title: t.value.translationToastNoTranslationTitle,
            description: t.value.translationToastNoTranslationDescription,
            duration: 3000,
        })
        return
    }

    if (isSavingVocabulary.value) {
        return
    }

    isSavingVocabulary.value = true
    try {
        await translationApi.addVocabularyByTranslation(translationId.value)
        addToast({
            title: t.value.translationToastVocabSuccessTitle,
            description: t.value.translationToastVocabSuccessDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        const apiError = error as { status?: number }
        if (apiError.status === 409) {
            addToast({
                title: t.value.translationToastAlreadyExistsTitle,
                description: t.value.translationToastAlreadyExistsDescription,
                duration: 3000,
            })
            return
        }

        addToast({
            title: t.value.translationToastVocabErrorTitle,
            description: t.value.translationToastVocabErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSavingVocabulary.value = false
    }
}

const handleShortcut = (event: KeyboardEvent) => {
    if (event.ctrlKey && event.code === 'KeyL') {
        event.preventDefault()

        if (event.shiftKey) {
            void targetLanguageSelectorRef.value?.focusInput()
            return
        }

        void sourceLanguageSelectorRef.value?.focusInput()
        return
    }

    if (event.key === 'Tab' && !event.ctrlKey && !event.metaKey && !event.altKey) {
        event.preventDefault()

        const sourceElement = sourceTextareaRef.value
        const targetElement = targetTextareaRef.value
        const activeElement = document.activeElement

        if (event.shiftKey) {
            if (activeElement === targetElement) {
                sourceElement?.focus()
                activeField.value = 'source'
                return
            }

            targetElement?.focus()
            activeField.value = 'target'
            return
        }

        if (activeElement === sourceElement) {
            targetElement?.focus()
            activeField.value = 'target'
            return
        }

        sourceElement?.focus()
        activeField.value = 'source'
        return
    }

    if (!event.ctrlKey || event.code !== 'KeyS') {
        return
    }

    event.preventDefault()

    if (event.shiftKey) {
        handleSwapLanguages()
        return
    }

    void saveTranslationToVocabulary()
}

onMounted(() => {
    window.addEventListener('keydown', handleShortcut)
    void nextTick(() => {
        sourceTextareaRef.value?.focus()
        activeField.value = 'source'
    })
})

onBeforeUnmount(() => {
    window.removeEventListener('keydown', handleShortcut)
    if (debounceTimer) {
        clearTimeout(debounceTimer)
    }
    if (settingsSaveTimer) {
        clearTimeout(settingsSaveTimer)
    }
})
</script>

<template>
    <main class="px-6 py-8">
        <div class="max-w-5xl mx-auto">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label for="source-text" class="text-sm font-medium text-foreground">{{ t.translationFrom }}</label>
                        <div class="w-52">
                            <LanguageSelector
                                ref="sourceLanguageSelectorRef"
                                v-model="sourceLang"
                                :placeholder="t.translationFrom"
                                :disabled-values="[targetLang]"
                                aria-label="Source language"
                            />
                        </div>
                    </div>
                    <div class="relative">
                        <textarea
                            id="source-text"
                            ref="sourceTextareaRef"
                            v-model="sourceText"
                            @focus="activeField = 'source'"
                            :placeholder="t.translationFromPlaceholder"
                            class="w-full h-40 md:h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                        />
                        <div
                            v-if="isLoadingSource"
                            role="status"
                            :aria-label="t.translationTranslating"
                            class="absolute inset-0 flex items-center justify-center bg-background/50 rounded-lg"
                        >
                            <div class="motion-safe:animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                        </div>
                    </div>
                    <p class="text-xs text-muted-foreground text-right">{{ sourceText.length }} {{ t.translationCharacters }}</p>
                </div>

                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label for="target-text" class="text-sm font-medium text-foreground">{{ t.translationTo }}</label>
                        <div class="w-52">
                            <LanguageSelector
                                ref="targetLanguageSelectorRef"
                                v-model="targetLang"
                                :placeholder="t.translationTo"
                                :disabled-values="[sourceLang]"
                                aria-label="Target language"
                            />
                        </div>
                    </div>
                    <div class="relative">
                        <textarea
                            id="target-text"
                            ref="targetTextareaRef"
                            v-model="translatedText"
                            @focus="activeField = 'target'"
                            :placeholder="t.translationToPlaceholder"
                            class="w-full h-40 md:h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                        />
                        <div
                            v-if="isLoadingTarget"
                            role="status"
                            :aria-label="t.translationTranslating"
                            class="absolute inset-0 flex items-center justify-center bg-background/50 rounded-lg"
                        >
                            <div class="motion-safe:animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                        </div>
                    </div>
                    <p class="text-xs text-muted-foreground text-right">{{ translatedText.length }} {{ t.translationCharacters }}</p>
                </div>
            </div>

            <p v-if="translationSource" class="mt-3 text-center text-xs text-muted-foreground">
                {{ t.translationSourcePrefix }} {{ translationSourceLabel }}
            </p>
            <div v-if="isPhoneViewport" class="mt-4 flex justify-center">
                <Button @click="saveTranslationToVocabulary" :disabled="isSavingVocabulary || !translationId">
                    {{ isSavingVocabulary ? t.translationSaving : t.translationSaveToVocabulary }}
                </Button>
            </div>
            <div
                class="mt-4 hidden w-fit mx-auto grid-cols-[max-content_max-content] items-center gap-x-3 gap-y-2 text-xs text-muted-foreground md:grid"
            >
                <span class="justify-self-end text-right">{{ t.translationShortcutSave }}</span>
                <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + S</Kbd>
                <span class="justify-self-end text-right">{{ t.translationShortcutSwap }}</span>
                <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + Shift + S</Kbd>
                <span class="justify-self-end text-right">{{ t.translationShortcutFocusFirst }}</span>
                <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + L</Kbd>
                <span class="justify-self-end text-right">{{ t.translationShortcutFocusSecond }}</span>
                <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + Shift + L</Kbd>
            </div>
        </div>
    </main>
</template>
