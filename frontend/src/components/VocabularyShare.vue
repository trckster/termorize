<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { Check, Forward, Loader2 } from '@lucide/vue'
import { vocabularyApi, type VocabularyItem } from '@/api/vocabulary'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast'
import { Button } from '@/components/ui/button'

const props = defineProps<{ item: VocabularyItem }>()
const { t } = useI18n()
const { addToast } = useToast()
const busy = ref(false)
const showLoading = ref(false)
const copied = ref(false)
let loadingTimer: ReturnType<typeof setTimeout> | undefined
let copiedTimer: ReturnType<typeof setTimeout> | undefined

function finishLoading() {
    clearTimeout(loadingTimer)
    busy.value = false
    showLoading.value = false
}

onBeforeUnmount(() => {
    clearTimeout(loadingTimer)
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
    loadingTimer = setTimeout(() => {
        showLoading.value = true
    }, 250)
    let link: string
    try {
        const share = await vocabularyApi.shareVocabulary(props.item.id)
        link = `${window.location.origin}${share.share_path}`
    } catch {
        addToast({ title: t.value.toastErrorTitle, description: t.value.vocabularyShareError, variant: 'destructive' })
        finishLoading()
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
            description: t.value.vocabularyShareCopiedDescription,
            variant: 'success',
            duration: 2500,
        })
    } catch {
        addToast({
            title: t.value.toastErrorTitle,
            description: t.value.vocabularyShareCopyError,
            variant: 'destructive',
        })
    } finally {
        finishLoading()
    }
}
</script>

<template>
    <Button
        variant="ghost"
        size="icon"
        class="min-h-11 min-w-11 text-muted-foreground hover:text-foreground"
        :class="copied ? 'bg-primary/10 text-primary hover:text-primary' : ''"
        :data-state="copied ? 'copied' : busy ? 'loading' : 'idle'"
        :aria-disabled="busy"
        :aria-busy="busy"
        :aria-label="`${t.vocabularyShareAction} ${item.translation.original.word}`"
        :title="copied ? t.collectionLinkCopiedTitle : t.vocabularyShareAction"
        @click="shareWord"
    >
        <Check v-if="copied" class="share-copied h-4 w-4" data-share-icon="copied" aria-hidden="true" />
        <Loader2
            v-else-if="showLoading"
            class="h-4 w-4 animate-spin motion-reduce:animate-none"
            data-share-icon="loading"
            aria-hidden="true"
        />
        <Forward v-else class="h-4 w-4" aria-hidden="true" />
    </Button>
</template>

<style scoped>
.share-copied :deep(path) {
    stroke-dasharray: 28;
    animation: share-check-draw 650ms cubic-bezier(0.25, 0.1, 0.25, 1) both;
}

@keyframes share-check-draw {
    from {
        stroke-dashoffset: 28;
    }
    to {
        stroke-dashoffset: 0;
    }
}

@media (prefers-reduced-motion: reduce) {
    .share-copied :deep(path) {
        animation: none;
    }
}
</style>
