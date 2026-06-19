<script setup lang="ts">
import { computed } from 'vue'
import { Check, Moon, Sun } from 'lucide-vue-next'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useI18n } from '@/composables/useI18n'
import { useTheme, type PaletteId } from '@/composables/useTheme'

const { t } = useI18n()
const { palette, isDark, options, setPalette, setDark } = useTheme()

const descriptions = computed<Record<PaletteId, string>>(() => ({
    evergreen: t.value.themeDescEvergreen,
    sage: t.value.themeDescSage,
    emerald: t.value.themeDescEmerald,
}))
</script>

<template>
    <Card>
        <CardHeader>
            <CardTitle>{{ t.settingsAppearanceTitle }}</CardTitle>
            <CardDescription>{{ t.settingsAppearanceDescription }}</CardDescription>
        </CardHeader>
        <CardContent class="space-y-8">
            <!-- Palette -->
            <fieldset class="space-y-3">
                <legend class="text-sm font-semibold text-foreground">{{ t.settingsAppearanceThemeLabel }}</legend>
                <div class="grid gap-3 sm:grid-cols-3">
                    <button
                        v-for="option in options"
                        :key="option.id"
                        type="button"
                        :aria-pressed="palette === option.id"
                        class="group relative flex flex-col gap-3 rounded-xl border bg-card p-4 text-left transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
                        :class="palette === option.id ? 'border-primary ring-1 ring-primary' : 'border-border hover:bg-accent/40'"
                        @click="setPalette(option.id)"
                    >
                        <span
                            v-if="palette === option.id"
                            class="absolute right-3 top-3 inline-flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground"
                        >
                            <Check class="h-3 w-3" />
                        </span>
                        <span class="flex items-center gap-1.5" aria-hidden="true">
                            <span
                                v-for="(color, i) in option.preview"
                                :key="i"
                                class="h-6 w-6 rounded-full border border-black/5 first:h-7 first:w-7"
                                :style="{ backgroundColor: color }"
                            />
                        </span>
                        <span>
                            <span class="block text-sm font-semibold text-foreground">{{ option.name }}</span>
                            <span class="mt-0.5 block text-xs leading-snug text-muted-foreground">
                                {{ descriptions[option.id] }}
                            </span>
                        </span>
                    </button>
                </div>
                <p class="text-xs text-muted-foreground">{{ t.settingsAppearanceThemeExplanation }}</p>
            </fieldset>

            <!-- Light / Dark -->
            <fieldset class="space-y-3">
                <legend class="text-sm font-semibold text-foreground">{{ t.settingsAppearanceModeLabel }}</legend>
                <div class="inline-flex rounded-lg border border-border bg-background p-1">
                    <button
                        type="button"
                        :aria-pressed="!isDark"
                        class="inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        :class="!isDark ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
                        @click="setDark(false)"
                    >
                        <Sun class="h-4 w-4" />
                        {{ t.settingsAppearanceModeLight }}
                    </button>
                    <button
                        type="button"
                        :aria-pressed="isDark"
                        class="inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        :class="isDark ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'"
                        @click="setDark(true)"
                    >
                        <Moon class="h-4 w-4" />
                        {{ t.settingsAppearanceModeDark }}
                    </button>
                </div>
                <p class="text-xs text-muted-foreground">{{ t.settingsAppearanceModeExplanation }}</p>
            </fieldset>
        </CardContent>
    </Card>
</template>
