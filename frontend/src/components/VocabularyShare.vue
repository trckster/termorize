<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2, Share2 } from '@lucide/vue'
import { vocabularyApi } from '@/api/vocabulary'
import type { CollectionDetail } from '@/api/collections'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

const props = defineProps<{ selectedIds: string[]; selecting: boolean; hasItems: boolean }>()
const emit = defineEmits<{ toggle: []; shared: [] }>()
const { t } = useI18n()
const open = ref(false)
const title = ref('')
const busy = ref(false)
const error = ref('')
const collection = ref<CollectionDetail | null>(null)
const copied = ref(false)
const shareURL = ref('')

function startShare() {
    title.value = ''
    collection.value = null
    error.value = ''
    copied.value = false
    open.value = true
}

async function createShare() {
    if (busy.value || !title.value.trim() || !props.selectedIds.length) return
    busy.value = true
    error.value = ''
    try {
        collection.value = await vocabularyApi.shareVocabulary(title.value.trim(), props.selectedIds)
        shareURL.value = `${window.location.origin}${collection.value.share_path}`
        emit('shared')
    } catch {
        error.value = t.value.vocabularyShareError
    } finally {
        busy.value = false
    }
}

async function copyLink() {
    try {
        await navigator.clipboard.writeText(shareURL.value)
        copied.value = true
        error.value = ''
    } catch {
        error.value = t.value.collectionLinkCopiedErrorDescription
    }
}
</script>

<template>
    <div class="mb-4 flex flex-wrap items-center gap-3">
        <Button variant="outline" class="min-h-11" :disabled="!selecting && !hasItems" @click="emit('toggle')">
            <Share2 v-if="!selecting" class="mr-2 h-4 w-4" />
            {{ selecting ? t.cancel : t.vocabularyShareSelect }}
        </Button>
        <template v-if="selecting">
            <span class="text-sm text-muted-foreground" role="status"
                >{{ selectedIds.length }} {{ t.vocabularyShareSelected }}</span
            >
            <Button class="min-h-11" :disabled="selectedIds.length === 0" @click="startShare">
                {{ t.vocabularyShareCreate }}
            </Button>
            <p class="w-full text-sm text-muted-foreground">{{ t.vocabularyShareSelectionHint }}</p>
        </template>
    </div>
    <Dialog
        v-model:open="open"
        @update:open="
            () => {
                if (busy) open = true
            }
        "
    >
        <DialogContent class="sm:max-w-md">
            <DialogHeader>
                <DialogTitle>{{ t.vocabularyShareCreate }}</DialogTitle>
                <DialogDescription>{{
                    collection ? t.vocabularyShareReady : t.vocabularyShareDescription
                }}</DialogDescription>
            </DialogHeader>
            <form v-if="!collection" class="space-y-4" @submit.prevent="createShare">
                <div class="space-y-2">
                    <label for="vocabulary-share-title" class="text-sm font-medium">{{ t.vocabularyShareTitle }}</label>
                    <input
                        id="vocabulary-share-title"
                        v-model="title"
                        required
                        maxlength="200"
                        :disabled="busy"
                        class="min-h-11 w-full rounded-md border border-border bg-background px-3 py-2 text-base text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    />
                </div>
                <Button type="submit" class="min-h-11" :disabled="busy || !title.trim()">
                    <Loader2 v-if="busy" class="mr-2 h-4 w-4 animate-spin motion-reduce:animate-none" />
                    {{ t.vocabularyShareCreate }} ({{ selectedIds.length }})
                </Button>
            </form>
            <div v-else class="space-y-4">
                <label for="vocabulary-share-link" class="block text-sm font-medium">{{ t.vocabularyShareLink }}</label>
                <input
                    id="vocabulary-share-link"
                    :value="shareURL"
                    readonly
                    class="min-h-11 w-full rounded-md border border-border bg-background px-3 py-2 text-base text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    @focus="($event.target as HTMLInputElement).select()"
                />
                <div class="flex flex-wrap items-center gap-3">
                    <Button class="min-h-11" @click="copyLink">{{
                        copied ? t.collectionLinkCopiedTitle : t.vocabularyShareCopy
                    }}</Button>
                    <Button as-child variant="outline" class="min-h-11">
                        <RouterLink :to="`/collections/${collection.id}`">{{ t.vocabularyShareManage }}</RouterLink>
                    </Button>
                </div>
            </div>
            <p v-if="error" class="text-sm text-destructive" role="alert">{{ error }}</p>
        </DialogContent>
    </Dialog>
</template>
