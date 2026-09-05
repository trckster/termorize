<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { LoaderCircle, RotateCcw, Search } from 'lucide-vue-next'
import { adminApi, type AdminWordDescription, type DescriptionModel, type DescriptionPreview } from '@/api/admin'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { useSettingsStore } from '@/stores/settings'
import { formatDate, formatNumber } from '@/lib/utils'

const { t } = useI18n()
const { addToast } = useToast()
const settings = useSettingsStore()
const descriptions = ref<AdminWordDescription[]>([])
const models = ref<DescriptionModel[]>([])
const selectedModel = ref('google/gemini-2.5-flash')
const page = ref(1)
const total = ref(0)
const totalPages = ref(0)
const searchInput = ref('')
const search = ref('')
const loading = ref(true)
const loadError = ref(false)
const generatingId = ref<string | null>(null)
const preview = ref<DescriptionPreview | null>(null)
const original = ref<AdminWordDescription | null>(null)
const dialogOpen = ref(false)
const approving = ref(false)
const approvalError = ref(false)
let debounce: ReturnType<typeof setTimeout> | undefined
let requestVersion = 0
let disposed = false
const tiers = computed(() => ({
    basic: t.value.descriptionsBasic,
    medium: t.value.descriptionsMedium,
    smart: t.value.descriptionsSmart,
}))
const languageLabel = (code: string) => {
    const language = settings.languageOptions.find((item) => item.code === code)
    return language ? `${language.emoji} ${language.name}` : code.toUpperCase()
}
const fetchDescriptions = async (nextPage = page.value) => {
    const version = ++requestVersion
    loading.value = true
    loadError.value = false
    try {
        const [response, options] = await Promise.all([
            adminApi.getWordDescriptions(nextPage, search.value || undefined),
            models.value.length ? Promise.resolve(models.value) : adminApi.getDescriptionModels(),
        ])
        if (disposed || version !== requestVersion) return
        models.value = options
        const lastPage = Math.max(1, response.pagination.total_pages)
        if (nextPage > lastPage) {
            await fetchDescriptions(lastPage)
            return
        }
        descriptions.value = response.data
        page.value = response.pagination.page
        total.value = response.pagination.total
        totalPages.value = response.pagination.total_pages
    } catch {
        if (!disposed && version === requestVersion) loadError.value = true
    } finally {
        if (!disposed && version === requestVersion) loading.value = false
    }
}
const regenerate = async (description: AdminWordDescription) => {
    generatingId.value = description.id
    try {
        const result = await adminApi.previewWordDescription(description.id, selectedModel.value)
        if (disposed) return
        preview.value = result
        original.value = description
        approvalError.value = false
        dialogOpen.value = true
    } catch {
        if (!disposed)
            addToast({
                title: t.value.toastErrorTitle,
                description: t.value.descriptionsGenerateError,
                variant: 'destructive',
            })
    } finally {
        generatingId.value = null
    }
}
const discard = () => {
    dialogOpen.value = false
    preview.value = null
}
const approve = async () => {
    if (!original.value || !preview.value || approving.value) return
    approving.value = true
    approvalError.value = false
    try {
        await adminApi.approveWordDescription(original.value.id, preview.value.id)
        if (disposed) return
        dialogOpen.value = false
        preview.value = null
        addToast({ title: t.value.descriptionsApproved, variant: 'success' })
        await fetchDescriptions()
    } catch {
        if (!disposed) approvalError.value = true
    } finally {
        approving.value = false
    }
}
watch(searchInput, (value) => {
    clearTimeout(debounce)
    debounce = setTimeout(() => {
        search.value = value.trim()
        void fetchDescriptions(1)
    }, 350)
})
onMounted(() => void fetchDescriptions())
onBeforeUnmount(() => {
    disposed = true
    clearTimeout(debounce)
})
</script>

