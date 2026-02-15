<script setup lang="ts">
import { computed } from 'vue'
import { useSettingsStore } from '@/stores/settings.ts'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const props = withDefaults(
    defineProps<{
        modelValue?: string
        placeholder?: string
    }>(),
    {
        modelValue: '',
        placeholder: 'Select language',
    }
)

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void
}>()

const settingsStore = useSettingsStore()

const selectedValue = computed({
    get: () => props.modelValue || '',
    set: (value: string) => emit('update:modelValue', value),
})

const options = computed(() =>
    settingsStore.languageOptions.map((option) => ({
        value: option.code,
        label: `${option.emoji} ${option.name}`,
    }))
)
</script>

<template>
    <Select v-model="selectedValue">
        <SelectTrigger>
            <SelectValue :placeholder="placeholder" />
        </SelectTrigger>
        <SelectContent>
            <SelectItem v-for="option in options" :key="option.value" :value="option.value">{{
                option.label
            }}</SelectItem>
        </SelectContent>
    </Select>
</template>
