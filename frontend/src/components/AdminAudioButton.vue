<script lang="ts">
const audioBlobCache = new Map<string, Blob>()
let stopCurrentAudio: (() => void) | null = null
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { CircleAlert, LoaderCircle, Pause, Play } from 'lucide-vue-next'
import { adminApi } from '@/api/admin.ts'

type PlaybackStatus = 'idle' | 'loading' | 'playing' | 'error'

const props = defineProps<{
    pronunciationId: string
    word: string
    listenLabel: string
    pauseLabel: string
    loadingLabel: string
    errorLabel: string
}>()

const status = ref<PlaybackStatus>('idle')
let audio: HTMLAudioElement | null = null
let objectUrl: string | null = null
let requestSequence = 0

const formatLabel = (template: string) => template.replace('{word}', props.word)
const buttonLabel = computed(() => formatLabel(status.value === 'playing' ? props.pauseLabel : props.listenLabel))
const liveStatus = computed(() => {
    if (status.value === 'loading') return formatLabel(props.loadingLabel)
    if (status.value === 'error') return formatLabel(props.errorLabel)
    return ''
})

const releaseAudio = () => {
    audio?.pause()
    audio = null
    if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
        objectUrl = null
    }
}

const stopPlayback = () => {
    audio?.pause()
    if (status.value !== 'error') status.value = 'idle'
    if (stopCurrentAudio === stopPlayback) stopCurrentAudio = null
}

const togglePlayback = async () => {
    if (status.value === 'loading') return
    if (status.value === 'playing') {
        stopPlayback()
        return
    }

    stopCurrentAudio?.()
    stopCurrentAudio = stopPlayback
    const sequence = ++requestSequence
    status.value = 'loading'

    try {
        let blob = audioBlobCache.get(props.pronunciationId)
        if (!blob) {
            blob = await adminApi.getWordPronunciationAudio(props.pronunciationId)
            audioBlobCache.set(props.pronunciationId, blob)
        }
        if (sequence !== requestSequence) return

        releaseAudio()
        objectUrl = URL.createObjectURL(blob)
        const nextAudio = new Audio(objectUrl)
        audio = nextAudio
        nextAudio.addEventListener('ended', stopPlayback, { once: true })
        nextAudio.addEventListener(
            'error',
            () => {
                if (audio === nextAudio) {
                    releaseAudio()
                    status.value = 'error'
                    if (stopCurrentAudio === stopPlayback) stopCurrentAudio = null
                }
            },
            { once: true }
        )
        await nextAudio.play()
        if (sequence === requestSequence && audio === nextAudio && !nextAudio.paused) status.value = 'playing'
    } catch {
        if (sequence === requestSequence) {
            releaseAudio()
            status.value = 'error'
            if (stopCurrentAudio === stopPlayback) stopCurrentAudio = null
        }
    }
}

watch(
    () => props.pronunciationId,
    () => {
        requestSequence += 1
        releaseAudio()
        status.value = 'idle'
    }
)

onBeforeUnmount(() => {
    requestSequence += 1
    releaseAudio()
    if (stopCurrentAudio === stopPlayback) stopCurrentAudio = null
})
</script>

<template>
    <button
        type="button"
        class="inline-flex size-11 shrink-0 items-center justify-center rounded-full border border-input bg-background text-foreground transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 sm:size-10"
        :class="status === 'playing' ? 'border-primary bg-primary text-primary-foreground hover:bg-primary/90' : ''"
        :disabled="status === 'loading'"
        :aria-label="buttonLabel"
        :aria-pressed="status === 'playing'"
        :title="status === 'error' ? formatLabel(errorLabel) : buttonLabel"
        @click="togglePlayback"
    >
        <LoaderCircle v-if="status === 'loading'" class="size-4 motion-safe:animate-spin" aria-hidden="true" />
        <Pause v-else-if="status === 'playing'" class="size-4 fill-current" aria-hidden="true" />
        <CircleAlert v-else-if="status === 'error'" class="size-4 text-destructive" aria-hidden="true" />
        <Play v-else class="size-4 fill-current" aria-hidden="true" />
        <span class="sr-only" aria-live="polite">{{ liveStatus }}</span>
    </button>
</template>
