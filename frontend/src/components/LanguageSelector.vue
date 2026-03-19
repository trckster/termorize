<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
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
import { useSettingsStore } from '@/stores/settings.ts'
import { cn } from '@/lib/utils.ts'

const props = withDefaults(
    defineProps<{
        modelValue?: string
        placeholder?: string
        allowedValues?: string[]
        disabledValues?: string[]
        disabled?: boolean
        ariaLabel?: string
    }>(),
    {
        modelValue: '',
        placeholder: 'Select language',
        allowedValues: () => [],
        disabledValues: () => [],
        disabled: false,
        ariaLabel: undefined,
    }
)

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void
}>()

const settingsStore = useSettingsStore()
const isOpen = ref(false)
const rootRef = ref<HTMLElement | null>(null)

const selectedValue = computed({
    get: () => props.modelValue || '',
    set: (value: string) => emit('update:modelValue', value),
})

const selectedLabel = computed(() => {
    const selectedOption = settingsStore.languageOptions.find((option) => option.code === selectedValue.value)

    return selectedOption ? `${selectedOption.emoji} ${selectedOption.name}` : ''
})

const getSelectedLabel = () => selectedLabel.value

const options = computed(() =>
    settingsStore.languageOptions
        .filter((option) => props.allowedValues.length === 0 || props.allowedValues.includes(option.code))
        .map((option) => ({
            value: option.code,
            label: `${option.emoji} ${option.name}`,
            disabled: props.disabledValues.includes(option.code),
        }))
)

const focusInput = async () => {
    isOpen.value = true
    await nextTick()
    const inputElement = rootRef.value?.querySelector('input')
    inputElement?.focus()
    inputElement?.select()
}

const blurInput = async () => {
    await nextTick()

    const inputElement = rootRef.value?.querySelector('input')

    if (!(inputElement instanceof HTMLInputElement)) return

    const cursorPosition = inputElement.value.length

    inputElement.setSelectionRange(cursorPosition, cursorPosition)
    inputElement.blur()
}

const handleUnmountAutoFocus = (event: Event) => {
    event.preventDefault()
    void blurInput()
}

watch(isOpen, async (open) => {
    if (open) return

    await blurInput()
})

defineExpose({
    focusInput,
})
</script>

<template>
    <div ref="rootRef" class="w-full">
        <ComboboxRoot
            v-model="selectedValue"
            v-model:open="isOpen"
            :open-on-click="!disabled"
            class="w-full"
            :disabled="disabled"
        >
            <ComboboxAnchor class="relative w-full">
                <ComboboxInput
                    :display-value="getSelectedLabel"
                    :placeholder="placeholder"
                    :disabled="disabled"
                    :aria-label="ariaLabel"
                    class="w-full rounded-md border border-input bg-background px-3 py-2 pr-10 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                />
                <ComboboxTrigger
                    :disabled="disabled"
                    class="absolute right-0 top-0 flex h-full px-3 items-center text-muted-foreground transition-colors hover:text-foreground"
                >
                    <ChevronsUpDown class="h-4 w-4" />
                </ComboboxTrigger>
            </ComboboxAnchor>

            <ComboboxPortal>
                <ComboboxContent
                    position="popper"
                    :side-offset="6"
                    @unmount-auto-focus="handleUnmountAutoFocus"
                    :class="
                        cn(
                            'z-50 w-[var(--reka-combobox-trigger-width)] max-h-64 overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-md'
                        )
                    "
                >
                    <ComboboxViewport class="p-1">
                        <ComboboxEmpty class="py-6 text-center text-sm text-muted-foreground">
                            No languages found.
                        </ComboboxEmpty>
                        <ComboboxItem
                            v-for="option in options"
                            :key="option.value"
                            :value="option.value"
                            :text-value="option.label"
                            :disabled="option.disabled"
                            class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
                        >
                            <ComboboxItemIndicator
                                class="absolute left-2 inline-flex h-4 w-4 items-center justify-center"
                            >
                                <Check class="h-4 w-4" />
                            </ComboboxItemIndicator>
                            <span class="truncate">{{ option.label }}</span>
                        </ComboboxItem>
                    </ComboboxViewport>
                </ComboboxContent>
            </ComboboxPortal>
        </ComboboxRoot>
    </div>
</template>
