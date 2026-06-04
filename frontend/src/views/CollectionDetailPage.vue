<template>
    <main class="px-4 py-4 sm:px-6 sm:py-8">
        <div class="mx-auto max-w-4xl">
            <router-link
                to="/collections"
                class="mb-6 inline-flex items-center gap-1 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
                <ArrowLeft class="h-4 w-4" />
                {{ t.collectionBack }}
            </router-link>

            <div v-if="isLoading && !collection" class="flex min-h-72 items-center justify-center">
                <Loader2 class="h-6 w-6 animate-spin text-muted-foreground" />
            </div>

            <div
                v-else-if="errorMessage"
                class="flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 px-6 text-center"
            >
                <h2 class="text-xl font-semibold text-foreground">{{ t.collectionLoadErrorTitle }}</h2>
                <p class="mt-2 max-w-md text-sm text-muted-foreground">{{ errorMessage }}</p>
                <Button variant="outline" class="mt-5" @click="$router.push('/collections')">{{
                    t.collectionJoinBack
                }}</Button>
            </div>

            <div v-else-if="collection">
                <div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                    <div class="min-w-0">
                        <div class="flex flex-wrap items-center gap-2">
                            <h1 class="break-words text-3xl font-bold text-foreground">{{ collection.title }}</h1>
                            <Button
                                v-if="canManage"
                                variant="ghost"
                                size="icon"
                                class="h-8 w-8 text-muted-foreground hover:text-foreground"
                                :aria-label="t.collectionEditTitle"
                                @click="isEditTitleOpen = true"
                            >
                                <Pencil class="h-4 w-4" />
                            </Button>
                            <span v-if="collection.is_admin">
                                <span
                                    v-if="!collection.is_published && isAdmin"
                                    class="rounded bg-amber-500/15 px-2 py-0.5 text-xs font-medium text-amber-600 dark:text-amber-400"
                                >
                                    {{ t.collectionsDraftBadge }}
                                </span>
                                <span
                                    v-else-if="isAdmin"
                                    class="rounded border border-primary/30 bg-primary/30 px-2 py-0.5 text-xs font-medium text-primary"
                                >
                                    {{ t.collectionsGlobalBadge }}
                                </span>
                            </span>
                            <span
                                v-else-if="collection.is_owner"
                                class="rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground"
                            >
                                {{ t.collectionsPrivateBadge }}
                            </span>
                        </div>
                        <div class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
                            <span
                                >{{ formatNumber(collection.translation_count) }}
                                {{ t.collectionTranslationsLabel }}</span
                            >
                            <span v-if="collection.user_add_count > 0">{{ saves(collection.user_add_count) }}</span>
                            <span v-if="!collection.is_admin && !collection.is_owner && collection.owner_username">
                                {{ t.collectionBy }} @{{ collection.owner_username }}
                            </span>
                        </div>
                        <div v-if="collection.languages.length > 0" class="mt-3">
                            <span class="text-xs font-medium uppercase tracking-wide text-muted-foreground">{{
                                t.collectionLanguagesUsed
                            }}</span>
                            <div class="mt-1 flex flex-wrap items-center gap-2 text-xl">
                                <span
                                    v-for="lang in collection.languages"
                                    :key="lang"
                                    class="inline-flex items-center gap-1 text-sm text-foreground"
                                >
                                    <span role="img" :aria-label="getLanguageName(lang)">{{
                                        settingsStore.getFlag(lang)
                                    }}</span>
                                    <span>{{ getLanguageName(lang) }}</span>
                                </span>
                            </div>
                        </div>
                    </div>

                    <div class="flex flex-wrap items-center gap-2">
                        <Button
                            v-if="canManage && collection.is_admin"
                            :variant="collection.is_published ? 'outline' : 'default'"
                            size="sm"
                            :disabled="isPublishing"
                            @click="handleTogglePublish"
                        >
                            <Loader2 v-if="isPublishing" class="mr-2 h-4 w-4 animate-spin" />
                            <template v-else>
                                <EyeOff v-if="collection.is_published" class="mr-2 h-4 w-4" />
                                <Globe v-else class="mr-2 h-4 w-4" />
                            </template>
                            {{ collection.is_published ? t.collectionUnpublish : t.collectionPublish }}
                        </Button>
                        <Button v-if="inviteLink" variant="outline" size="sm" @click="isShareDialogOpen = true">
                            <Share2 class="mr-2 h-4 w-4" />
                            {{ t.collectionShare }}
                        </Button>
                        <Dialog v-if="canManage">
                            <DialogTrigger as-child>
                                <Button variant="outline" size="sm" class="text-destructive hover:bg-destructive/10">
                                    <Trash2 class="mr-2 h-4 w-4" />
                                    {{ t.collectionDelete }}
                                </Button>
                            </DialogTrigger>
                            <DialogContent class="sm:max-w-md">
                                <DialogHeader>
                                    <DialogTitle>{{ t.collectionDeleteDialogTitle }}</DialogTitle>
                                    <DialogDescription>
                                        {{ t.collectionDeleteConfirmPrefix
                                        }}<span class="font-medium text-foreground">{{ collection.title }}</span
                                        >{{ t.collectionDeleteConfirmSuffix }}
                                    </DialogDescription>
                                </DialogHeader>
                                <DialogFooter class="flex gap-2 sm:justify-end">
                                    <DialogClose as-child>
                                        <Button type="button" variant="secondary">{{ t.cancel }}</Button>
                                    </DialogClose>
                                    <DialogClose as-child>
                                        <Button
                                            type="button"
                                            variant="destructive"
                                            :disabled="isDeleting"
                                            @click="handleDelete"
                                        >
                                            <Loader2 v-if="isDeleting" class="mr-2 h-4 w-4 animate-spin" />
                                            {{ isDeleting ? t.deleting : t.delete }}
                                        </Button>
                                    </DialogClose>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    </div>
                </div>

                <Dialog v-if="canManage" v-model:open="isEditTitleOpen">
                    <DialogContent class="sm:max-w-md">
                        <DialogHeader>
                            <DialogTitle>{{ t.collectionEditTitleDialogTitle }}</DialogTitle>
                            <DialogDescription>{{ t.collectionEditTitleDialogDescription }}</DialogDescription>
                        </DialogHeader>
                        <form @submit.prevent="handleSaveTitle" class="space-y-4 py-4">
                            <div class="space-y-2">
                                <label for="collection-edit-title" class="text-sm font-medium">{{
                                    t.collectionsTitleLabel
                                }}</label>
                                <input
                                    id="collection-edit-title"
                                    v-model="editTitle"
                                    type="text"
                                    :placeholder="t.collectionsTitlePlaceholder"
                                    maxlength="255"
                                    class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                />
                            </div>
                            <DialogFooter class="justify-center sm:justify-center pt-2">
                                <Button type="submit" :disabled="isSavingTitle || editTitle.trim().length === 0">
                                    <Loader2 v-if="isSavingTitle" class="mr-2 h-4 w-4 animate-spin" />
                                    {{ isSavingTitle ? t.saving : t.save }}
                                </Button>
                            </DialogFooter>
                        </form>
                    </DialogContent>
                </Dialog>

                <div
                    v-if="collection.is_admin && !collection.is_published"
                    class="mb-6 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-700 dark:text-amber-300"
                >
                    {{ t.collectionDraftNotice }}
                </div>

                <div class="mb-6 flex flex-wrap items-center gap-3">
                    <Button
                        :disabled="isAddingToVocabulary || collection.translation_count === 0 || (isSelecting && selectedCount === 0)"
                        @click="handleAddToVocabulary"
                    >
                        <Loader2 v-if="isAddingToVocabulary" class="mr-2 h-4 w-4 animate-spin" />
                        <BookmarkPlus v-else class="mr-2 h-4 w-4" />
                        {{ addToVocabularyLabel }}
                    </Button>
                    <Button
                        v-if="!isSelecting"
                        variant="outline"
                        :disabled="isAddingToVocabulary || collection.translation_count === 0"
                        @click="startSelecting"
                    >
                        <ListChecks class="mr-2 h-4 w-4" />
                        {{ t.collectionAddSelectButton }}
                    </Button>
                    <Button v-else variant="ghost" :disabled="isAddingToVocabulary" @click="cancelSelecting">
                        {{ t.collectionSelectCancel }}
                    </Button>
                </div>

                <Dialog v-if="canManage" v-model:open="isAddTranslationOpen">
                    <DialogContent class="sm:max-w-md">
                        <DialogHeader>
                            <DialogTitle>{{ t.collectionAddTranslationDialogTitle }}</DialogTitle>
                            <DialogDescription>{{ t.collectionAddTranslationDialogDescription }}</DialogDescription>
                        </DialogHeader>
                        <form @submit.prevent="handleAddTranslation" class="space-y-4 py-4">
                            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div class="space-y-2">
                                    <label for="collection-word1" class="text-sm font-medium">{{
                                        t.vocabularyWord1
                                    }}</label>
                                    <input
                                        id="collection-word1"
                                        ref="newTranslationWord1Ref"
                                        v-model="newTranslation.word1"
                                        type="text"
                                        :placeholder="t.vocabularyWord1Placeholder"
                                        maxlength="500"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                                <div class="space-y-2">
                                    <label for="collection-word2" class="text-sm font-medium">{{
                                        t.vocabularyWord2
                                    }}</label>
                                    <input
                                        id="collection-word2"
                                        v-model="newTranslation.word2"
                                        type="text"
                                        :placeholder="t.vocabularyWord2Placeholder"
                                        maxlength="500"
                                        class="w-full px-3 py-2 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                                    />
                                </div>
                            </div>
                            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">{{ t.vocabularyLanguage1 }}</label>
                                    <LanguageSelector
                                        v-model="newTranslation.language1"
                                        :placeholder="t.vocabularySelectLanguagePlaceholder"
                                        :disabled-values="[newTranslation.language2]"
                                        :aria-label="t.vocabularyLanguage1"
                                        :empty-text="t.languageSelectorNoResults"
                                    />
                                </div>
                                <div class="space-y-2">
                                    <label class="text-sm font-medium">{{ t.vocabularyLanguage2 }}</label>
                                    <LanguageSelector
                                        v-model="newTranslation.language2"
                                        :placeholder="t.vocabularySelectLanguagePlaceholder"
                                        :disabled-values="[newTranslation.language1]"
                                        :aria-label="t.vocabularyLanguage2"
                                        :empty-text="t.languageSelectorNoResults"
                                    />
                                </div>
                            </div>
                            <label class="flex items-center gap-2 text-sm font-medium text-foreground">
                                <input
                                    v-model="openAddTranslationAgain"
                                    type="checkbox"
                                    class="h-4 w-4 rounded border-border text-primary accent-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                                />
                                {{ t.collectionOpenAddTranslationAgain }}
                            </label>
                            <DialogFooter class="justify-center sm:justify-center pt-2">
                                <Button type="submit" :disabled="isAddingTranslation || !isTranslationFormValid">
                                    <Loader2 v-if="isAddingTranslation" class="mr-2 h-4 w-4 animate-spin" />
                                    {{ isAddingTranslation ? t.adding : t.collectionAddTranslationButton }}
                                </Button>
                            </DialogFooter>
                        </form>
                    </DialogContent>
                </Dialog>

                <h2 class="mb-3 text-lg font-semibold text-foreground">{{ t.collectionTranslationsTitle }}</h2>

                <div v-if="collection.translations.length > 0">
                    <VueDraggable
                        v-model="orderedTranslations"
                        :disabled="!canManage"
                        handle=".collection-drag-handle"
                        :animation="150"
                        class="space-y-2"
                        @end="handleReorder"
                    >
                        <div
                            v-for="item in orderedTranslations"
                            :key="item.id"
                            class="group flex items-center justify-between gap-4 rounded-lg border border-border bg-card p-4 transition-colors hover:bg-accent/50"
                        >
                            <div class="flex min-w-0 items-center gap-2">
                                <span
                                    v-if="canManage"
                                    class="collection-drag-handle shrink-0 cursor-grab touch-none text-muted-foreground transition-colors hover:text-foreground active:cursor-grabbing"
                                    role="button"
                                    :aria-label="t.collectionReorderTranslationLabel"
                                >
                                    <GripVertical class="h-4 w-4" />
                                </span>
                                <h3 class="flex min-w-0 items-center gap-2 font-semibold text-foreground">
                                    <span
                                        class="text-xl"
                                        role="img"
                                        :aria-label="getLanguageName(item.original.language)"
                                        >{{ settingsStore.getFlag(item.original.language) }}</span
                                    >
                                    <span class="min-w-0 break-words text-lg">{{ item.original.word }}</span>
                                    <span class="text-muted-foreground">-</span>
                                    <span class="min-w-0 break-words text-lg">{{ item.translation.word }}</span>
                                    <span
                                        class="text-xl"
                                        role="img"
                                        :aria-label="getLanguageName(item.translation.language)"
                                        >{{ settingsStore.getFlag(item.translation.language) }}</span
                                    >
                                </h3>
                            </div>
                            <div class="flex shrink-0 items-center gap-2">
                                <input
                                    v-if="isSelecting"
                                    type="checkbox"
                                    class="h-5 w-5 shrink-0 cursor-pointer rounded border-border text-primary accent-primary focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                                    :checked="selectedIds.has(item.id)"
                                    :disabled="isAddingToVocabulary"
                                    :aria-label="t.collectionSelectTranslationLabel"
                                    @change="toggleSelection(item.id)"
                                />
                                <Button
                                    v-if="canManage"
                                    variant="ghost"
                                    size="icon"
                                    class="shrink-0 text-muted-foreground opacity-100 transition-opacity hover:bg-destructive/10 hover:text-destructive focus:opacity-100 md:opacity-0 md:group-hover:opacity-100"
                                    :aria-label="t.collectionRemoveTranslationLabel"
                                    :disabled="removingId === item.id"
                                    @click="handleRemoveTranslation(item.id)"
                                >
                                    <Loader2 v-if="removingId === item.id" class="h-4 w-4 animate-spin" />
                                    <Trash2 v-else class="h-4 w-4" />
                                </Button>
                            </div>
                        </div>
                    </VueDraggable>

                    <button
                        v-if="canManage"
                        type="button"
                        class="mt-2 flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-border bg-card/40 p-4 text-sm font-medium text-muted-foreground transition-colors hover:border-primary/50 hover:bg-accent/50 hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        @click="isAddTranslationOpen = true"
                    >
                        <Plus class="h-4 w-4" />
                        {{ t.collectionAddTranslationButton }}
                    </button>
                </div>

                <div
                    v-else
                    class="flex min-h-48 flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 px-6 text-center"
                >
                    <p class="max-w-md text-sm text-muted-foreground">{{ t.collectionDetailEmpty }}</p>
                    <Button v-if="canManage" class="mt-5" @click="isAddTranslationOpen = true">
                        <Plus class="mr-2 h-4 w-4" />
                        {{ t.collectionAddTranslationButton }}
                    </Button>
                </div>
            </div>
        </div>

        <Dialog v-if="collection && inviteLink" v-model:open="isShareDialogOpen">
            <DialogContent class="w-max max-w-[95vw]">
                <DialogHeader>
                    <DialogTitle>{{ t.collectionShareDialogTitle }}</DialogTitle>
                    <DialogDescription>{{ t.collectionShareDialogDescription }}</DialogDescription>
                </DialogHeader>
                <div class="flex flex-col gap-3 py-2">
                    <div
                        ref="linkRef"
                        class="w-max whitespace-nowrap select-all rounded-md border border-border bg-muted px-3 py-2 text-sm font-mono text-foreground"
                    >
                        {{ inviteLink }}
                    </div>
                    <Button variant="outline" size="sm" class="w-full" @click="copyInviteLink">
                        <Check v-if="justCopied" class="mr-2 h-4 w-4 text-green-600" />
                        <Copy v-else class="mr-2 h-4 w-4" />
                        {{ justCopied ? t.collectionCopied : t.collectionCopyLink }}
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    </main>
</template>

