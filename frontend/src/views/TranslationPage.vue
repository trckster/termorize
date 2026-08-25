<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ArrowUpDown, Loader2, Play } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { settingsApi } from '@/api/settings.ts'
import { translationApi } from '@/api/translation.ts'
import { vocabularyApi } from '@/api/vocabulary.ts'
import LanguageSelector from '@/components/LanguageSelector.vue'
import PronunciationButton from '@/components/PronunciationButton.vue'
import { Kbd } from '@/components/ui/kbd'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { useToast } from '@/composables/useToast.ts'
import { usePhoneViewport } from '@/composables/usePhoneViewport.ts'
import { useAuthStore } from '@/stores/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { useI18n } from '@/composables/useI18n'
import {
    createProgrammaticChangeGuard,
    getLanguageChangeDirection,
    hasMeaningfulVocabularyEdits,
    isEditableVocabularySaveShortcut,
    isEditableVocabularyShortcut,
    type EditableVocabularyPair,
    type TranslationField,
} from '@/lib/translationPageState.ts'

type LanguageSelectorInstance = {
    focusInput: () => Promise<void>
}

const authStore = useAuthStore()
const settingsStore = useSettingsStore()
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
const sourceWordId = ref<string | null>(null)
const targetWordId = ref<string | null>(null)
const isSavingVocabulary = ref(false)
const isEditVocabularyDialogOpen = ref(false)
const editableTranslationRef = ref<HTMLTextAreaElement | null>(null)
const editableVocabularyPair = ref<EditableVocabularyPair>({
    original: '',
    translation: '',
})
const editableVocabularySnapshot = ref<
    | (EditableVocabularyPair & {
          translationId: string
          originalLanguage: string
          translationLanguage: string
      })
    | null
>(null)
const programmaticTextChanges = createProgrammaticChangeGuard<'source' | 'target'>()

const { addToast } = useToast()

let debounceTimer: ReturnType<typeof setTimeout> | null = null
let settingsSaveTimer: ReturnType<typeof setTimeout> | null = null
let isSwappingLanguages = false
let lastTranslationDirection: 'source-to-target' | 'target-to-source' | null = null
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
const isEditableVocabularyPairValid = computed(
    () =>
        editableVocabularyPair.value.original.trim().length > 0 &&
        editableVocabularyPair.value.translation.trim().length > 0
)

const getLanguageLabel = (language: string) =>
    settingsStore.getLanguageName(language, authStore.user?.settings.system_language ?? 'en')

