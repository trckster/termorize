<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { Check, Forward } from '@lucide/vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { Button } from '@/components/ui/button'

const props = defineProps<{ item: VocabularyItem }>()
const { t } = useI18n()
const { addToast } = useToast()
const busy = ref(false)
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

onBeforeUnmount(() => {
    clearTimeout(copiedTimer)
})

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
    copied.value = false
    clearTimeout(copiedTimer)
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
        copied.value = true
        copiedTimer = setTimeout(() => {
            copied.value = false
        }, 2000)
        addToast({
            title: t.value.collectionLinkCopiedTitle,
            description: t.value.collectionLinkCopiedDescription,
            variant: 'success',
            duration: 2500,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.collectionLinkCopiedErrorDescription,
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
        :data-state="copied ? 'copied' : busy ? 'loading' : 'idle'"
        :aria-disabled="busy"
        :aria-busy="busy"
        :aria-label="`${t.vocabularyShareAction} ${item.translation.original.word}`"
        :title="copied ? t.collectionLinkCopiedTitle : t.vocabularyShareAction"
        @click="shareWord"
    >
        <Check v-if="copied" class="h-4 w-4 text-success" data-share-icon="copied" aria-hidden="true" />
        <Forward v-else class="h-4 w-4" data-share-icon="share" aria-hidden="true" />
    </Button>
</template>