<script setup lang="ts">
import { collectionsApi, type CollectionDetail, type CollectionTranslation } from '@/api/collections.ts'
import { VueDraggable } from 'vue-draggable-plus'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.ts'
import { useSettingsStore } from '@/stores/settings.ts'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast.ts'
import LanguageSelector from '@/components/LanguageSelector.vue'
import { Button } from '@/components/ui/button'
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'
import { formatNumber } from '@/lib/utils.ts'
import {
    ArrowLeft,
    BookmarkPlus,
    Check,
    Copy,
    EyeOff,
    Globe,
    GripVertical,
    ListChecks,
    Loader2,
    Pencil,
    Plus,
    Share2,
    Trash2,
} from 'lucide-vue-next'

const { t, saves } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { addToast } = useToast()

const collection = ref<CollectionDetail | null>(null)
const isLoading = ref(false)
const errorMessage = ref('')
const isAddingToVocabulary = ref(false)
const isShareDialogOpen = ref(false)
const isDeleting = ref(false)
const isPublishing = ref(false)
const removingId = ref<string | null>(null)
const orderedTranslations = ref<CollectionTranslation[]>([])
const isSelecting = ref(false)
const selectedIds = ref<Set<string>>(new Set())
const justCopied = ref(false)
let copyTimeoutId: ReturnType<typeof setTimeout> | null = null

