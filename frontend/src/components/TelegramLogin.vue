<script setup lang="ts">
import { Send } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/composables/useI18n'

defineProps<{
    loading?: boolean
    insideTelegram?: boolean
}>()

const emit = defineEmits<{
    (e: 'start'): void
}>()

const { t } = useI18n()
</script>

<template>
    <Button
        type="button"
        size="lg"
        class="w-full gap-3 bg-[oklch(0.65_0.15_230)] text-[oklch(0.985_0.006_220)] hover:bg-[oklch(0.61_0.145_230)] focus-visible:ring-[oklch(0.65_0.15_230)]"
        :disabled="loading"
        @click="emit('start')"
    >
        <Send class="size-4" />
        {{
            loading
                ? insideTelegram
                    ? t.telegramLoginButtonLoading
                    : t.telegramLoginButtonRedirecting
                : insideTelegram
                  ? t.telegramLoginButtonInsideTelegram
                  : t.telegramLoginButtonVia
        }}
    </Button>
</template>
