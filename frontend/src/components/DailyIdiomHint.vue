<script lang="ts">
const STORAGE_KEY = 'termorize:daily-idiom-hint-dismissed'
// Mirrors the stored flag so the hint stays dismissed for this visit even when storage is blocked.
let dismissedThisVisit = false

function isDismissed() {
    if (dismissedThisVisit) return true
    try {
        return localStorage.getItem(STORAGE_KEY) === 'true'
    } catch {
        return false
    }
}
</script>

<script setup lang="ts">
import { onBeforeUnmount, ref, useId, watch } from 'vue'
import { useEventListener } from '@vueuse/core'
import { RouterLink } from 'vue-router'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'

const props = defineProps<{ target: HTMLElement | null }>()
const { t } = useI18n()
const titleId = useId()
const bodyId = useId()
const root = ref<HTMLElement | null>(null)
const open = ref(false)
let observer: IntersectionObserver | undefined
let showTimer: ReturnType<typeof setTimeout> | undefined

function stopWatching() {
    observer?.disconnect()
    clearTimeout(showTimer)
}

// Appear once at least half of the idiom card is on screen, never on load below the fold.
watch(
    () => props.target,
    (target) => {
        stopWatching()
        if (!target || isDismissed()) return
        observer = new IntersectionObserver(
            (entries) => {
                if (!entries.some((entry) => entry.isIntersecting)) return
                stopWatching()
                showTimer = setTimeout(() => {
                    open.value = !isDismissed()
                }, 320)
            },
            { threshold: 0.5 }
        )
        observer.observe(target)
    },
    { immediate: true }
)

onBeforeUnmount(stopWatching)

function dismiss() {
    dismissedThisVisit = true
    try {
        localStorage.setItem(STORAGE_KEY, 'true')
    } catch {
        // Storage can be unavailable in private modes; the in-memory flag still applies.
    }
    if (root.value?.contains(document.activeElement)) {
        const next = [...(props.target?.querySelectorAll<HTMLElement>('button, a[href]') ?? [])].find(
            (element) => !root.value?.contains(element)
        )
        next?.focus({ preventScroll: true })
    }
    open.value = false
}

useEventListener(document, 'keydown', (event: KeyboardEvent) => {
    if (event.key !== 'Escape' || event.defaultPrevented || !open.value) return
    const active = document.activeElement
    // Escape belongs to whatever else holds focus (dialogs, comboboxes, text fields).
    if (active && active !== document.body && !props.target?.contains(active)) return
    dismiss()
})

useEventListener(window, 'storage', (event: StorageEvent) => {
    if (event.key !== STORAGE_KEY || event.newValue !== 'true') return
    dismissedThisVisit = true
    open.value = false
})
</script>

<template>
    <Transition name="idiom-hint">
        <div v-if="open" ref="root" class="idiom-hint">
            <span class="idiom-hint__frame" aria-hidden="true" />
            <span class="idiom-hint__leader" aria-hidden="true" />
            <span class="idiom-hint__anchor" aria-hidden="true" />
            <div
                role="dialog"
                :aria-labelledby="titleId"
                :aria-describedby="bodyId"
                class="idiom-hint__note overflow-hidden rounded-xl border border-primary/30 bg-popover text-left text-popover-foreground shadow-[0_18px_40px_-16px_hsl(var(--overlay)/0.32),0_2px_6px_-2px_hsl(var(--overlay)/0.1)]"
            >
                <div class="px-4 pb-1 pt-4">
                    <h3 :id="titleId" class="text-balance text-[0.9375rem] font-semibold leading-5 tracking-tight">
                        {{ t.dailyIdiomHintTitle }}
                    </h3>
                    <div :id="bodyId" class="mt-1.5 space-y-2 text-pretty text-sm leading-[1.4rem] text-muted-foreground">
                        <p>
                            <template v-for="(part, index) in t.dailyIdiomHintBody.split('{settings}')" :key="index"><RouterLink
                                v-if="index > 0"
                                to="/settings"
                                class="rounded-sm text-primary underline underline-offset-2 hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-popover"
                            >{{ t.dailyIdiomHintSettings }}</RouterLink>{{ part }}</template>
                        </p>
                        <p>{{ t.dailyIdiomHintTelegram }}</p>
                    </div>
                </div>
                <div class="flex justify-end px-4 pb-4 pt-3">
                    <Button size="sm" class="px-4" @click="dismiss">{{ t.dailyIdiomHintDismiss }}</Button>
                </div>
            </div>
        </div>
    </Transition>
</template>

