<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ArrowRight, BookOpen, Globe2, Loader2, UserRound } from 'lucide-vue-next'
import { useRoute } from 'vue-router'
import { collectionsApi, type PublicCollectionDetail } from '@/api/collections'
import PronunciationButton from '@/components/PronunciationButton.vue'
import { Button } from '@/components/ui/button'
import TelegramLogin from '@/components/TelegramLogin.vue'
import { useI18n } from '@/composables/useI18n'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { formatNumber } from '@/lib/utils'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'
import { rememberPostAuthPath } from '@/lib/postAuthRedirect'

const props = defineProps<{
    mode: 'direct' | 'join'
    identifier: string
}>()

const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const { t, saves } = useI18n()

const collection = ref<PublicCollectionDetail | null>(null)
const isLoading = ref(true)
const isTelegramLoading = ref(false)
const isGuestLoading = ref(false)
const errorMessage = ref('')
const loginError = ref('')
const isInsideTelegram = isTelegramWebApp()

const originalTitle = document.title
const descriptionElement = document.querySelector<HTMLMetaElement>('meta[name="description"]')
const originalDescription = descriptionElement?.content || ''

const isLoginBusy = computed(() => isTelegramLoading.value || isGuestLoading.value)
const actionTitle = computed(() =>
    props.mode === 'join' ? t.value.collectionPreviewJoinTitle : t.value.collectionPreviewUseTitle
)
const actionDescription = computed(() =>
    props.mode === 'join' ? t.value.collectionPreviewJoinDescription : t.value.collectionPreviewUseDescription
)

const getLanguageName = (code: string) =>
    settingsStore.languageOptions.find((language) => language.code === code)?.name || code.toUpperCase()

const updateDocumentMetadata = (value: PublicCollectionDetail | null) => {
    if (!value) return

    document.title = `${value.title} · Termorize`
    if (descriptionElement) {
        descriptionElement.content = t.value.collectionPreviewMetaDescription
            .replace('{count}', formatNumber(value.translation_count))
            .replace('{title}', value.title)
    }
}

const fetchCollection = async () => {
    isLoading.value = true
    errorMessage.value = ''
    collection.value = null

    try {
        const result =
            props.mode === 'join'
                ? await collectionsApi.getPublicCollectionByShareIdentifier(props.identifier)
                : await collectionsApi.getPublicCollection(props.identifier)
        collection.value = result
        updateDocumentMetadata(result)
    } catch {
        errorMessage.value = t.value.collectionPreviewUnavailableDescription
    } finally {
        isLoading.value = false
    }
}

const startTelegramLogin = async () => {
    if (isLoginBusy.value) return

    loginError.value = ''
    isTelegramLoading.value = true

    try {
        const initData = getTelegramWebAppInitData()
        if (initData) {
            await authStore.completeTelegramLogin({ init_data: initData })
            return
        }

        rememberPostAuthPath(route.fullPath)
        const authURL = await authStore.startTelegramLogin()
        window.location.assign(authURL)
    } catch (error) {
        loginError.value = getErrorMessage(error, t.value.loginStartError)
        isTelegramLoading.value = false
    }
}

const continueAsGuest = async () => {
    if (isLoginBusy.value) return

    loginError.value = ''
    isGuestLoading.value = true

    try {
        await authStore.continueAsGuest()
    } catch (error) {
        loginError.value = getErrorMessage(error, t.value.collectionPreviewGuestError)
        isGuestLoading.value = false
    }
}

const getErrorMessage = (error: unknown, fallback: string): string => {
    if (error instanceof Error) return error.message
    if (typeof error === 'object' && error !== null && 'body' in error) {
        const body = (error as { body?: { error?: string; details?: string; message?: string } }).body
        return body?.details || body?.error || body?.message || fallback
    }
    return fallback
}

watch(() => [props.mode, props.identifier], fetchCollection, { immediate: true })

