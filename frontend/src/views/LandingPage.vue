<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'

const router = useRouter()
const authStore = useAuthStore()

const isTelegramLoading = ref(false)
const isGuestLoading = ref(false)
const isLoading = computed(() => isTelegramLoading.value || isGuestLoading.value)
const error = ref<string | null>(null)
const isInsideTelegram = isTelegramWebApp()
const menuOpen = ref(false)

const originalTitle = document.title

onMounted(() => {
    document.title = 'Termorize: Vocabulary trainer with Telegram support'

    // demo Run button
    const runBtn = document.querySelector('.run-demo') as HTMLElement | null
    if (runBtn) {
        const label = runBtn.querySelector('.run-label') as HTMLElement | null
        const quips = [
            'Demo only \uD83D\uDE43',
            'Loading...',
            'Okay fine\u2026 vrrr\u2026 nope.',
            'Log in to actually try \u2192',
        ]
        let q = 0
        let fessed = false
        const shake = () => {
            runBtn.classList.add('nudge')
            setTimeout(() => runBtn.classList.remove('nudge'), 420)
        }
        runBtn.addEventListener('click', () => {
            if (!label) return
            if (!fessed) {
                fessed = true
                label.textContent = 'Running\u2026'
                runBtn.classList.add('is-busy')
                setTimeout(() => {
                    runBtn.classList.remove('is-busy')
                    label.textContent = quips[0] ?? ''
                    q = 1
                    shake()
                }, 750)
                return
            }
            label.textContent = quips[q] ?? ''
            q = (q + 1) % quips.length
            shake()
        })
    }

    // Telegram auto-login inside WebApp
    const initData = getTelegramWebAppInitData()
    if (initData) {
        void startTelegramLogin()
    }
})

onBeforeUnmount(() => {
    document.title = originalTitle
})

const startTelegramLogin = async () => {
    if (isLoading.value) return
    try {
        error.value = null
        isTelegramLoading.value = true

        const initData = getTelegramWebAppInitData()
        if (initData) {
            await authStore.completeTelegramLogin({ init_data: initData })
            await router.replace({ name: 'translation' })
            return
        }

        const authUrl = await authStore.startTelegramLogin()
        window.location.assign(authUrl)
    } catch (err) {
        error.value = getErrorMessage(err)
        isTelegramLoading.value = false
    }
}

const tryWithoutRegistration = async () => {
    if (isLoading.value) return

    try {
        error.value = null
        isGuestLoading.value = true
        await authStore.continueAsGuest()
        await router.replace({ name: 'translation' })
    } catch (err) {
        error.value = getErrorMessage(err, 'Could not create a temporary account. Please try again.')
        isGuestLoading.value = false
    }
}

function getErrorMessage(err: unknown, fallback = 'Login failed. Please try again.'): string {
    if (err instanceof Error) {
        return err.message
    }
    if (typeof err === 'object' && err !== null && 'body' in err) {
        const body = (err as { body?: { error?: string; details?: string; message?: string } }).body
        return body?.details || body?.error || body?.message || fallback
    }
    return fallback
}
</script>

