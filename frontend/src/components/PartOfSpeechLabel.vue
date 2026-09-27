<script setup lang="ts">
import { useI18n } from '@/composables/useI18n'
import { partOfSpeechLabel, type PartOfSpeech } from '@/lib/partOfSpeech'

defineProps<{ value: PartOfSpeech | null }>()
const { locale } = useI18n()
</script>

<template>
    <span :lang="locale" class="block text-xs font-normal leading-5 text-muted-foreground">{{
        partOfSpeechLabel(value, locale)
    }}</span>
</template>
