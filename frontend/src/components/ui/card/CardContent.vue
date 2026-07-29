<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
    class?: HTMLAttributes['class']
    disabled?: boolean
}>()
</script>

<template>
    <div :class="cn('p-4 pt-0 sm:p-6 sm:pt-0', props.class)">
        <div class="disable-message flex justify-center items-center inset-0 z-10" v-if="disabled">
            <slot name="disable-reason" />
        </div>

        <div
            :inert="disabled || undefined"
            :aria-hidden="disabled ? 'true' : undefined"
            :class="{ 'disabled-area': disabled }"
        >
            <slot />
        </div>
    </div>
</template>

<style lang="postcss" scoped>
.disabled-area {
    opacity: 0.45;
    pointer-events: none;
}

.disable-message {
    position: absolute;
    padding: 6px 12px;
}
</style>