<template>
    <div class="landing-view">
        <div class="glow"></div>

        <svg width="0" height="0" style="position: absolute">
            <defs>
                <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0" stop-color="oklch(0.68 0.16 152)" />
                    <stop offset="1" stop-color="oklch(0.46 0.13 152)" />
                </linearGradient>
            </defs>
        </svg>

        <nav class="pt-safe">
            <div class="wrap nav-inner">
                <a class="brand" href="#top">
                    <span class="mark">
                        <svg viewBox="0 0 64 64" fill="none">
                            <rect x="2" y="2" width="60" height="60" rx="17" fill="url(#brandGrad)" />
                            <g stroke="#06140c" stroke-width="6.2" stroke-linecap="round">
                                <g opacity="0.22" transform="translate(7 7)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <g opacity="0.45" transform="translate(3.5 3.5)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <path d="M20 23h24M32 23v23" />
                            </g>
                        </svg>
                    </span>
                    Termorize
                </a>
                <div id="landing-navigation" class="nav-links" :class="{ open: menuOpen }">
                    <a href="#features" @click="menuOpen = false">Features</a>
                    <a href="#showcase" @click="menuOpen = false">Showcase</a>
                    <a href="#how" @click="menuOpen = false">How it works</a>
                    <a href="#telegram" @click="menuOpen = false">Telegram</a>
                </div>
                <button
                    class="nav-toggle"
                    aria-label="Toggle menu"
                    aria-controls="landing-navigation"
                    :aria-expanded="menuOpen"
                    @click="menuOpen = !menuOpen"
                    @keydown.esc="menuOpen = false"
                >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
                        <path v-if="!menuOpen" d="M4 6h16M4 12h16M4 18h16" />
                        <path v-else d="M6 6l12 12M6 18L18 6" />
                    </svg>
                </button>
                <div class="nav-cta">
                    <button class="btn btn-tg" :disabled="isLoading" @click="startTelegramLogin">
                        <svg viewBox="0 0 24 24" fill="none">
                            <path
                                d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                fill="currentColor"
                            />
                        </svg>
                        <span v-if="isTelegramLoading">{{
                            isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                        }}</span>
                        <span v-else>Continue via Telegram</span>
                    </button>
                </div>
            </div>
        </nav>

        <header id="top" class="hero">
            <div class="wrap hero-grid">
                <div class="hero-copy reveal">
                    <div class="eyebrow">
                        <span class="dot"></span>
                        VOCABULARY TRAINER &middot; TELEGRAM SUPPORT
                    </div>
                    <h1 class="hero-title">
                        Learn words faster on the <span class="accent">web</span> and in Telegram.
                    </h1>
                    <p class="hero-sub">
                        Termorize brings translation, your personal vocabulary, practice and statistics together in one
                        site, with a Telegram bot that keeps you learning on the go.
                    </p>
                    <div class="hero-actions">
                        <div class="guest-action">
                            <button
                                type="button"
                                class="btn btn-green btn-lg"
                                :disabled="isLoading"
                                aria-describedby="guest-account-note"
                                @click="tryWithoutRegistration"
                            >
                                <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                                    <path d="M8 5v14l11-7z" />
                                </svg>
                                <span v-if="isGuestLoading">Preparing your account...</span>
                                <span v-else>Try without registration</span>
                            </button>
                            <p id="guest-account-note" class="guest-note">
                                No signup. Start with 50 example words. Temporary accounts are deleted after 7 days.
                            </p>
                        </div>
                        <button class="btn btn-tg btn-lg" :disabled="isLoading" @click="startTelegramLogin">
                            <svg viewBox="0 0 24 24" fill="none">
                                <path
                                    d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                    fill="currentColor"
                                />
                            </svg>
                            <span v-if="isTelegramLoading">{{
                                isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                            }}</span>
                            <span v-else>Continue via Telegram</span>
                        </button>
                    </div>
                    <div v-if="error" class="hero-error">{{ error }}</div>
                    <div class="hero-meta">
                        <span class="flags">
                            <span class="flag-set">🇬🇧 🇩🇪 🇪🇸 🇮🇹 🇷🇺</span>
                            <span class="flag-more">+&nbsp;more</span>
                        </span>
                        <span>One account &middot; the bot and the site, always in sync</span>
                    </div>
                </div>

                <div class="hero-mock reveal">
                    <div class="mock">
                        <div class="mock-top"><span></span><span></span><span></span></div>
                        <div class="mock-panes">
                            <div>
                                <div class="pane-label">From <span class="lang-pill">🇬🇧 English</span></div>
                                <div class="pane-box filled"><span class="typed">resilience</span></div>
                            </div>
                            <div class="swap-btn">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M7 4 3 8l4 4M3 8h14M17 20l4-4-4-4M21 16H7" />
                                </svg>
                            </div>
                            <div>
                                <div class="pane-label">To <span class="lang-pill">🇷🇺 Russian</span></div>
                                <div class="pane-box filled"><span class="typed">устойчивость</span></div>
                            </div>
                        </div>
                        <div class="mock-shortcuts">
                            <div class="sc-row"><span>Save to vocabulary</span><kbd>Ctrl + S</kbd></div>
                            <div class="sc-row"><span>Swap languages</span><kbd>Ctrl + Shift + S</kbd></div>
                        </div>
                    </div>
                    <div class="mock-run">
                        <h4>Practice your vocabulary</h4>
                        <p>Review saved words with built-in exercises.</p>
                        <button type="button" class="btn btn-green run-demo" style="width: 100%">
                            <svg viewBox="0 0 24 24" fill="currentColor">
                                <path d="M8 5v14l11-7z" />
                            </svg>
                            <span class="run-label">Run</span>
                        </button>
                    </div>
                </div>
            </div>
        </header>

        <section id="features" class="block">
            <div class="wrap">
                <div class="sec-head reveal">
                    <h2 class="sec-title">Translate, save, practice, then automate it.</h2>
                    <p class="sec-desc">
                        Four tools working together so a word you look up today becomes a word you actually remember.
                    </p>
                </div>
                <div class="feat-grid">
                    <div class="feat reveal">
                        <div class="feat-head">
                            <div class="feat-ico">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M4 5h7M9 3v2c0 4-2 7-6 8M5 9c0 3 3 5 6 6M14 19l4-9 4 9M15.5 16h5" />
                                </svg>
                            </div>
                            <h3>Translate instantly</h3>
                        </div>
                        <p>
                            Translate words and short phrases with your selected language pair, with keyboard shortcuts
                            for everything.
                        </p>
                    </div>
                    <div class="feat reveal">
                        <div class="feat-head">
                            <div class="feat-ico">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path
                                        d="M12 6c-2-1.3-5-1.3-8 0v13c3-1.3 6-1.3 8 0M12 6c2-1.3 5-1.3 8 0v13c-3-1.3-6-1.3-8 0M12 6v13"
                                    />
                                </svg>
                            </div>
                            <h3>Build your vocabulary</h3>
                        </div>
                        <p>
                            Save your own word pairs and keep everything in one personal, searchable list across
                            devices.
                        </p>
                    </div>
                    <div class="feat reveal">
                        <div class="feat-head">
                            <div class="feat-ico">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M4 20V10M10 20V4M16 20v-6M22 20H2" />
                                </svg>
                            </div>
                            <h3>Practice and track progress</h3>
                        </div>
                        <p>Run website exercises, quiz mode and statistics to see exactly how each word is sticking.</p>
                    </div>
                    <div class="feat reveal">
                        <div class="feat-head">
                            <div class="feat-ico">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path
                                        d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3M21.5 4.5l-3.4 15.9c-.2.7-.6.9-1.2.6L9 16M21.5 4.5 9 16"
                                    />
                                </svg>
                            </div>
                            <h3>Automate in Telegram</h3>
                        </div>
                        <p>
                            Enable daily exercises, control the schedule and keep learning without ever leaving
                            Telegram.
                        </p>
                    </div>
                </div>
            </div>
        </section>

        <section id="showcase" class="block">
            <div class="wrap">
                <div class="sec-head reveal">
                    <h2 class="sec-title">Your words, organised the way you learn.</h2>
                    <p class="sec-desc">
                        Build and share your own collections, or start from ready-made sets curated by Termorize. Then
                        watch each word climb toward mastery.
                    </p>
                </div>
                <div class="showcase-grid">
                    <div class="show-card reveal">
                        <div class="card-title">Collections</div>
                        <div class="card-sub">Themed sets you can keep private or publish globally.</div>
                        <div class="coll-list">
                            <div class="coll-item">
                                <div class="ci-left">
                                    <span class="ci-name">Human Body Parts &middot; IT / RU</span>
                                    <span class="ci-flags"
                                        >🇮🇹 🇷🇺 &nbsp;<span
                                            style="color: var(--text-dim); font-size: 11.5px; font-weight: 500"
                                            >4 translations</span
                                        ></span
                                    >
                                </div>
                            </div>
                            <div class="coll-item">
                                <div class="ci-left">
                                    <span class="ci-name">German Dishes</span>
                                    <span class="ci-flags"
                                        >🇩🇪 🇬🇧 &nbsp;<span
                                            style="color: var(--text-dim); font-size: 11.5px; font-weight: 500"
                                            >10 translations</span
                                        ></span
                                    >
                                </div>
                            </div>
                            <div class="coll-item">
                                <div class="ci-left">
                                    <span class="ci-name">10 Most Popular Trees</span>
                                    <span class="ci-flags"
                                        >🇮🇹 🇷🇺 &nbsp;<span
                                            style="color: var(--text-dim); font-size: 11.5px; font-weight: 500"
                                            >10 translations</span
                                        ></span
                                    >
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="show-card reveal">
                        <div class="card-title">Saved words</div>
                        <div class="card-sub">Live learning progress on every pair you save.</div>
                        <div class="vocab-list">
                            <div class="vocab-item">
                                <div class="vi-top">
                                    <span class="vi-word">🇫🇷 le bonheur<span class="ar">:</span> happiness 🇬🇧</span>
                                    <span class="vi-pct">10%</span>
                                </div>
                                <div class="bar"><i style="width: 10%"></i></div>
                            </div>
                            <div class="vocab-item">
                                <div class="vi-top">
                                    <span class="vi-word"
                                        >🇩🇪 der Schmetterling<span class="ar">:</span> mariposa 🇪🇸</span
                                    >
                                    <span class="vi-pct">35%</span>
                                </div>
                                <div class="bar"><i style="width: 35%"></i></div>
                            </div>
                            <div class="vocab-item">
                                <div class="vi-top">
                                    <span class="vi-word">🇮🇹 la libreria<span class="ar">:</span> bookshop 🇬🇧</span>
                                    <span class="vi-pct">55%</span>
                                </div>
                                <div class="bar"><i style="width: 55%"></i></div>
                            </div>
                            <div class="vocab-item">
                                <div class="vi-top">
                                    <span class="vi-word">🇷🇺 свобода<span class="ar">:</span> die Freiheit 🇩🇪</span>
                                    <span class="vi-pct">80%</span>
                                </div>
                                <div class="bar"><i style="width: 80%"></i></div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <section id="how" class="block">
            <div class="wrap">
                <div class="sec-head reveal">
                    <h2 class="sec-title">From a quick lookup to a learned word.</h2>
                </div>
                <div class="steps">
                    <div class="step reveal">
                        <div class="num">01</div>
                        <h3>Translate &amp; save</h3>
                        <p>Look up a word in your language pair and save it to your vocabulary with one shortcut.</p>
                    </div>
                    <div class="step reveal">
                        <div class="num">02</div>
                        <h3>Practice</h3>
                        <p>Run quizzes on the web or let the bot send daily exercises on your own schedule.</p>
                    </div>
                    <div class="step reveal">
                        <div class="num">03</div>
                        <h3>Track &amp; repeat</h3>
                        <p>Statistics show what's sticking, so Termorize keeps resurfacing the words you need.</p>
                    </div>
                </div>
            </div>
        </section>

        <section id="telegram" class="block">
            <div class="wrap">
                <div class="tg-band reveal">
                    <div class="tg-glow"></div>
                    <div class="tg-copy">
                        <h2>Keep learning inside Telegram.</h2>
                        <p>
                            One sign-in connects the bot and the website. Set how many exercises you want each day and
                            let them arrive automatically.
                        </p>
                        <div class="tg-feats">
                            <div class="tg-feat">
                                <span class="ck">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                                        <path d="M5 12l4 4 10-10" />
                                    </svg>
                                </span>
                                Daily exercises delivered on your schedule
                            </div>
                            <div class="tg-feat">
                                <span class="ck">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                                        <path d="M5 12l4 4 10-10" />
                                    </svg>
                                </span>
                                Pick 1–100 exercises per day, your timezone
                            </div>
                            <div class="tg-feat">
                                <span class="ck">
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                                        <path d="M5 12l4 4 10-10" />
                                    </svg>
                                </span>
                                The same account across bot and web
                            </div>
                        </div>
                    </div>
                    <div class="tg-card">
                        <div class="tg-ico">
                            <svg viewBox="0 0 24 24" fill="none">
                                <path
                                    d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                    fill="currentColor"
                                />
                            </svg>
                        </div>
                        <h4>Login with Telegram</h4>
                        <p>Sign in once and keep the same account across the bot and website.</p>
                        <button
                            class="btn btn-tg"
                            style="width: 100%"
                            :disabled="isLoading"
                            @click="startTelegramLogin"
                        >
                            <svg viewBox="0 0 24 24" fill="none">
                                <path
                                    d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                    fill="currentColor"
                                />
                            </svg>
                            <span v-if="isTelegramLoading">{{
                                isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                            }}</span>
                            <span v-else>Continue via Telegram</span>
                        </button>
                    </div>
                </div>
            </div>
        </section>

        <section class="final">
            <div class="wrap reveal">
                <h2><span class="block">Start remembering</span><span class="block">the words that matter.</span></h2>
                <p>
                    Translate, build your vocabulary and practice on the web, with a Telegram bot for daily learning.
                    One account.
                </p>
                <div class="final-actions">
                    <button
                        type="button"
                        class="btn btn-green btn-lg"
                        :disabled="isLoading"
                        @click="tryWithoutRegistration"
                    >
                        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                            <path d="M8 5v14l11-7z" />
                        </svg>
                        <span v-if="isGuestLoading">Preparing your practice...</span>
                        <span v-else>Try without registration</span>
                    </button>
                    <button class="btn btn-tg btn-lg" :disabled="isLoading" @click="startTelegramLogin">
                        <svg viewBox="0 0 24 24" fill="none">
                            <path
                                d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                fill="currentColor"
                            />
                        </svg>
                        <span v-if="isTelegramLoading">{{
                            isInsideTelegram ? 'Signing in...' : 'Redirecting...'
                        }}</span>
                        <span v-else>Continue via Telegram</span>
                    </button>
                </div>
            </div>
        </section>

        <footer>
            <div class="wrap foot-inner">
                <a class="brand" href="#top">
                    <span class="mark">
                        <svg viewBox="0 0 64 64" fill="none">
                            <rect x="2" y="2" width="60" height="60" rx="17" fill="url(#brandGrad)" />
                            <g stroke="#06140c" stroke-width="6.2" stroke-linecap="round">
                                <g opacity="0.22" transform="translate(7 7)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <g opacity="0.45" transform="translate(3.5 3.5)">
                                    <path d="M20 23h24M32 23v23" />
                                </g>
                                <path d="M20 23h24M32 23v23" />
                            </g>
                        </svg>
                    </span>
                    Termorize
                </a>
                <div class="foot-links">
                    <a href="#features">Features</a>
                    <a href="#showcase">Showcase</a>
                    <a href="#how">How it works</a>
                    <a href="#telegram">Telegram</a>
                </div>
                <div class="foot-copy">&copy; 2026 Termorize. Vocabulary trainer with Telegram support.</div>
            </div>
        </footer>
    </div>
</template>

<style src="../assets/landing.css"></style>