const invalidateTranslationResult = () => {
    latestTranslationRequestId += 1
    translationId.value = null
    sourceWordId.value = null
    targetWordId.value = null
    translationSource.value = ''
    isLoadingSource.value = false
    isLoadingTarget.value = false
}

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
    direction: 'source-to-target' | 'target-to-source',
    updateTarget: (text: string) => void,
    setLoading: (loading: boolean) => void
) => {
    if (!fromText.trim()) {
        updateTarget('')
        translationSource.value = ''
        translationId.value = null
        sourceWordId.value = null
        targetWordId.value = null
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
        if (direction === 'source-to-target') {
            sourceWordId.value = result.original_word_id
            targetWordId.value = result.translation_word_id
        } else {
            sourceWordId.value = result.translation_word_id
            targetWordId.value = result.original_word_id
        }
    } catch (error) {
        if (requestId !== latestTranslationRequestId) {
            return
        }

        console.error('Translation error:', error)
        translationSource.value = ''
        translationId.value = null
        sourceWordId.value = null
        targetWordId.value = null
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
    direction: 'source-to-target' | 'target-to-source',
    updateTarget: (text: string) => void,
    setLoading: (loading: boolean) => void
) => {
    if (debounceTimer) {
        clearTimeout(debounceTimer)
    }

    lastTranslationDirection = direction
    debounceTimer = setTimeout(() => {
        performTranslation(fromText, fromLang, toLang, direction, updateTarget, setLoading)
    }, 500)
}

const queueSourceToTargetTranslation = (fromText: string) => {
    debouncedTranslate(
        fromText,
        sourceLang.value,
        targetLang.value,
        'source-to-target',
        (text) => {
            if (translatedText.value === text) return

            programmaticTextChanges.mark('target', text)
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
        'target-to-source',
        (text) => {
            if (sourceText.value === text) return

            programmaticTextChanges.mark('source', text)
            sourceText.value = text
        },
        (loading) => {
            isLoadingSource.value = loading
        }
    )
}

const retryLastTranslation = () => {
    if (lastTranslationDirection === 'source-to-target' && sourceText.value.trim()) {
        invalidateTranslationResult()
        queueSourceToTargetTranslation(sourceText.value)
        return
    }

    if (lastTranslationDirection === 'target-to-source' && translatedText.value.trim()) {
        invalidateTranslationResult()
        queueTargetToSourceTranslation(translatedText.value)
    }
}

const translateForLanguageChange = (field: TranslationField) => {
    const direction = getLanguageChangeDirection(field)

    if (direction === 'source-to-target' && sourceText.value.trim()) {
        invalidateTranslationResult()
        queueSourceToTargetTranslation(sourceText.value)
        return
    }

    if (direction === 'target-to-source' && translatedText.value.trim()) {
        invalidateTranslationResult()
        queueTargetToSourceTranslation(translatedText.value)
    }
}

watch(
    sourceText,
    (newValue) => {
        if (programmaticTextChanges.consume('source', newValue)) return
        if (activeField.value !== 'source') return
        invalidateTranslationResult()
        translationErrorMessage.value = ''
        queueSourceToTargetTranslation(newValue)
    },
    { immediate: false }
)

watch(
    translatedText,
    (newValue) => {
        if (programmaticTextChanges.consume('target', newValue)) return
        if (activeField.value !== 'target') return
        invalidateTranslationResult()
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
        translateForLanguageChange('source')
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
        translateForLanguageChange('target')
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
    ;[sourceWordId.value, targetWordId.value] = [targetWordId.value, sourceWordId.value]

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

const showNoTranslationToast = () => {
    addToast({
        title: t.value.translationToastNoTranslationTitle,
        description: t.value.translationToastNoTranslationDescription,
        duration: 3000,
    })
}

const showVocabularySavedToast = () => {
    addToast({
        title: t.value.translationToastVocabSuccessTitle,
        description: t.value.translationToastVocabSuccessDescription,
        variant: 'success',
        duration: 3000,
    })
}

const showVocabularySaveError = (error: unknown) => {
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
}

const openEditableVocabularyDialog = () => {
    if (!translationId.value || !sourceText.value.trim() || !translatedText.value.trim()) {
        showNoTranslationToast()
        return
    }

    if (isSavingVocabulary.value) {
        return
    }

    editableVocabularySnapshot.value = {
        translationId: translationId.value,
        original: sourceText.value,
        translation: translatedText.value,
        originalLanguage: sourceLang.value,
        translationLanguage: targetLang.value,
    }
    editableVocabularyPair.value = {
        original: sourceText.value,
        translation: translatedText.value,
    }
    isEditVocabularyDialogOpen.value = true
}

const handleEditVocabularyDialogOpenChange = (isOpen: boolean) => {
    if (!isOpen && isSavingVocabulary.value) {
        return
    }

    isEditVocabularyDialogOpen.value = isOpen
    if (!isOpen) {
        editableVocabularySnapshot.value = null
    }
}

const focusEditableTranslation = () => {
    const textarea = editableTranslationRef.value
    if (!textarea) return

    textarea.focus()
    textarea.setSelectionRange(textarea.value.length, textarea.value.length)
}

const handleEditableVocabularyKeydown = (event: KeyboardEvent) => {
    if (!isEditableVocabularySaveShortcut(event)) {
        return
    }

    event.preventDefault()
    void saveEditableTranslationToVocabulary()
}

const saveEditableTranslationToVocabulary = async () => {
    const snapshot = editableVocabularySnapshot.value
    if (!snapshot || !isEditableVocabularyPairValid.value || isSavingVocabulary.value) {
        return
    }

    const editedPair = {
        original: editableVocabularyPair.value.original.trim(),
        translation: editableVocabularyPair.value.translation.trim(),
    }

    isSavingVocabulary.value = true
    try {
        if (hasMeaningfulVocabularyEdits(snapshot, editedPair)) {
            await vocabularyApi.addVocabulary(
                editedPair.original,
                editedPair.translation,
                snapshot.originalLanguage,
                snapshot.translationLanguage
            )
        } else {
            await translationApi.addVocabularyByTranslation(snapshot.translationId)
        }

        isEditVocabularyDialogOpen.value = false
        editableVocabularySnapshot.value = null
        showVocabularySavedToast()
    } catch (error) {
        showVocabularySaveError(error)
    } finally {
        isSavingVocabulary.value = false
    }
}

const saveTranslationToVocabulary = async () => {
    if (!translationId.value) {
        showNoTranslationToast()
        return
    }

    if (isSavingVocabulary.value) {
        return
    }

    isSavingVocabulary.value = true
    try {
        await translationApi.addVocabularyByTranslation(translationId.value)
        showVocabularySavedToast()
    } catch (error) {
        showVocabularySaveError(error)
    } finally {
        isSavingVocabulary.value = false
    }
}

const handleShortcut = (event: KeyboardEvent) => {
    if (isEditableVocabularyShortcut(event)) {
        event.preventDefault()

        if (!isEditVocabularyDialogOpen.value) {
            openEditableVocabularyDialog()
        }
        return
    }

    if (isEditVocabularyDialogOpen.value) {
        if (event.ctrlKey && (event.code === 'KeyL' || event.code === 'KeyS')) {
            event.preventDefault()
        }
        return
    }

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
    if (!isPhoneViewport.value) {
        void nextTick(() => {
            sourceTextareaRef.value?.focus()
            activeField.value = 'source'
        })
    }
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

            <Dialog :open="isEditVocabularyDialogOpen" @update:open="handleEditVocabularyDialogOpenChange">
                <DialogContent
                    class="sm:max-w-xl"
                    :hide-close="isSavingVocabulary"
                    @open-auto-focus.prevent="focusEditableTranslation"
                >
                    <DialogHeader>
                        <DialogTitle>{{ t.translationEditDialogTitle }}</DialogTitle>
                        <DialogDescription>
                            {{ t.translationEditDialogDescription }}
                        </DialogDescription>
                    </DialogHeader>

                    <form
                        class="space-y-5 pt-2"
                        :aria-busy="isSavingVocabulary"
                        @submit.prevent="saveEditableTranslationToVocabulary"
                        @keydown.esc.stop.prevent="handleEditVocabularyDialogOpenChange(false)"
                    >
                        <div class="space-y-2">
                            <div class="flex flex-wrap items-center justify-between gap-2">
                                <label for="editable-vocabulary-original" class="text-sm font-medium text-foreground">
                                    {{ t.translationEditOriginalLabel }}
                                </label>
                                <span
                                    v-if="editableVocabularySnapshot"
                                    id="editable-vocabulary-original-language"
                                    class="rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-muted-foreground"
                                >
                                    {{ getLanguageLabel(editableVocabularySnapshot.originalLanguage) }}
                                </span>
                            </div>
                            <textarea
                                id="editable-vocabulary-original"
                                v-model="editableVocabularyPair.original"
                                maxlength="5000"
                                rows="3"
                                aria-describedby="editable-vocabulary-original-language"
                                class="min-h-24 w-full resize-y rounded-lg border border-border bg-background px-3 py-2.5 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary sm:text-sm"
                                @keydown="handleEditableVocabularyKeydown"
                            />
                        </div>

                        <div class="space-y-2">
                            <div class="flex flex-wrap items-center justify-between gap-2">
                                <label
                                    for="editable-vocabulary-translation"
                                    class="text-sm font-medium text-foreground"
                                >
                                    {{ t.translationEditTranslationLabel }}
                                </label>
                                <span
                                    v-if="editableVocabularySnapshot"
                                    id="editable-vocabulary-translation-language"
                                    class="rounded-full bg-muted px-2.5 py-1 text-xs font-medium text-muted-foreground"
                                >
                                    {{ getLanguageLabel(editableVocabularySnapshot.translationLanguage) }}
                                </span>
                            </div>
                            <textarea
                                id="editable-vocabulary-translation"
                                ref="editableTranslationRef"
                                v-model="editableVocabularyPair.translation"
                                maxlength="5000"
                                rows="3"
                                aria-describedby="editable-vocabulary-translation-language"
                                class="min-h-24 w-full resize-y rounded-lg border border-border bg-background px-3 py-2.5 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary sm:text-sm"
                                @keydown="handleEditableVocabularyKeydown"
                            />
                        </div>

                        <p class="text-xs leading-5 text-muted-foreground">
                            {{ t.translationEditKeyboardHint }}
                        </p>

                        <DialogFooter>
                            <DialogClose as-child>
                                <Button type="button" variant="outline" :disabled="isSavingVocabulary">
                                    {{ t.cancel }}
                                </Button>
                            </DialogClose>
                            <Button type="submit" :disabled="isSavingVocabulary || !isEditableVocabularyPairValid">
                                <Loader2 v-if="isSavingVocabulary" class="motion-safe:animate-spin" />
                                {{ isSavingVocabulary ? t.translationSaving : t.translationSaveToVocabulary }}
                            </Button>
                        </DialogFooter>
                    </form>
                </DialogContent>
            </Dialog>

            <div
                class="grid grid-cols-1 gap-1 sm:gap-4 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] lg:gap-5 xl:gap-6"
            >
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
                            class="h-32 w-full resize-none rounded-lg border border-border bg-background p-4 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary min-[360px]:h-40 sm:text-sm lg:h-72"
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
                    <div class="flex min-h-11 items-center justify-between gap-3">
                        <PronunciationButton
                            :word-id="sourceWordId"
                            :word="sourceText"
                            :listen-label="t.pronunciationListen"
                            :pause-label="t.pronunciationPause"
                            :loading-label="t.pronunciationLoading"
                            :error-label="t.pronunciationError"
                        />
                        <p class="text-right text-xs text-muted-foreground">
                            {{ sourceText.length }} {{ t.translationCharacters }}
                        </p>
                    </div>
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
                            class="h-32 w-full resize-none rounded-lg border border-border bg-background p-4 text-base text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary min-[360px]:h-40 sm:text-sm lg:h-72"
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
                    <div class="flex min-h-11 items-center justify-between gap-3">
                        <PronunciationButton
                            :word-id="targetWordId"
                            :word="translatedText"
                            :listen-label="t.pronunciationListen"
                            :pause-label="t.pronunciationPause"
                            :loading-label="t.pronunciationLoading"
                            :error-label="t.pronunciationError"
                        />
                        <p class="text-right text-xs text-muted-foreground">
                            {{ translatedText.length }} {{ t.translationCharacters }}
                        </p>
                    </div>
                </div>
            </div>

            <div
                v-if="translationErrorMessage"
                class="mt-3 flex flex-col items-center justify-center gap-3 text-center sm:flex-row"
            >
                <p class="max-w-2xl text-sm text-destructive">{{ translationErrorMessage }}</p>
                <Button variant="outline" size="sm" @click="retryLastTranslation">{{ t.commonRetry }}</Button>
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
                    <span class="justify-self-end text-right">{{ t.translationShortcutEditBeforeSave }}</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + E</Kbd>
                    <span class="justify-self-end text-right">{{ t.translationShortcutSwap }}</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + Shift + S</Kbd>
                    <span class="justify-self-end text-right">{{ t.translationShortcutFocusFirst }}</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + L</Kbd>
                    <span class="justify-self-end text-right">{{ t.translationShortcutFocusSecond }}</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + Shift + L</Kbd>
                </div>
            </div>

            <section class="mt-8 border-t border-border py-8 text-center sm:mt-10 sm:py-10">
                <div class="mx-auto flex max-w-sm flex-col items-center">
                    <div class="space-y-2">
                        <h2 class="text-lg font-semibold tracking-tight text-foreground sm:text-xl">
                            {{ t.quizCardTitle }}
                        </h2>
                        <p class="mx-auto max-w-[30ch] text-sm leading-6 text-muted-foreground">
                            {{ t.quizCardDescription }}
                        </p>
                    </div>
                    <Button size="lg" class="mt-6 w-full sm:w-auto" @click="router.push({ name: 'quiz' })">
                        <Play class="size-4 fill-current" />
                        {{ t.quizRun }}
                    </Button>
                </div>
            </section>
        </div>
    </main>
</template>