<style scoped>
/*
 * A margin note on today's idiom: an anchor dot sits on the card border and a hairline
 * leader runs to the note. Phones and tablets stack the note above the card; wide screens
 * set it in the right margin, centred on the card. The idiom itself is never covered.
 */
.idiom-hint {
    --hint-gap: 1.5rem;
    --hint-ease: cubic-bezier(0.22, 1, 0.36, 1);
    position: absolute;
    inset: 0;
    z-index: 30;
    pointer-events: none;
}

.idiom-hint__frame {
    position: absolute;
    inset: -1px;
    border: 1px solid hsl(var(--primary) / 0.5);
    border-radius: calc(0.75rem + 1px);
    animation: idiom-hint-fade 420ms var(--hint-ease) both;
}

.idiom-hint__anchor {
    position: absolute;
    top: 0;
    left: 50%;
    width: 0.625rem;
    height: 0.625rem;
    border-radius: 9999px;
    background: hsl(var(--primary));
    box-shadow: 0 0 0 3px hsl(var(--background));
    transform: translate(-50%, -50%);
    animation: idiom-hint-anchor 360ms var(--hint-ease) both;
}

/* Two slow rings from the anchor, then stillness. */
.idiom-hint__anchor::after {
    content: '';
    position: absolute;
    inset: 0;
    border: 1.5px solid hsl(var(--primary));
    border-radius: inherit;
    opacity: 0;
    animation: idiom-hint-ring 1600ms cubic-bezier(0.16, 1, 0.3, 1) 900ms 2;
}

.idiom-hint__leader {
    position: absolute;
    bottom: 100%;
    left: 50%;
    width: 1px;
    height: var(--hint-gap);
    background: hsl(var(--primary));
    transform: translateX(-50%);
    transform-origin: bottom;
    animation: idiom-hint-draw-y 260ms var(--hint-ease) 140ms both;
}

.idiom-hint__note {
    position: absolute;
    right: 0;
    bottom: calc(100% + var(--hint-gap));
    left: 0;
    width: 100%;
    max-width: 24rem;
    margin-inline: auto;
    pointer-events: auto;
    animation: idiom-hint-rise 380ms var(--hint-ease) 300ms both;
}

@media (min-width: 1280px) {
    .idiom-hint {
        --hint-gap: 2.25rem;
    }

    .idiom-hint__anchor {
        top: 50%;
        left: 100%;
    }

    .idiom-hint__leader {
        top: 50%;
        bottom: auto;
        left: 100%;
        width: var(--hint-gap);
        height: 1px;
        transform: translateY(-50%);
        transform-origin: left;
        animation-name: idiom-hint-draw-x;
    }

    /* The card is centred on the page, so its right margin is half the viewport minus half the card. */
    .idiom-hint__note {
        top: 50%;
        right: auto;
        bottom: auto;
        left: calc(100% + var(--hint-gap));
        width: min(24rem, calc(50vw - 50% - var(--hint-gap) - 1.5rem));
        max-width: none;
        margin: 0;
        transform: translateY(-50%);
        animation-name: idiom-hint-slide;
    }
}

.idiom-hint-leave-active {
    transition: opacity 180ms ease-out;
}

.idiom-hint-leave-to {
    opacity: 0;
}

@keyframes idiom-hint-fade {
    from {
        opacity: 0;
    }
}

@keyframes idiom-hint-anchor {
    from {
        transform: translate(-50%, -50%) scale(0);
    }
    60% {
        transform: translate(-50%, -50%) scale(1.3);
    }
}

@keyframes idiom-hint-ring {
    from {
        opacity: 0.6;
        transform: scale(1);
    }
    to {
        opacity: 0;
        transform: scale(3.4);
    }
}

@keyframes idiom-hint-draw-y {
    from {
        transform: translateX(-50%) scaleY(0);
    }
}

@keyframes idiom-hint-draw-x {
    from {
        transform: translateY(-50%) scaleX(0);
    }
}

@keyframes idiom-hint-rise {
    from {
        opacity: 0;
        filter: blur(4px);
        transform: translateY(0.5rem);
    }
}

@keyframes idiom-hint-slide {
    from {
        opacity: 0;
        filter: blur(4px);
        transform: translate(-0.5rem, -50%);
    }
}

@media (prefers-reduced-motion: reduce) {
    .idiom-hint {
        animation: idiom-hint-fade 160ms ease-out both;
    }

    .idiom-hint *,
    .idiom-hint *::after {
        animation: none !important;
        transition: none !important;
    }
}
</style>
