<script setup lang="ts">
import { ToastProvider, ToastRoot, ToastTitle, ToastDescription, ToastViewport, ToastClose } from 'reka-ui'
import { reactive } from 'vue'
import { useI18n } from '@/composables/useI18n'
import { useToast } from '@/composables/useToast.ts'
import { X } from '@lucide/vue'

const { toasts, removeToast } = useToast()
const { t } = useI18n()
const closing = reactive(new Set<string>())
const finishDismissal = (id: string) => {
    removeToast(id)
    closing.delete(id)
}
</script>

<template>
    <ToastProvider>
        <slot />

        <ToastRoot
            v-for="toast in toasts"
            :key="toast.id"
            :duration="toast.duration"
            :open="!closing.has(toast.id)"
            class="toast-notification group pointer-events-auto relative flex w-full items-center justify-between space-x-4 overflow-hidden rounded-md border p-6 pr-8 shadow-lg"
            :class="{
                'border-primary/50 bg-accent text-accent-foreground': toast.variant === 'success',
                'border-destructive bg-destructive text-destructive-foreground': toast.variant === 'destructive',
                'border-border bg-popover text-popover-foreground': toast.variant === 'default' || !toast.variant,
            }"
            @update:open="(open) => !open && closing.add(toast.id)"
            @after-leave="finishDismissal(toast.id)"
        >
            <div class="grid gap-1">
                <ToastTitle v-if="toast.title" class="text-sm font-semibold">
                    {{ toast.title }}
                </ToastTitle>
                <ToastDescription v-if="toast.description" class="text-sm opacity-90">
                    {{ toast.description }}
                </ToastDescription>
            </div>
            <ToastClose as-child>
                <button
                    :aria-label="t.close"
                    class="absolute right-1 top-1 inline-flex h-11 w-11 items-center justify-center rounded-md text-current opacity-60 transition-opacity hover:opacity-100 focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-current group-hover:opacity-100"
                >
                    <X class="h-4 w-4" />
                </button>
            </ToastClose>
        </ToastRoot>

        <ToastViewport
            class="fixed top-0 z-[100] flex max-h-screen w-full flex-col-reverse gap-2 p-4 sm:bottom-0 sm:right-0 sm:top-auto sm:flex-col md:max-w-[420px]"
        />
    </ToastProvider>
</template>

<style>
.toast-notification {
    --toast-enter-y: calc(-100% - 1rem);
    --toast-exit-x: 0px;
}

.toast-notification[data-state='open'] {
    animation: toast-enter 1000ms cubic-bezier(0.25, 0.1, 0.25, 1) backwards;
}

.toast-notification[data-state='closed'] {
    animation: toast-exit 400ms ease-in forwards;
}

.toast-notification[data-swipe='move'] {
    animation: none;
    transform: translateX(var(--reka-toast-swipe-move-x));
}

.toast-notification[data-swipe='cancel'] {
    animation: none;
    transform: translateX(0);
    transition: transform 180ms ease-out;
}

.toast-notification[data-swipe='end'] {
    --toast-exit-x: var(--reka-toast-swipe-end-x);
    animation: toast-exit 400ms ease-in forwards;
}

@media (min-width: 640px) {
    .toast-notification {
        --toast-enter-y: calc(100% + 1rem);
    }
}

@keyframes toast-enter {
    from {
        opacity: 0.3;
        transform: translateY(var(--toast-enter-y));
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes toast-exit {
    from {
        opacity: 1;
        transform: translateX(var(--toast-exit-x));
    }
    to {
        opacity: 0;
        transform: translateX(calc(100% + 1rem));
    }
}

@keyframes toast-fade-in {
    from {
        opacity: 0.3;
    }
    to {
        opacity: 1;
    }
}

@keyframes toast-fade-out {
    from {
        opacity: 1;
    }
    to {
        opacity: 0;
    }
}

@media (prefers-reduced-motion: reduce) {
    .toast-notification[data-state='open'] {
        animation: toast-fade-in 120ms ease-out backwards;
    }
    .toast-notification[data-state='closed'],
    .toast-notification[data-swipe='end'] {
        animation: toast-fade-out 120ms ease-in forwards;
    }
}
</style>