onBeforeUnmount(() => {
    document.title = originalTitle
    if (descriptionElement) {
        descriptionElement.content = originalDescription
    }
})
</script>

<template>
    <div class="min-h-screen bg-background text-foreground">
        <header class="pt-safe border-b border-border bg-background">
            <div class="mx-auto flex min-h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
                <router-link
                    to="/"
                    class="-ml-2 inline-flex min-h-11 items-center gap-2 rounded-md px-2 text-sm font-semibold tracking-tight transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                    <img src="/favicon.svg" alt="" class="h-7 w-7" />
                    <span>Termorize</span>
                </router-link>
                <span
                    class="rounded-full border border-border bg-muted/50 px-3 py-1 text-xs font-medium text-muted-foreground"
                >
                    {{ t.collectionPreviewPublished }}
                </span>
            </div>
        </header>

        <main class="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 sm:py-12">
            <div v-if="isLoading" class="flex min-h-[60vh] flex-col items-center justify-center gap-3" role="status">
                <Loader2 class="h-6 w-6 animate-spin text-primary motion-reduce:animate-none" />
                <p class="text-sm text-muted-foreground">{{ t.collectionPreviewLoading }}</p>
            </div>

            <section
                v-else-if="!collection"
                class="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center text-center"
            >
                <div class="flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                    <BookOpen class="h-6 w-6" />
                </div>
                <h1 class="mt-5 text-2xl font-semibold tracking-tight sm:text-3xl">
                    {{ t.collectionPreviewUnavailableTitle }}
                </h1>
                <p class="mt-3 max-w-md text-sm leading-6 text-muted-foreground">{{ errorMessage }}</p>
                <Button as-child variant="outline" class="mt-6">
                    <router-link to="/">{{ t.collectionPreviewBack }}</router-link>
                </Button>
            </section>

            <article v-else>
                <section
                    class="grid gap-8 border-b border-border pb-0 lg:grid-cols-[minmax(0,1fr)_22rem] lg:gap-14 lg:pb-10"
                >
                    <div class="min-w-0">
                        <h1 class="max-w-3xl break-words text-3xl font-semibold tracking-[-0.03em] sm:text-5xl">
                            {{ collection.title }}
                        </h1>

                        <div class="mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
                            <span class="inline-flex items-center gap-2">
                                <Globe2 v-if="collection.is_admin" class="h-4 w-4 text-primary" />
                                <UserRound v-else class="h-4 w-4 text-primary" />
                                {{ collection.is_admin ? t.collectionsGlobalBadge : t.collectionPreviewShared }}
                            </span>
                            <span v-if="collection.owner_username">@{{ collection.owner_username }}</span>
                            <span>
                                {{ formatNumber(collection.translation_count) }}
                                {{ t.collectionTranslationsLabel }}
                            </span>
                            <span v-if="collection.user_add_count > 0">{{ saves(collection.user_add_count) }}</span>
                        </div>

                        <div v-if="collection.languages.length > 0" class="mt-6 flex flex-wrap gap-2">
                            <span
                                v-for="language in collection.languages"
                                :key="language"
                                class="inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1.5 text-sm"
                            >
                                <span role="img" :aria-label="getLanguageName(language)">{{
                                    settingsStore.getFlag(language)
                                }}</span>
                                {{ getLanguageName(language) }}
                            </span>
                        </div>
                    </div>

                    <aside class="border-t border-border py-6 lg:border-t-0 lg:border-l lg:py-1 lg:pl-8">
                        <h2 class="text-lg font-semibold">{{ actionTitle }}</h2>
                        <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ actionDescription }}</p>
                        <div class="mt-5 space-y-2">
                            <TelegramLogin
                                :loading="isTelegramLoading"
                                :inside-telegram="isInsideTelegram"
                                @start="startTelegramLogin"
                            />
                            <Button
                                type="button"
                                variant="ghost"
                                class="w-full"
                                :disabled="isLoginBusy"
                                @click="continueAsGuest"
                            >
                                <Loader2
                                    v-if="isGuestLoading"
                                    class="h-4 w-4 animate-spin motion-reduce:animate-none"
                                />
                                <template v-else>
                                    {{ t.collectionPreviewTryGuest }}
                                    <ArrowRight class="h-4 w-4" />
                                </template>
                            </Button>
                        </div>
                        <p v-if="loginError" class="mt-3 text-sm text-destructive" role="alert">{{ loginError }}</p>
                    </aside>
                </section>

                <section class="pt-9 sm:pt-11">
                    <div class="flex flex-wrap items-end justify-between gap-3">
                        <div>
                            <h2 class="text-xl font-semibold tracking-tight sm:text-2xl">
                                {{ t.collectionPreviewWordsTitle }}
                            </h2>
                            <p class="mt-2 text-sm text-muted-foreground">{{ t.collectionPreviewWordsDescription }}</p>
                        </div>
                        <span class="text-sm tabular-nums text-muted-foreground">
                            {{ formatNumber(collection.translation_count) }}
                        </span>
                    </div>

                    <ol
                        v-if="collection.translations.length > 0"
                        class="mt-5 divide-y divide-border border-y border-border"
                    >
                        <li
                            v-for="(item, index) in collection.translations"
                            :key="item.id"
                            class="grid min-h-20 grid-cols-[2rem_minmax(0,1fr)] items-center gap-x-3 gap-y-2 py-4 sm:grid-cols-[3rem_minmax(0,1fr)_3rem_minmax(0,1fr)] sm:gap-5"
                        >
                            <span class="text-xs tabular-nums text-muted-foreground">{{ index + 1 }}</span>
                            <div class="min-w-0">
                                <div class="flex min-w-0 items-center gap-2">
                                    <p class="min-w-0 flex-1 break-words text-base font-medium sm:text-lg">
                                        {{ item.original.word }}
                                    </p>
                                    <PronunciationButton
                                        :word-id="item.original.id"
                                        :word="item.original.word"
                                        :listen-label="t.pronunciationListen"
                                        :pause-label="t.pronunciationPause"
                                        :loading-label="t.pronunciationLoading"
                                        :error-label="t.pronunciationError"
                                    />
                                </div>
                                <p class="mt-1 text-xs uppercase tracking-wide text-muted-foreground">
                                    {{ getLanguageName(item.original.language) }}
                                </p>
                            </div>
                            <ArrowRight
                                class="h-4 w-4 rotate-90 justify-self-center text-muted-foreground sm:rotate-0"
                                aria-hidden="true"
                            />
                            <div class="min-w-0">
                                <div class="flex min-w-0 items-center gap-2">
                                    <p class="min-w-0 flex-1 break-words text-base font-medium sm:text-lg">
                                        {{ item.translation.word }}
                                    </p>
                                    <PronunciationButton
                                        :word-id="item.translation.id"
                                        :word="item.translation.word"
                                        :listen-label="t.pronunciationListen"
                                        :pause-label="t.pronunciationPause"
                                        :loading-label="t.pronunciationLoading"
                                        :error-label="t.pronunciationError"
                                    />
                                </div>
                                <p class="mt-1 text-xs uppercase tracking-wide text-muted-foreground">
                                    {{ getLanguageName(item.translation.language) }}
                                </p>
                            </div>
                        </li>
                    </ol>

                    <div v-else class="mt-5 border-y border-border py-12 text-center text-sm text-muted-foreground">
                        {{ t.collectionDetailEmpty }}
                    </div>
                </section>
            </article>
        </main>

        <footer class="border-t border-border">
            <div
                class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-6 text-xs text-muted-foreground sm:px-6"
            >
                <span>Termorize</span>
                <span>{{ t.collectionPreviewFooter }}</span>
            </div>
        </footer>
    </div>
</template>