const isAddTranslationOpen = ref(false)
const isAddingTranslation = ref(false)
const openAddTranslationAgain = ref(false)
const isEditTitleOpen = ref(false)
const isSavingTitle = ref(false)
const editTitle = ref('')
const linkRef = ref<HTMLDivElement | null>(null)
const newTranslationWord1Ref = ref<HTMLInputElement | null>(null)

type NewTranslationForm = {
    word1: string
    word2: string
    language1: string
    language2: string
}

const defaultNewTranslation = (): NewTranslationForm => ({
    word1: '',
    word2: '',
    language1: authStore.user?.settings.translation_source_language || 'en',
    language2:
        authStore.user?.settings.translation_target_language === authStore.user?.settings.translation_source_language
            ? authStore.user?.settings.translation_source_language === 'en'
                ? 'ru'
                : 'en'
            : authStore.user?.settings.translation_target_language || 'ru',
})

const newTranslation = ref<NewTranslationForm>(defaultNewTranslation())

const isTranslationFormValid = computed(
    () => newTranslation.value.word1.trim().length > 0 && newTranslation.value.word2.trim().length > 0
)

const isAdmin = computed(() => !!authStore.user?.is_admin)

const canManage = computed(
    () => !!collection.value && (collection.value.is_owner || (collection.value.is_admin && isAdmin.value))
)

