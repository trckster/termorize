<script setup lang="ts">
import { ref, watch } from 'vue'
import { useSettingsStore } from '@/stores/settings.ts'
import { translationApi } from '@/api/translation.ts'

const sourceText = ref('')
const translatedText = ref('')
const sourceLang = ref('en')
const targetLang = ref('ru')

const settingsStore = useSettingsStore()

let debounceTimer: ReturnType<typeof setTimeout> | null = null
const activeField = ref<'source' | 'target' | null>(null)
const isLoadingSource = ref(false)
const isLoadingTarget = ref(false)

const performTranslation = async (
    fromText: string,
    fromLang: string,
    toLang: string,
    updateTarget: (text: string) => void,
    setLoading: (loading: boolean) => void
) => {
    if (!fromText.trim()) {
        updateTarget('')
        return
    }

    setLoading(true)
    try {
        const result = await translationApi.translate({
            from_word: fromText,
            from_language: fromLang,
            to_language: toLang,
        })
        updateTarget(result)
    } catch (error) {
        console.error('Translation error:', error)
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
</script>

<template>
    <main class="px-6 py-8">
        <div class="max-w-5xl mx-auto">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-sm font-medium text-foreground">From</label>
                        <select
                            v-model="sourceLang"
                            class="px-3 py-1.5 text-sm rounded-md border border-border bg-background text-foreground hover:bg-accent focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <option
                                v-for="lang in settingsStore.languageOptions"
                                :key="lang.code"
                                :value="lang.code"
                                :disabled="lang.code === targetLang"
                            >
                                {{ lang.name }}
                            </option>
                        </select>
                    </div>
                    <div class="relative">
                        <textarea
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
                        <select
                            v-model="targetLang"
                            class="px-3 py-1.5 text-sm rounded-md border border-border bg-background text-foreground hover:bg-accent focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <option
                                v-for="lang in settingsStore.languageOptions"
                                :key="lang.code"
                                :value="lang.code"
                                :disabled="lang.code === sourceLang"
                            >
                                {{ lang.name }}
                            </option>
                        </select>
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

            <div class="flex justify-center mt-6">
                <button
                    @click="handleSwapLanguages"
                    class="px-4 py-2 rounded-lg border border-border bg-background text-foreground hover:bg-accent transition-colors font-medium"
                >
                    ⇄ Swap Languages
                </button>
            </div>
        </div>
    </main>
</template>
