<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowUpDown, Play } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
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
const router = useRouter()
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
let isSwappingLanguages = false
const activeField = ref<'source' | 'target' | null>(null)
const isLoadingSource = ref(false)
const isLoadingTarget = ref(false)
const translationSource = ref('')
const translationErrorMessage = ref('')
let latestTranslationRequestId = 0
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
        translationErrorMessage.value = ''
        return
    }

    const requestId = ++latestTranslationRequestId
    setLoading(true)
    translationErrorMessage.value = ''

    try {
        const result = await translationApi.translate({
            from_word: fromText,
            from_language: fromLang,
            to_language: toLang,
        })

        if (requestId !== latestTranslationRequestId) {
            return
        }

        updateTarget(result.translation)
        translationSource.value = result.source
        translationId.value = result.id
    } catch (error) {
        if (requestId !== latestTranslationRequestId) {
            return
        }

        console.error('Translation error:', error)
        translationSource.value = ''
        translationId.value = null
        translationErrorMessage.value = t.value.translationTranslateErrorDescription
    } finally {
        if (requestId === latestTranslationRequestId) {
            setLoading(false)
        }
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
        translationErrorMessage.value = ''
        queueSourceToTargetTranslation(newValue)
    },
    { immediate: false }
)

watch(
    translatedText,
    (newValue) => {
        if (activeField.value !== 'target') return
        translationId.value = null
        translationErrorMessage.value = ''
        queueTargetToSourceTranslation(newValue)
    },
    { immediate: false }
)

watch(
    sourceLang,
    () => {
        if (!isSwappingLanguages) {
            void focusTextarea('source')
        }
        triggerActiveFieldTranslation(true)
        queuePersistTranslationLanguages()
    },
    { immediate: false }
)

watch(
    targetLang,
    () => {
        if (!isSwappingLanguages) {
            void focusTextarea('target')
        }
        triggerActiveFieldTranslation(true)
        queuePersistTranslationLanguages()
    },
    { immediate: false }
)

const handleSwapLanguages = () => {
    const fieldToRefocus = activeField.value

    isSwappingLanguages = true
    latestTranslationRequestId += 1
    translationErrorMessage.value = ''
    isLoadingSource.value = false
    isLoadingTarget.value = false
    ;[sourceLang.value, targetLang.value] = [targetLang.value, sourceLang.value]
    ;[sourceText.value, translatedText.value] = [translatedText.value, sourceText.value]

    void nextTick(() => {
        isSwappingLanguages = false

        if (fieldToRefocus) {
            void focusTextarea(fieldToRefocus)
        }
    })
}

