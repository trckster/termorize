<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Loader2 } from '@lucide/vue'
import { vocabularyApi, type VocabularyShare } from '@/api/vocabulary'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from '@/composables/useI18n'
import { useSettingsStore } from '@/stores/settings'
import { Button } from '@/components/ui/button'
import PronunciationButton from '@/components/PronunciationButton.vue'
import PartOfSpeechLabel from '@/components/PartOfSpeechLabel.vue'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { rememberPostAuthPath, clearPostAuthPath } from '@/lib/postAuthRedirect'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'

const route = useRoute()
const auth = useAuthStore()
const settings = useSettingsStore()
const { t } = useI18n()
const share = ref<VocabularyShare | null>(null)
const loading = ref(true)
const busy = ref(false)
const loginBusy = ref(false)
const error = ref('')
const saved = ref(false)
const originalTitle = document.title
let request = 0

async function load() {
    const current = ++request
    loading.value = true
    share.value = null
    error.value = ''
    saved.value = false
    try {
        const result = await vocabularyApi.getSharedVocabulary(String(route.params.token))
        if (current !== request) return
        share.value = result
        document.title = `${result.translation.original.word} — ${result.translation.translation.word} · Termorize`
    } catch {
        if (current === request) error.value = t.value.vocabularyShareUnavailable
    } finally {
        if (current === request) loading.value = false
    }
}

async function save() {
    if (!share.value || busy.value || loginBusy.value) return
    busy.value = true
    error.value = ''
    try {
        if (!auth.isAuthenticated) await auth.continueAsGuest()
        await vocabularyApi.saveSharedVocabulary(share.value.translation.id)
        saved.value = true
    } catch (err) {
        if ((err as { status?: number }).status === 409) saved.value = true
        else error.value = t.value.vocabularyShareSaveError
    } finally {
        busy.value = false
    }
}

async function login() {
    if (busy.value || loginBusy.value) return
    loginBusy.value = true
    error.value = ''
    try {
        const initData = getTelegramWebAppInitData()
        if (initData) {
            await auth.completeTelegramLogin({ init_data: initData })
            loginBusy.value = false
        } else {
            rememberPostAuthPath(route.fullPath)
            window.location.assign(await auth.startTelegramLogin())
        }
    } catch {
        clearPostAuthPath()
        error.value = t.value.loginStartError
        loginBusy.value = false
    }
}

watch(() => route.params.token, load, { immediate: true })
onBeforeUnmount(() => {
    request++
    document.title = originalTitle
})
</script>

<template>
    <div class="min-h-screen bg-background text-foreground">
        <header class="pt-safe border-b border-border">
            <div class="mx-auto flex min-h-16 max-w-3xl items-center px-4 sm:px-6">
                <RouterLink
                    to="/"
                    class="inline-flex min-h-11 items-center gap-2 rounded-md font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                    <img src="/favicon.svg" alt="" class="h-7 w-7" />Termorize
                </RouterLink>
            </div>
        </header>
        <main class="mx-auto max-w-xl px-4 py-10 sm:px-6 sm:py-16">
            <div v-if="loading" class="flex items-center gap-3 text-muted-foreground" role="status">
                <Loader2 class="h-5 w-5 animate-spin motion-reduce:animate-none" />{{ t.vocabularyShareLoading }}
            </div>
            <template v-else-if="share">
                <h1 class="mb-8 text-xl font-semibold">{{ t.vocabularyShareWordTitle }}</h1>
                <div class="space-y-6">
                    <div
                        v-for="word in [share.translation.original, share.translation.translation]"
                        :key="word.id"
                        class="flex items-start justify-between gap-4"
                    >
                        <div class="min-w-0 space-y-2">
                            <p class="text-sm text-muted-foreground">{{ settings.getLanguageName(word.language) }}</p>
                            <p :lang="word.language" class="break-words text-3xl font-semibold leading-tight">
                                {{ word.word }}
                            </p>
                            <PartOfSpeechLabel :value="word.part_of_speech" />
                        </div>
                        <PronunciationButton
                            :word-id="word.id"
                            :word="word.word"
                            :listen-label="t.pronunciationListen"
                            :pause-label="t.pronunciationPause"
                            :loading-label="t.pronunciationLoading"
                            :error-label="t.pronunciationError"
                        />
                    </div>
                </div>
                <div class="mt-10 space-y-4 border-t border-border pt-6">
                    <template v-if="saved">
                        <p role="status" class="text-sm text-foreground">{{ t.vocabularyShareSaved }}</p>
                        <Button as-child class="min-h-11"
                            ><RouterLink to="/vocabulary">{{ t.vocabularyTitle }}</RouterLink></Button
                        >
                    </template>
                    <template v-else>
                        <Button class="min-h-11 w-full" :disabled="busy || loginBusy" @click="save">
                            <Loader2 v-if="busy" class="mr-2 h-4 w-4 animate-spin motion-reduce:animate-none" />{{
                                t.vocabularyShareSave
                            }}
                        </Button>
                        <template v-if="!auth.isAuthenticated">
                            <p class="text-sm text-muted-foreground">{{ t.vocabularyShareGuestHint }}</p>
                            <TelegramLogin :loading="loginBusy" :inside-telegram="isTelegramWebApp()" @start="login" />
                        </template>
                    </template>
                </div>
            </template>
            <template v-else>
                <h1 class="text-xl font-semibold">{{ t.vocabularyShareUnavailableTitle }}</h1>
                <Button variant="outline" class="mt-4 min-h-11" @click="load">{{ t.commonRetry }}</Button>
            </template>
            <p v-if="error" class="mt-4 text-sm text-destructive" role="alert">{{ error }}</p>
        </main>
    </div>
</template>
