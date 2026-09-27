<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { adminApi, type Dictionary } from '@/api/admin'
import DictionarySource from '@/components/DictionarySource.vue'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'

const { t } = useI18n()
const dictionaries = ref<Dictionary[]>([])
const loading = ref(true)
const error = ref(false)
const load = async () => {
    loading.value = true
    error.value = false
    try {
        dictionaries.value = await adminApi.getDictionaries()
    } catch {
        error.value = true
    } finally {
        loading.value = false
    }
}
onMounted(load)
</script>

<template>
    <main class="px-4 py-6 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-7xl">
            <header class="mb-8">
                <h2 class="text-2xl font-semibold">{{ t.navDictionaries }}</h2>
                <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{{ t.dictionariesIntro }}</p>
                <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{{ t.dictionariesBackground }}</p>
            </header>
            <p v-if="loading" role="status" class="text-sm text-muted-foreground">{{ t.dictionaryLoading }}</p>
            <div v-else-if="error" role="alert" class="flex flex-wrap items-center gap-3 text-sm">
                <span class="text-foreground">{{ t.dictionaryLoadError }}</span>
                <Button variant="outline" @click="load">{{ t.dictionaryRefresh }}</Button>
            </div>
            <p v-else-if="!dictionaries.length" class="text-sm text-muted-foreground">{{ t.dictionariesEmpty }}</p>
            <div v-else class="divide-y divide-border">
                <DictionarySource v-for="dictionary in dictionaries" :key="dictionary.id" :dictionary="dictionary" />
            </div>
        </div>
    </main>
</template>
