<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { adminApi, type IdiomLanguageCoverage } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'
import { formatNumber } from '@/lib/utils'

const { t } = useI18n()
const languages = ref<IdiomLanguageCoverage[]>([])
const loading = ref(true)
const failed = ref(false)
let disposed = false
const load = async () => {
    loading.value = true
    failed.value = false
    try {
        const result = await adminApi.getIdiomCoverage()
        if (!disposed) languages.value = result
    } catch {
        if (!disposed) failed.value = true
    } finally {
        if (!disposed) loading.value = false
    }
}
onMounted(load)
onBeforeUnmount(() => {
    disposed = true
})
</script>

<template>
    <section aria-labelledby="idiom-coverage-title" class="mb-10">
        <div class="flex flex-wrap items-center justify-between gap-3">
            <h3 id="idiom-coverage-title" class="text-lg font-semibold">{{ t.idiomCoverageTitle }}</h3>
            <Button variant="outline" :disabled="loading" @click="load">{{ t.idiomCoverageRefresh }}</Button>
        </div>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{{ t.idiomCoverageIntro }}</p>
        <p v-if="loading" role="status" class="mt-4 text-sm text-muted-foreground">{{ t.idiomCoverageLoading }}</p>
        <p v-else-if="failed" role="alert" class="mt-4 text-sm text-destructive">{{ t.idiomCoverageError }}</p>
        <table v-else class="mt-4 w-full text-sm" aria-labelledby="idiom-coverage-title">
            <thead class="border-b border-border text-muted-foreground">
                <tr>
                    <th scope="col" class="py-3 pr-4 text-left font-medium">{{ t.idiomCoverageLanguage }}</th>
                    <th scope="col" class="py-3 text-right font-medium">{{ t.idiomCoverageCount }}</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-border">
                <tr v-for="entry in languages" :key="entry.language">
                    <th scope="row" class="py-3 pr-4 text-left font-medium">{{ entry.name }}</th>
                    <td class="py-3 text-right tabular-nums">
                        <template v-if="entry.idiom_count">{{ formatNumber(entry.idiom_count) }}</template>
                        <span v-else class="text-muted-foreground">0 · {{ t.idiomCoverageEmpty }}</span>
                    </td>
                </tr>
            </tbody>
        </table>
    </section>
</template>
