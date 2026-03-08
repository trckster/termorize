<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { translationApi } from '@/api/translation.ts'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Kbd } from '@/components/ui/kbd'
import { useToast } from '@/composables/useToast.ts'

const sourceText = ref('')
const translatedText = ref('')
const sourceTextareaRef = ref<HTMLTextAreaElement | null>(null)
const sourceLang = ref('en')
const targetLang = ref('ru')
const translationId = ref<string | null>(null)
const isSavingVocabulary = ref(false)

const { addToast } = useToast()

let debounceTimer: ReturnType<typeof setTimeout> | null = null
const activeField = ref<'source' | 'target' | null>(null)
const isLoadingSource = ref(false)
const isLoadingTarget = ref(false)
const translationSource = ref('')
const translationSourceLabel = computed(() => {
    if (translationSource.value === 'user') return 'User'
    if (translationSource.value === 'dictionary') return 'Dictionary'
    if (translationSource.value === 'google') return 'Google'
    return translationSource.value
})

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

watch(
    sourceText,
    (newValue) => {
        if (activeField.value !== 'source') return
        translationId.value = null
        debouncedTranslate(
            newValue,
            sourceLang.value,
            targetLang.value,
            (text) => {
                translatedText.value = text
            },
            (loading) => {
                isLoadingTarget.value = loading
            }
        )
    },
    { immediate: false }
)

watch(
    translatedText,
    (newValue) => {
        if (activeField.value !== 'target') return
        translationId.value = null
        debouncedTranslate(
            newValue,
            targetLang.value,
            sourceLang.value,
            (text) => {
                sourceText.value = text
            },
            (loading) => {
                isLoadingSource.value = loading
            }
        )
    },
    { immediate: false }
)

watch(
    sourceLang,
    () => {
        if (activeField.value === 'source' && sourceText.value.trim()) {
            translationId.value = null
            debouncedTranslate(
                sourceText.value,
                sourceLang.value,
                targetLang.value,
                (text) => {
                    translatedText.value = text
                },
                (loading) => {
                    isLoadingTarget.value = loading
                }
            )
        } else if (activeField.value === 'target' && translatedText.value.trim()) {
            translationId.value = null
            debouncedTranslate(
                translatedText.value,
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
    },
    { immediate: false }
)

watch(
    targetLang,
    () => {
        if (activeField.value === 'source' && sourceText.value.trim()) {
            translationId.value = null
            debouncedTranslate(
                sourceText.value,
                sourceLang.value,
                targetLang.value,
                (text) => {
                    translatedText.value = text
                },
                (loading) => {
                    isLoadingTarget.value = loading
                }
            )
        } else if (activeField.value === 'target' && translatedText.value.trim()) {
            translationId.value = null
            debouncedTranslate(
                translatedText.value,
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
            title: 'Warning',
            description: 'No translation is available yet. Translate text first.',
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
            title: 'Success!',
            description: 'Translation added to vocabulary.',
            variant: 'success',
            duration: 3000,
        })
    } catch (error) {
        const apiError = error as { status?: number }
        if (apiError.status === 409) {
            addToast({
                title: 'Warning',
                description: 'This vocabulary already exists.',
                duration: 3000,
            })
            return
        }

        addToast({
            title: 'Error',
            description: 'Failed to add translation to vocabulary. Please try again.',
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSavingVocabulary.value = false
    }
}

const handleShortcut = (event: KeyboardEvent) => {
    if (!event.ctrlKey || event.key.toLowerCase() !== 's') {
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
})
</script>

<template>
    <main class="px-6 py-8">
        <div class="max-w-5xl mx-auto">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-sm font-medium text-foreground">From</label>
                        <div class="w-52">
                            <LanguageSelector
                                v-model="sourceLang"
                                placeholder="From language"
                                :disabled-values="[targetLang]"
                            />
                        </div>
                    </div>
                    <div class="relative">
                        <textarea
                            ref="sourceTextareaRef"
                            v-model="sourceText"
                            @focus="activeField = 'source'"
                            placeholder="Enter text to translate..."
                            class="w-full h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                        />
                        <div
                            v-if="isLoadingSource"
                            class="absolute inset-0 flex items-center justify-center bg-background/50 rounded-lg"
                        >
                            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                        </div>
                    </div>
                    <p class="text-xs text-muted-foreground text-right">{{ sourceText.length }} characters</p>
                </div>

                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-sm font-medium text-foreground">To</label>
                        <div class="w-52">
                            <LanguageSelector
                                v-model="targetLang"
                                placeholder="To language"
                                :disabled-values="[sourceLang]"
                            />
                        </div>
                    </div>
                    <div class="relative">
                        <textarea
                            v-model="translatedText"
                            @focus="activeField = 'target'"
                            placeholder="Translation will appear here..."
                            class="w-full h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                        />
                        <div
                            v-if="isLoadingTarget"
                            class="absolute inset-0 flex items-center justify-center bg-background/50 rounded-lg"
                        >
                            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                        </div>
                    </div>
                    <p class="text-xs text-muted-foreground text-right">{{ translatedText.length }} characters</p>
                </div>
            </div>

            <p v-if="translationSource" class="mt-3 text-center text-xs text-muted-foreground">
                Source: {{ translationSourceLabel }}
            </p>
            <div class="mt-4 flex flex-col items-center gap-2 text-xs text-muted-foreground">
                <div class="flex items-center gap-2">
                    <span>Swap languages</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + Shift + S</Kbd>
                </div>
                <div class="flex items-center gap-2">
                    <span>Save to vocabulary</span>
                    <Kbd class="min-h-5 px-1.5 py-0.5 text-[10px]">Ctrl + S</Kbd>
                </div>
            </div>
        </div>
    </main>
</template>
