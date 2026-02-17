<script setup lang="ts">
interface Props {
    modelValue: number
}

defineProps<Props>()

const emit = defineEmits<{
    'update:modelValue': [value: number]
}>()

const handleInput = (event: Event) => {
    emit('update:modelValue', Number((event.target as HTMLInputElement).value))
}
</script>

<template>
    <input
        type="number"
        :value="modelValue"
        class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
        v-bind="$attrs"
        @input="handleInput"
    />
</template>

<style scoped>
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
}

input[type='number'] {
    appearance: textfield;
    -moz-appearance: textfield;
}
</style>