const inviteLink = computed(() =>
    collection.value?.invite_token ? `${window.location.origin}/collections/join/${collection.value.invite_token}` : ''
)

const selectedCount = computed(() => selectedIds.value.size)

const allSelected = computed(
    () => orderedTranslations.value.length > 0 && selectedCount.value === orderedTranslations.value.length
)

const addToVocabularyLabel = computed(() => {
    if (isAddingToVocabulary.value) return t.value.adding
    if (isSelecting.value && !allSelected.value) return `${t.value.collectionAddSelectedLabel} ${selectedCount.value}`
    return t.value.collectionAddToVocabulary
})

const startSelecting = () => {
    selectedIds.value = new Set(orderedTranslations.value.map((item) => item.id))
    isSelecting.value = true
}

const cancelSelecting = () => {
    isSelecting.value = false
    selectedIds.value = new Set()
}

const toggleSelection = (id: string) => {
    const next = new Set(selectedIds.value)
    if (next.has(id)) {
        next.delete(id)
    } else {
        next.add(id)
    }
    selectedIds.value = next
}

const getLanguageName = (code: string) =>
    settingsStore.languageOptions.find((l) => l.code === code)?.name || code.toUpperCase()

const focusNewTranslationWord1 = async () => {
    await nextTick()
    newTranslationWord1Ref.value?.focus()
}

