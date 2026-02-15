<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { Check, ChevronsUpDown } from 'lucide-vue-next'
import {
    ComboboxAnchor,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxItemIndicator,
    ComboboxPortal,
    ComboboxRoot,
    ComboboxTrigger,
    ComboboxViewport,
} from 'reka-ui'
import { cn } from '@/lib/utils'

interface ComboboxOption {
    value: string
    label: string
}

const props = withDefaults(
    defineProps<{
        modelValue?: string
        options: ComboboxOption[]
        placeholder?: string
        searchPlaceholder?: string
        emptyText?: string
        class?: HTMLAttributes['class']
    }>(),
    {
        modelValue: '',
        placeholder: 'Select option',
        searchPlaceholder: 'Search...',
        emptyText: 'No results found.',
    }
)

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void
}>()

const selectedValue = computed({
    get: () => props.modelValue || '',
    set: (value: string) => emit('update:modelValue', value),
})

const getLabel = (value: string) => props.options.find((option) => option.value === value)?.label || value
</script>

<template>
    <ComboboxRoot v-model="selectedValue" class="w-full">
        <ComboboxAnchor class="relative w-full">
            <ComboboxInput
                :display-value="(value) => getLabel(value as string)"
                :placeholder="selectedValue ? searchPlaceholder : placeholder"
                class="w-full px-3 py-2 pr-10 text-sm rounded-md border border-border bg-background text-foreground placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <ComboboxTrigger
                class="absolute right-0 top-0 h-full px-3 text-muted-foreground transition-colors hover:text-foreground"
            >
                <ChevronsUpDown class="h-4 w-4" />
            </ComboboxTrigger>
        </ComboboxAnchor>

        <ComboboxPortal>
            <ComboboxContent
                position="popper"
                :side-offset="6"
                :class="
                    cn(
                        'z-50 w-[var(--reka-combobox-trigger-width)] max-h-64 overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-md',
                        props.class
                    )
                "
            >
                <ComboboxViewport class="p-1">
                    <ComboboxEmpty class="py-6 text-center text-sm text-muted-foreground">{{
                        emptyText
                    }}</ComboboxEmpty>
                    <ComboboxItem
                        v-for="option in options"
                        :key="option.value"
                        :value="option.value"
                        :text-value="option.label"
                        class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none data-[disabled]:pointer-events-none data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                    >
                        <ComboboxItemIndicator class="absolute left-2 inline-flex h-4 w-4 items-center justify-center">
                            <Check class="h-4 w-4" />
                        </ComboboxItemIndicator>
                        <span class="truncate">{{ option.label }}</span>
                    </ComboboxItem>
                </ComboboxViewport>
            </ComboboxContent>
        </ComboboxPortal>
    </ComboboxRoot>
</template>
