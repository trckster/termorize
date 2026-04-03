<script setup lang="ts">
interface Props {
    modelValue: boolean
    disabled?: boolean
    label?: string
    labelledby?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
    'update:modelValue': [value: boolean]
}>()

const handleClick = () => {
    if (props.disabled) return
    emit('update:modelValue', !props.modelValue)
}
</script>

<template>
    <button
        type="button"
        class="relative inline-flex h-11 w-12 items-center justify-center rounded-md focus:outline-none focus:ring-2 focus:ring-primary disabled:cursor-not-allowed disabled:opacity-50"
        role="switch"
        :aria-checked="modelValue"
        :aria-label="label"
        :aria-labelledby="labelledby"
        :disabled="disabled"
        @click="handleClick"
    >
        <span
            class="absolute h-6 w-11 rounded-full transition-colors"
            :class="modelValue ? 'bg-primary' : 'bg-muted'"
        ></span>
        <span
            class="absolute h-5 w-5 rounded-full bg-background transition-transform"
            :class="modelValue ? 'translate-x-2.5' : '-translate-x-2.5'"
        ></span>
    </button>
</template>
