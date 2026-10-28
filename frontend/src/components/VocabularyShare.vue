<script setup lang="ts">
import { ref } from 'vue'
import { Forward, Loader2 } from '@lucide/vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary'
import { useI18n } from '@/composables/useI18n'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

const props = defineProps<{ item: VocabularyItem }>()
const { t } = useI18n()
const open = ref(false)
const busy = ref(false)
const error = ref('')
const copied = ref(false)
const shareURL = ref('')

async function startShare() {
    open.value = true
    busy.value = true
    error.value = ''
    copied.value = false
    shareURL.value = ''
    try {
        const share = await vocabularyApi.shareVocabulary(props.item.id)
        shareURL.value = `${window.location.origin}${share.share_path}`
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
        error.value = t.value.vocabularyShareCopyError
    }
}

async function stopSharing() {
    busy.value = true
    error.value = ''
    try {
        await vocabularyApi.stopSharing(props.item.id)
        shareURL.value = ''
        open.value = false
    } catch {
        error.value = t.value.vocabularyShareStopError
    } finally {
        busy.value = false
    }
}
</script>

<template>
    <Button
        variant="ghost"
        size="icon"
        class="min-h-11 min-w-11 text-muted-foreground hover:text-foreground"
        :aria-label="`${t.vocabularyShareAction} ${item.translation.original.word}`"
        :title="t.vocabularyShareAction"
        @click="startShare"
    >
        <Forward class="h-4 w-4" aria-hidden="true" />
    </Button>
    <Dialog
        :open="open"
        @update:open="
            (value) => {
                if (!busy) open = value
            }
        "
    >
        <DialogContent
            class="sm:max-w-md"
            :hide-close="busy"
            @interact-outside="
                (event) => {
                    if (busy) event.preventDefault()
                }
            "
            @escape-key-down="
                (event) => {
                    if (busy) event.preventDefault()
                }
            "
        >
            <DialogHeader>
                <DialogTitle>{{ t.vocabularyShareAction }}</DialogTitle>
                <DialogDescription>{{ t.vocabularyShareDescription }}</DialogDescription>
            </DialogHeader>
            <p class="break-words text-lg font-medium text-foreground">
                <span :lang="item.translation.original.language">{{ item.translation.original.word }}</span>
                <span class="text-muted-foreground"> — </span>
                <span :lang="item.translation.translation.language">{{ item.translation.translation.word }}</span>
            </p>
            <div v-if="busy" class="flex items-center gap-2 text-sm text-muted-foreground" role="status">
                <Loader2 class="h-4 w-4 animate-spin motion-reduce:animate-none" />{{ t.vocabularyShareLoading }}
            </div>
            <template v-if="shareURL">
                <div class="space-y-2">
                    <label :for="`word-share-${item.id}`" class="block text-sm font-medium">{{
                        t.vocabularyShareLink
                    }}</label>
                    <input
                        :id="`word-share-${item.id}`"
                        :value="shareURL"
                        readonly
                        class="min-h-11 w-full rounded-md border border-border bg-background px-3 py-2 text-base text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                        @focus="($event.target as HTMLInputElement).select()"
                    />
                </div>
                <div class="flex flex-wrap items-center gap-3">
                    <Button class="min-h-11" :disabled="busy" @click="copyLink">{{
                        copied ? t.collectionLinkCopiedTitle : t.vocabularyShareCopy
                    }}</Button>
                    <Button variant="outline" class="min-h-11" :disabled="busy" @click="stopSharing">{{
                        t.vocabularyShareStop
                    }}</Button>
                </div>
                <p class="text-sm text-muted-foreground" role="status">
                    {{ copied ? t.collectionLinkCopiedDescription : t.vocabularyShareStopHint }}
                </p>
            </template>
            <Button v-else-if="!busy" variant="outline" @click="startShare">{{ t.commonRetry }}</Button>
            <p v-if="error" class="text-sm text-destructive" role="alert">{{ error }}</p>
        </DialogContent>
    </Dialog>
</template>
