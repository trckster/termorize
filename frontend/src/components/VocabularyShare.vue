<script setup lang="ts">
import { ref } from 'vue'
import { Forward, Loader2 } from '@lucide/vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { Button } from '@/components/ui/button'

const props = defineProps<{ item: VocabularyItem }>()
const { t } = useI18n()
const { addToast } = useToast()
const busy = ref(false)

async function copyLink(link: string) {
    if (navigator.clipboard && window.isSecureContext) {
        try {
            await navigator.clipboard.writeText(link)
            return
        } catch {
            // Embedded browsers may deny Clipboard API access but allow selection-based copying.
        }
    }
    const previousFocus = document.activeElement
    const textArea = document.createElement('textarea')
    textArea.value = link
    textArea.setAttribute('readonly', '')
    textArea.style.position = 'fixed'
    textArea.style.opacity = '0'
    document.body.appendChild(textArea)
    textArea.select()
    try {
        if (!document.execCommand('copy')) throw new Error('Copy failed')
    } finally {
        textArea.remove()
        if (previousFocus instanceof HTMLElement) previousFocus.focus()
    }
}

async function shareWord() {
    if (busy.value) return
    busy.value = true
    let link: string
    try {
        const share = await vocabularyApi.shareVocabulary(props.item.id)
        link = `${window.location.origin}${share.share_path}`
    } catch {
        addToast({ title: t.value.toastErrorTitle, description: t.value.vocabularyShareError, variant: 'destructive' })
        busy.value = false
        return
    }
    try {
        await copyLink(link)
        addToast({
            title: t.value.collectionLinkCopiedTitle,
            description: t.value.vocabularyShareCopiedDescription,
            variant: 'success',
            duration: 4000,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.vocabularyShareCopyError,
            variant: 'destructive',
        })
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
        :aria-disabled="busy"
        :aria-busy="busy"
        :aria-label="`${t.vocabularyShareAction} ${item.translation.original.word}`"
        :title="t.vocabularyShareAction"
        @click="shareWord"
    >
        <Loader2 v-if="busy" class="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />
        <Forward v-else class="h-4 w-4" aria-hidden="true" />
    </Button>
</template>