const clearNewTranslationWords = () => {
    newTranslation.value = {
        ...newTranslation.value,
        word1: '',
        word2: '',
    }
}

watch(isAddTranslationOpen, (isOpen) => {
    if (isOpen) {
        newTranslation.value = defaultNewTranslation()
        openAddTranslationAgain.value = false
        void focusNewTranslationWord1()
    }
})

watch(isEditTitleOpen, (isOpen) => {
    if (isOpen && collection.value) {
        editTitle.value = collection.value.title
    }
})

watch(
    () => collection.value?.translations,
    (translations) => {
        orderedTranslations.value = translations ? [...translations] : []
    },
    { immediate: true }
)

const fetchCollection = async (id: string) => {
    isLoading.value = true
    errorMessage.value = ''

    try {
        collection.value = await collectionsApi.getCollection(id)
    } catch {
        collection.value = null
        errorMessage.value = t.value.collectionLoadErrorDescription
    } finally {
        isLoading.value = false
    }
}

const handleAddToVocabulary = async () => {
    if (!collection.value) return
    if (isSelecting.value && selectedCount.value === 0) return
    const translationIds = isSelecting.value ? Array.from(selectedIds.value) : undefined

    isAddingToVocabulary.value = true
    try {
        const result = await collectionsApi.addToVocabulary(collection.value.id, translationIds)
        if (collection.value) {
            collection.value.user_add_count = result.user_add_count
        }
        cancelSelecting()
        addToast({
            title: t.value.collectionAddedToVocabularyTitle,
            description: `${result.added} ${t.value.collectionAddedLabel}, ${result.skipped} ${t.value.collectionSkippedLabel}`,
            variant: 'success',
            duration: 4000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionAddToVocabularyErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isAddingToVocabulary.value = false
    }
}

const handleAddTranslation = async () => {
    if (!collection.value || !isTranslationFormValid.value) return

    isAddingTranslation.value = true
    try {
        collection.value = await collectionsApi.addTranslation(
            collection.value.id,
            newTranslation.value.word1.trim(),
            newTranslation.value.word2.trim(),
            newTranslation.value.language1,
            newTranslation.value.language2
        )
        if (openAddTranslationAgain.value) {
            clearNewTranslationWords()
            void focusNewTranslationWord1()
        } else {
            isAddTranslationOpen.value = false
        }
        addToast({
            title: t.value.collectionTranslationAddedTitle,
            description: t.value.collectionTranslationAddedDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionTranslationAddErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isAddingTranslation.value = false
    }
}

const handleRemoveTranslation = async (translationId: string) => {
    if (!collection.value) return

    removingId.value = translationId
    try {
        await collectionsApi.removeTranslation(collection.value.id, translationId)
        await fetchCollection(collection.value.id)
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionTranslationRemoveErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        removingId.value = null
    }
}

const handleReorder = async () => {
    if (!collection.value) return

    const newOrder = orderedTranslations.value.map((item) => item.id)
    const currentOrder = collection.value.translations.map((item) => item.id)
    // vue-draggable-plus fires @end even when the item is dropped in its original spot.
    if (newOrder.join(',') === currentOrder.join(',')) return

    try {
        collection.value = await collectionsApi.reorderTranslations(collection.value.id, newOrder)
    } catch {
        orderedTranslations.value = collection.value ? [...collection.value.translations] : []
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionTranslationReorderErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    }
}

const handleDelete = async () => {
    if (!collection.value) return

    isDeleting.value = true
    try {
        await collectionsApi.deleteCollection(collection.value.id)
        addToast({
            title: t.value.collectionDeletedTitle,
            description: t.value.collectionDeletedDescription,
            variant: 'success',
            duration: 3000,
        })
        router.push('/collections')
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionDeleteErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isDeleting.value = false
    }
}

const handleSaveTitle = async () => {
    if (!collection.value || editTitle.value.trim().length === 0) return

    isSavingTitle.value = true
    try {
        collection.value = await collectionsApi.updateTitle(collection.value.id, editTitle.value.trim())
        isEditTitleOpen.value = false
        addToast({
            title: t.value.collectionTitleSavedTitle,
            description: t.value.collectionTitleSavedDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionTitleSaveErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isSavingTitle.value = false
    }
}

const handleTogglePublish = async () => {
    if (!collection.value) return

    const nextPublished = !collection.value.is_published
    isPublishing.value = true
    try {
        collection.value = await collectionsApi.setPublished(collection.value.id, nextPublished)
        addToast({
            title: nextPublished ? t.value.collectionPublishedTitle : t.value.collectionUnpublishedTitle,
            description: nextPublished
                ? t.value.collectionPublishedDescription
                : t.value.collectionUnpublishedDescription,
            variant: 'success',
            duration: 3000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionPublishErrorDescription,
            variant: 'destructive',
            duration: 5000,
        })
    } finally {
        isPublishing.value = false
    }
}

const copyInviteLink = () => {
    if (!inviteLink.value || !linkRef.value) return

    const showSuccess = () => {
        if (copyTimeoutId) clearTimeout(copyTimeoutId)
        justCopied.value = true
        copyTimeoutId = setTimeout(() => {
            justCopied.value = false
        }, 2000)
        addToast({
            title: t.value.collectionLinkCopiedTitle,
            description: t.value.collectionLinkCopiedDescription,
            variant: 'success',
            duration: 2500,
        })
    }

    const showError = () => {
        addToast({
            title: t.value.toastErrorTitle,
            description:
                t.value.collectionLinkCopiedErrorDescription || 'Unable to copy. Please copy the link manually.',
            variant: 'destructive',
            duration: 5000,
        })
    }

    const fallbackCopy = (): boolean => {
        const selection = window.getSelection()
        const range = document.createRange()
        range.selectNodeContents(linkRef.value!)
        selection?.removeAllRanges()
        selection?.addRange(range)
        let ok = false
        try {
            ok = document.execCommand('copy')
        } catch {
            ok = false
        }
        selection?.removeAllRanges()
        return ok
    }

    if (typeof navigator !== 'undefined' && navigator.clipboard && window.isSecureContext) {
        navigator.clipboard
            .writeText(inviteLink.value)
            .then(() => showSuccess())
            .catch(() => {
                if (fallbackCopy()) showSuccess()
                else showError()
            })
        return
    }

    if (fallbackCopy()) showSuccess()
    else showError()
}

watch(
    () => route.params.id,
    (id) => {
        if (typeof id === 'string' && id) {
            void fetchCollection(id)
        }
    }
)

onMounted(async () => {
    const id = route.params.id
    if (typeof id === 'string' && id) {
        await fetchCollection(id)
    }
})
</script>
