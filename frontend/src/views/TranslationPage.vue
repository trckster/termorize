<script setup lang="ts">
import { ref } from 'vue'
import Header from '@/components/Header.vue'
import { useSettingsStore } from '@/stores/settings.ts'

const sourceText = ref('')
const translatedText = ref('')
const sourceLang = ref('en')
const targetLang = ref('ru')

const settingsStore = useSettingsStore()

const handleSwapLanguages = () => {
    ;[sourceLang.value, targetLang.value] = [targetLang.value, sourceLang.value]
    ;[sourceText.value, translatedText.value] = [translatedText.value, sourceText.value]
}
</script>

<template>
    <Header />
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
                            <option v-for="lang in settingsStore.languageOptions" :key="lang.code" :value="lang.code">
                                {{ lang.name }}
                            </option>
                        </select>
                    </div>
                    <textarea
                        v-model="sourceText"
                        placeholder="Enter text to translate..."
                        class="w-full h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                    />
                    <p class="text-xs text-muted-foreground text-right">{{ sourceText.length }} characters</p>
                </div>

                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <label class="text-sm font-medium text-foreground">To</label>
                        <select
                            v-model="targetLang"
                            class="px-3 py-1.5 text-sm rounded-md border border-border bg-background text-foreground hover:bg-accent focus:outline-none focus:ring-2 focus:ring-primary"
                        >
                            <option v-for="lang in settingsStore.languageOptions" :key="lang.code" :value="lang.code">
                                {{ lang.name }}
                            </option>
                        </select>
                    </div>
                    <textarea
                        v-model="translatedText"
                        placeholder="Translation will appear here..."
                        class="w-full h-64 p-4 rounded-lg border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary resize-none"
                    />
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