<template>
    <main class="px-4 py-6 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-7xl space-y-6">
            <header>
                <h2 class="text-2xl font-semibold">
                    {{ t.navDescriptions }} <span class="text-muted-foreground">({{ formatNumber(total) }})</span>
                </h2>
                <p class="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{{ t.descriptionsIntro }}</p>
            </header>
            <div class="flex flex-col gap-4 sm:flex-row sm:items-end">
                <div class="relative flex-1">
                    <Search
                        class="pointer-events-none absolute left-3 top-3 size-5 text-muted-foreground"
                        aria-hidden="true"
                    />
                    <input
                        v-model="searchInput"
                        type="search"
                        :aria-label="t.descriptionsSearch"
                        :placeholder="t.descriptionsSearch"
                        class="h-11 w-full rounded-md border border-input bg-background pl-10 pr-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    />
                </div>
                <div>
                    <label for="description-model" class="mb-1 block text-sm font-medium">{{
                        t.descriptionsModel
                    }}</label>
                    <select
                        id="description-model"
                        v-model="selectedModel"
                        :disabled="!!generatingId || loading || !models.length"
                        class="h-11 w-full rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                        <option v-for="model in models" :key="model.id" :value="model.id">
                            {{ tiers[model.tier] }} · {{ model.name }}
                        </option>
                    </select>
                </div>
            </div>
            <div v-if="loadError" role="alert" class="rounded-lg border border-destructive p-4 text-destructive">
                {{ t.descriptionsLoadError }}
                <Button variant="outline" @click="fetchDescriptions()">{{ t.commonRetry }}</Button>
            </div>
            <div v-else-if="loading" class="space-y-3" aria-busy="true">
                <div
                    v-for="n in 4"
                    :key="n"
                    class="h-28 animate-pulse rounded-lg bg-muted motion-reduce:animate-none"
                />
            </div>
            <p
                v-else-if="!descriptions.length"
                class="rounded-lg border border-border p-8 text-center text-muted-foreground"
            >
                {{ t.descriptionsEmpty }}
            </p>
            <div v-else class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card">
                <article v-for="description in descriptions" :key="description.id" class="space-y-3 p-4 sm:p-5">
                    <div class="flex flex-wrap items-center justify-between gap-3">
                        <div class="min-w-0">
                            <h3 class="break-words font-semibold">{{ description.word }}</h3>
                            <p class="text-sm text-muted-foreground">{{ languageLabel(description.language) }}</p>
                        </div>
                        <Button
                            variant="outline"
                            :disabled="!!generatingId || !models.length"
                            @click="regenerate(description)"
                        >
                            <LoaderCircle v-if="generatingId === description.id" class="mr-2 size-4 animate-spin" />
                            <RotateCcw v-else class="mr-2 size-4" />
                            {{ t.descriptionsRegenerate }}
                        </Button>
                    </div>
                    <p class="break-words leading-7">{{ description.description }}</p>
                    <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                        <span class="break-all">{{ description.model }}</span>
                        <span>{{ t.descriptionsCreated }}: {{ formatDate(description.created_at) }}</span>
                    </div>
                </article>
            </div>
            <nav
                v-if="totalPages > 1"
                class="flex flex-wrap items-center justify-center gap-3"
                :aria-label="t.navDescriptions"
            >
                <Button variant="outline" :disabled="loading || page <= 1" @click="fetchDescriptions(page - 1)">{{
                    t.descriptionsPrevious
                }}</Button>
                <span class="text-sm text-muted-foreground">{{
                    t.descriptionsPage.replace('{page}', String(page)).replace('{total}', String(totalPages))
                }}</span>
                <Button
                    variant="outline"
                    :disabled="loading || page >= totalPages"
                    @click="fetchDescriptions(page + 1)"
                    >{{ t.descriptionsNext }}</Button
                >
            </nav>
        </div>
        <Dialog
            :open="dialogOpen"
            @update:open="
                (value) => {
                    if (!approving) dialogOpen = value
                }
            "
        >
            <DialogContent
                class="max-h-[90dvh] overflow-y-auto sm:max-w-xl"
                @escape-key-down="
                    (event) => {
                        if (approving) event.preventDefault()
                    }
                "
                @interact-outside="
                    (event) => {
                        if (approving) event.preventDefault()
                    }
                "
            >
                <DialogHeader>
                    <DialogTitle>{{ t.descriptionsPreviewTitle }}: {{ original?.word }}</DialogTitle>
                    <DialogDescription>{{ t.descriptionsPreviewNote }}</DialogDescription>
                </DialogHeader>
                <div class="space-y-4 py-2">
                    <section class="rounded-lg border border-border p-4">
                        <h3 class="mb-2 text-sm font-medium text-muted-foreground">{{ t.descriptionsCurrent }}</h3>
                        <p class="break-words leading-6">{{ preview?.original_description }}</p>
                    </section>
                    <section class="rounded-lg border border-primary/40 bg-primary/5 p-4">
                        <h3 class="mb-2 text-sm font-medium">{{ t.descriptionsProposed }}</h3>
                        <p class="break-words leading-6">{{ preview?.description }}</p>
                        <p class="mt-3 break-all text-xs text-muted-foreground">{{ preview?.model }}</p>
                    </section>
                    <p v-if="approvalError" role="alert" class="text-sm text-destructive">
                        {{ t.descriptionsApproveError }}
                    </p>
                </div>
                <DialogFooter class="gap-2">
                    <Button variant="outline" :disabled="approving" @click="discard">{{ t.descriptionsCancel }}</Button>
                    <Button :disabled="approving" @click="approve"
                        ><LoaderCircle v-if="approving" class="mr-2 size-4 animate-spin" />{{
                            t.descriptionsApprove
                        }}</Button
                    >
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </main>
</template>