const handleTextareaTab = (field: 'source' | 'target', event: KeyboardEvent) => {
    event.preventDefault()
    void focusTextarea(field === 'source' ? 'target' : 'source')
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
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-6xl">
            <h1 class="sr-only">{{ t.navHome }}</h1>
            <div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] lg:gap-5 xl:gap-6">
                <div class="space-y-3">
                    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <label for="source-text" class="text-sm font-medium text-foreground">{{
                            t.translationFrom
                        }}</label>
                        <div class="w-full sm:w-52">
                            <LanguageSelector
                                ref="sourceLanguageSelectorRef"
                                v-model="sourceLang"
                                :placeholder="t.translationFrom"
                                :disabled-values="[targetLang]"
                                aria-label="Source language"
                                :empty-text="t.languageSelectorNoResults"
                            />
                        </div>
                    </div>
                    <div class="relative min-w-0">
                        <textarea
                            id="source-text"
                            ref="sourceTextareaRef"
                            v-model="sourceText"
                            @focus="activeField = 'source'"
                            @keydown.tab="handleTextareaTab('source', $event)"
                            :placeholder="t.translationFromPlaceholder"
                            maxlength="5000"
                            class="h-40 w-full resize-none rounded-lg border border-border bg-background p-4 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary sm:text-sm lg:h-72"
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
                    <p class="text-xs text-muted-foreground text-right">
                        {{ sourceText.length }} {{ t.translationCharacters }}
                    </p>
                </div>

                <div class="flex items-center justify-center lg:pt-14">
                    <Button
                        variant="outline"
                        size="icon"
                        class="h-11 w-11 rounded-full"
                        :aria-label="t.translationShortcutSwap"
                        @click="handleSwapLanguages"
                    >
                        <ArrowUpDown class="h-4 w-4 lg:rotate-90" />
                    </Button>
                </div>

                <div class="space-y-3">
                    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                        <label for="target-text" class="text-sm font-medium text-foreground">{{
                            t.translationTo
                        }}</label>
                        <div class="w-full sm:w-52">
                            <LanguageSelector
                                ref="targetLanguageSelectorRef"
                                v-model="targetLang"
                                :placeholder="t.translationTo"
                                :disabled-values="[sourceLang]"
                                aria-label="Target language"
                                :empty-text="t.languageSelectorNoResults"
                            />
                        </div>
                    </div>
                    <div class="relative min-w-0">
                        <textarea
                            id="target-text"
                            ref="targetTextareaRef"
                            v-model="translatedText"
                            @focus="activeField = 'target'"
                            @keydown.tab="handleTextareaTab('target', $event)"
                            :placeholder="t.translationToPlaceholder"
                            maxlength="5000"
                            class="h-40 w-full resize-none rounded-lg border border-border bg-background p-4 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary sm:text-sm lg:h-72"
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
                    <p class="text-xs text-muted-foreground text-right">
                        {{ translatedText.length }} {{ t.translationCharacters }}
                    </p>
                </div>
            </div>

            <div
                v-if="translationErrorMessage"
                class="mt-3 flex flex-col items-center justify-center gap-3 text-center sm:flex-row"
            >
                <p class="max-w-2xl text-sm text-destructive">{{ translationErrorMessage }}</p>
                <Button variant="outline" size="sm" @click="triggerActiveFieldTranslation(true)">{{
                    t.commonRetry
                }}</Button>
            </div>

            <p v-if="translationSource" class="mt-3 text-center text-xs text-muted-foreground">
                {{ t.translationSourcePrefix }} {{ translationSourceLabel }}
            </p>
            <div v-if="isPhoneViewport" class="mt-4 flex justify-center">
                <Button
                    class="min-h-11 w-full sm:w-auto"
                    @click="saveTranslationToVocabulary"
                    :disabled="isSavingVocabulary || !translationId"
                >
                    {{ isSavingVocabulary ? t.translationSaving : t.translationSaveToVocabulary }}
                </Button>
            </div>
            <div v-else class="mt-4 flex justify-center">
                <div
                    class="hidden w-fit grid-cols-[max-content_max-content] items-center gap-x-3 gap-y-2 text-xs text-muted-foreground md:grid"
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

            <Card class="mt-6 border-primary/20 bg-gradient-to-r from-primary/8 via-background to-background sm:mt-8">
                <CardContent class="p-3 sm:p-4">
                    <div
                        class="flex min-h-[220px] w-full max-w-full flex-col items-center justify-center rounded-2xl border border-primary/20 bg-background/90 px-5 py-6 text-center shadow-sm backdrop-blur-sm sm:min-h-[260px] lg:ml-auto lg:max-w-[320px]"
                    >
                        <div class="space-y-2">
                            <p class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
                                {{ t.quizCardTitle }}
                            </p>
                            <p class="mx-auto max-w-[24ch] text-sm leading-6 text-muted-foreground">
                                {{ t.quizCardDescription }}
                            </p>
                        </div>
                        <Button
                            size="lg"
                            class="mt-7 h-12 rounded-full border border-primary/20 bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[0_10px_30px_-12px_hsl(var(--primary)/0.8)] transition-transform duration-200 hover:-translate-y-0.5 hover:bg-primary/90"
                            @click="router.push({ name: 'quiz' })"
                        >
                            <Play class="size-4 fill-current" />
                            {{ t.quizRun }}
                        </Button>
                    </div>
                </CardContent>
            </Card>
        </div>
    </main>
</template>
