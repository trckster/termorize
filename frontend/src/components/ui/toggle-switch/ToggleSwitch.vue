<script setup lang="ts">
interface Props {
    modelValue: boolean
    disabled?: boolean
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
        class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50"
        :class="modelValue ? 'bg-primary' : 'bg-muted'"
        role="switch"
        :aria-checked="modelValue"
        :disabled="disabled"
        @click="handleClick"
    >
        <span
            class="inline-block h-5 w-5 transform rounded-full bg-background transition-transform"
            :class="modelValue ? 'translate-x-5' : 'translate-x-1'"
        />
    </button>
</template>
