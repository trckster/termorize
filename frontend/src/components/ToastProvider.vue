<script setup lang="ts">
import { ToastProvider, ToastRoot, ToastTitle, ToastDescription, ToastViewport } from 'reka-ui'
import { useToast } from '@/composables/useToast.ts'
import { X } from 'lucide-vue-next'

const { toasts, removeToast } = useToast()
</script>

<template>
    <ToastProvider>
        <slot />

        <ToastRoot
            v-for="toast in toasts"
            :key="toast.id"
            :duration="toast.duration"
            class="group pointer-events-auto relative flex w-full items-center justify-between space-x-4 overflow-hidden rounded-md border p-6 pr-8 shadow-lg transition-all data-[swipe=cancel]:translate-x-0 data-[swipe=end]:translate-x-[var(--reka-toast-swipe-end-x)] data-[swipe=move]:translate-x-[var(--reka-toast-swipe-move-x)] data-[state=open]:animate-in data-[state=closed]:animate-out data-[swipe=end]:animate-out data-[state=closed]:fade-out-80 data-[state=closed]:slide-out-to-right-full data-[state=open]:slide-in-from-top-full data-[state=open]:sm:slide-in-from-bottom-full"
            :class="{
                'border-green-500 bg-green-50 text-green-900 dark:border-green-500 dark:bg-green-950 dark:text-green-100':
                    toast.variant === 'success',
                'border-destructive bg-destructive text-destructive-foreground': toast.variant === 'destructive',
                'border-border bg-background text-foreground': toast.variant === 'default' || !toast.variant,
            }"
            @update:open="(open) => !open && removeToast(toast.id)"
        >
            <div class="grid gap-1">
                <ToastTitle v-if="toast.title" class="text-sm font-semibold">
                    {{ toast.title }}
                </ToastTitle>
                <ToastDescription v-if="toast.description" class="text-sm opacity-90">
                    {{ toast.description }}
                </ToastDescription>
            </div>
            <button
                class="absolute right-2 top-2 rounded-md p-1 text-foreground/50 opacity-0 transition-opacity hover:text-foreground focus:opacity-100 focus:outline-none focus:ring-2 group-hover:opacity-100"
                @click="removeToast(toast.id)"
            >
                <X class="h-4 w-4" />
            </button>
        </ToastRoot>

        <ToastViewport
            class="fixed top-0 z-[100] flex max-h-screen w-full flex-col-reverse p-4 sm:bottom-0 sm:right-0 sm:top-auto sm:flex-col md:max-w-[420px]"
        />
    </ToastProvider>
</template>
