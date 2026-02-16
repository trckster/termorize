<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
    class?: HTMLAttributes['class']
    disabled?: boolean
}>()
</script>

<template>
    <div :class="cn('p-6 pt-0', props.class)">
        <div class="disable-message flex justify-center items-center inset-0 z-10" v-if="disabled">
            <slot name="disable-reason" />
        </div>

        <div :class="{ 'disabled-area': disabled }">
            <slot />
        </div>
    </div>
</template>

<style lang="postcss" scoped>
.disabled-area {
    filter: blur(2px);
    opacity: 0.6;
    pointer-events: none;
}

.disabled-area::after {
    content: '';
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(
        45deg,
        rgba(0, 0, 0, 0.1) 0,
        rgba(0, 0, 0, 0.1) 10px,
        transparent 10px,
        transparent 20px
    );
}

.disable-message {
    position: absolute;
    padding: 6px 12px;
}
</style>
