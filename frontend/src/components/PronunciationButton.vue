<template>
    <button
        type="button"
        class="relative inline-flex shrink-0 items-center justify-center transition-[color,background-color,border-color,box-shadow,transform] duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.97] disabled:pointer-events-none"
        :class="[
            prominent
                ? status === 'playing'
                    ? 'min-h-14 min-w-44 gap-2.5 rounded-lg border border-primary bg-primary px-5 py-3 text-sm font-semibold text-primary-foreground shadow-md hover:bg-primary/90 disabled:opacity-70'
                    : 'min-h-14 min-w-44 gap-2.5 rounded-lg border border-primary/45 bg-primary/10 px-5 py-3 text-sm font-semibold text-foreground shadow-sm hover:border-primary/65 hover:bg-primary/15 disabled:opacity-70'
                : status === 'playing'
                  ? 'size-11 rounded-full bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-40'
                  : 'size-11 rounded-full text-muted-foreground hover:bg-accent hover:text-accent-foreground disabled:opacity-40',
        ]"
        :disabled="!wordId || status === 'loading'"
        :aria-label="buttonLabel"
        :aria-pressed="status === 'playing'"
        :title="status === 'error' ? errorLabel : buttonLabel"
        @click="togglePlayback"
    >
        <LoaderCircle
            v-if="status === 'loading'"
            :class="prominent ? 'size-5' : 'size-[18px]'"
            class="motion-safe:animate-spin"
            aria-hidden="true"
        />
        <Pause
            v-else-if="status === 'playing'"
            :class="prominent ? 'size-5' : 'size-[18px]'"
            class="fill-current"
            aria-hidden="true"
        />
        <CircleAlert
            v-else-if="status === 'error'"
            :class="prominent ? 'size-5' : 'size-[18px]'"
            class="text-destructive"
            aria-hidden="true"
        />
        <Volume2
            v-else
            :class="[prominent ? 'size-5' : 'size-[18px]', prominent ? 'text-primary' : '']"
            aria-hidden="true"
        />
        <span v-if="prominent">{{ visibleLabel }}</span>
        <span class="sr-only" aria-live="polite">{{ liveStatus }}</span>
    </button>
</template>

<script lang="ts">
const pronunciationBlobCache = new Map<string, Blob>()
let deactivateCurrentPronunciation: (() => void) | null = null
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { CircleAlert, LoaderCircle, Pause, Volume2 } from 'lucide-vue-next'
import { pronunciationApi } from '@/api/pronunciation.ts'

type PlaybackStatus = 'idle' | 'loading' | 'playing' | 'error'

const props = defineProps<{
    wordId: string | null | undefined
    word: string
    listenLabel: string
    pauseLabel: string
    loadingLabel: string
    errorLabel: string
    prominent?: boolean
}>()

const status = ref<PlaybackStatus>('idle')
let audio: HTMLAudioElement | null = null
let objectUrl: string | null = null
let requestSequence = 0

const formatLabel = (template: string) => template.replace('{word}', props.word)
const buttonLabel = computed(() => formatLabel(status.value === 'playing' ? props.pauseLabel : props.listenLabel))
const visibleLabel = computed(() => (status.value === 'loading' ? formatLabel(props.loadingLabel) : buttonLabel.value))
const liveStatus = computed(() => {
    if (status.value === 'loading') return formatLabel(props.loadingLabel)
    if (status.value === 'error') return formatLabel(props.errorLabel)
    return ''
})

const releaseAudio = () => {
    if (audio) {
        audio.pause()
        audio.removeAttribute('src')
        audio.load()
        audio = null
    }
    if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
        objectUrl = null
    }
}

const stopPlayback = () => {
    if (audio) {
        audio.pause()
    }
    if (status.value === 'playing' || status.value === 'loading') {
        status.value = 'idle'
    }
    if (deactivateCurrentPronunciation === deactivatePlayback) {
        deactivateCurrentPronunciation = null
    }
}

const deactivatePlayback = () => {
    requestSequence += 1
    stopPlayback()
}

const playBlob = async (blob: Blob, sequence: number) => {
    if (sequence !== requestSequence) return

    releaseAudio()
    objectUrl = URL.createObjectURL(blob)
    const nextAudio = new Audio(objectUrl)
    audio = nextAudio

    const markPlaybackIdle = () => {
        if (audio === nextAudio) {
            status.value = 'idle'
            if (deactivateCurrentPronunciation === deactivatePlayback) {
                deactivateCurrentPronunciation = null
            }
        }
    }

    nextAudio.addEventListener('ended', markPlaybackIdle)
    nextAudio.addEventListener('pause', markPlaybackIdle)
    nextAudio.addEventListener(
        'error',
        () => {
            if (audio === nextAudio) {
                releaseAudio()
                status.value = 'error'
                if (deactivateCurrentPronunciation === deactivatePlayback) {
                    deactivateCurrentPronunciation = null
                }
            }
        },
        { once: true }
    )

    await nextAudio.play()
    if (
        sequence === requestSequence &&
        audio === nextAudio &&
        !nextAudio.paused &&
        !nextAudio.ended &&
        deactivateCurrentPronunciation === deactivatePlayback
    ) {
        status.value = 'playing'
    }
}

const resumeAudio = async (sequence: number) => {
    if (!audio || sequence !== requestSequence) return

    if (audio.ended || (Number.isFinite(audio.duration) && audio.currentTime >= audio.duration)) {
        audio.currentTime = 0
    }

    await audio.play()
    if (
        sequence === requestSequence &&
        !audio.paused &&
        !audio.ended &&
        deactivateCurrentPronunciation === deactivatePlayback
    ) {
        status.value = 'playing'
    }
}

const togglePlayback = async () => {
    if (!props.wordId || status.value === 'loading') return

    if (status.value === 'playing') {
        stopPlayback()
        return
    }

    deactivateCurrentPronunciation?.()
    deactivateCurrentPronunciation = deactivatePlayback

    if (audio) {
        try {
            await resumeAudio(requestSequence)
        } catch {
            releaseAudio()
            status.value = 'error'
            if (deactivateCurrentPronunciation === deactivatePlayback) {
                deactivateCurrentPronunciation = null
            }
        }
        return
    }

    const sequence = ++requestSequence
    status.value = 'loading'

    try {
        let blob = pronunciationBlobCache.get(props.wordId)
        if (!blob) {
            blob = await pronunciationApi.getWordAudio(props.wordId)
            pronunciationBlobCache.set(props.wordId, blob)
        }
        await playBlob(blob, sequence)
    } catch {
        if (sequence === requestSequence) {
            releaseAudio()
            status.value = 'error'
            if (deactivateCurrentPronunciation === deactivatePlayback) {
                deactivateCurrentPronunciation = null
            }
        }
    }
}

watch(
    () => props.wordId,
    () => {
        deactivatePlayback()
        releaseAudio()
        status.value = 'idle'
    }
)

onBeforeUnmount(() => {
    deactivatePlayback()
    releaseAudio()
})
</script>
