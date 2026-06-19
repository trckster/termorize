<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getTelegramWebAppInitData, isTelegramWebApp } from '@/lib/telegram'

const router = useRouter()
const authStore = useAuthStore()

const isLoading = ref(false)
const error = ref<string | null>(null)
const isInsideTelegram = isTelegramWebApp()
const menuOpen = ref(false)

const originalTitle = document.title

onMounted(() => {
    document.title = 'Termorize — Vocabulary trainer with Telegram support'

    // scroll reveal
    const io = new IntersectionObserver(
        (entries) => {
            entries.forEach((e) => {
                if (e.isIntersecting) {
                    e.target.classList.add('in')
                    io.unobserve(e.target)
                }
            })
        },
        { threshold: 0.12, rootMargin: '0px 0px -40px 0px' }
    )

    document.querySelectorAll('.reveal').forEach((el) => io.observe(el))

    document.querySelectorAll('.feat-grid, .steps, .showcase-grid').forEach((grid) => {
        ;[...grid.querySelectorAll('.reveal')].forEach((el, i) => {
            ;(el as HTMLElement).style.transitionDelay = i * 90 + 'ms'
        })
    })

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
        isLoading.value = true

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
        isLoading.value = false
    }
}

function getErrorMessage(err: unknown): string {
    if (err instanceof Error) {
        return err.message
    }
    if (typeof err === 'object' && err !== null && 'body' in err) {
        const body = (err as { body?: { error?: string; details?: string; message?: string } }).body
        return body?.details || body?.error || body?.message || 'Login failed. Please try again.'
    }
    return 'Login failed. Please try again.'
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

        <nav>
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
                <div class="nav-links" :class="{ open: menuOpen }">
                    <a href="#features" @click="menuOpen = false">Features</a>
                    <a href="#showcase" @click="menuOpen = false">Showcase</a>
                    <a href="#how" @click="menuOpen = false">How it works</a>
                    <a href="#telegram" @click="menuOpen = false">Telegram</a>
                </div>
                <button class="nav-toggle" aria-label="Toggle menu" @click="menuOpen = !menuOpen">
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
                        <span v-if="isLoading">{{ isInsideTelegram ? 'Signing in...' : 'Redirecting...' }}</span>
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
                        Learn words faster on the <span class="accent">web</span> — and in Telegram.
                    </h1>
                    <p class="hero-sub">
                        Termorize brings translation, your personal vocabulary, practice and statistics together in one
                        site — with a Telegram bot that keeps you learning on the go.
                    </p>
                    <div class="hero-actions">
                        <button class="btn btn-tg btn-lg" :disabled="isLoading" @click="startTelegramLogin">
                            <svg viewBox="0 0 24 24" fill="none">
                                <path
                                    d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                    fill="currentColor"
                                />
                            </svg>
                            <span v-if="isLoading">{{ isInsideTelegram ? 'Signing in...' : 'Redirecting...' }}</span>
                            <span v-else>Continue via Telegram</span>
                        </button>
                        <a href="#features" class="btn btn-outline btn-lg">See how it works</a>
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
                                <div class="pane-label">
                                    From <span class="lang-pill">🇬🇧 English</span>
                                </div>
                                <div class="pane-box filled"><span class="typed">resilience</span></div>
                            </div>
                            <div class="swap-btn">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M7 4 3 8l4 4M3 8h14M17 20l4-4-4-4M21 16H7" />
                                </svg>
                            </div>
                            <div>
                                <div class="pane-label">
                                    To <span class="lang-pill">🇷🇺 Russian</span>
                                </div>
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
                    <h2 class="sec-title">Translate, save, practice — then automate it.</h2>
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
                        <p>Translate words and short phrases with your selected language pair, with keyboard shortcuts for everything.</p>
                    </div>
                    <div class="feat reveal">
                        <div class="feat-head">
                            <div class="feat-ico">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <path d="M12 6c-2-1.3-5-1.3-8 0v13c3-1.3 6-1.3 8 0M12 6c2-1.3 5-1.3 8 0v13c-3-1.3-6-1.3-8 0M12 6v13" />
                                </svg>
                            </div>
                            <h3>Build your vocabulary</h3>
                        </div>
                        <p>Save your own word pairs and keep everything in one personal, searchable list across devices.</p>
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
                                    <path d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3M21.5 4.5l-3.4 15.9c-.2.7-.6.9-1.2.6L9 16M21.5 4.5 9 16" />
                                </svg>
                            </div>
                            <h3>Automate in Telegram</h3>
                        </div>
                        <p>Enable daily exercises, control the schedule and keep learning without ever leaving Telegram.</p>
                    </div>
                </div>
            </div>
        </section>

        <section id="showcase" class="block">
            <div class="wrap">
                <div class="sec-head reveal">
                    <h2 class="sec-title">Your words, organised the way you learn.</h2>
                    <p class="sec-desc">
                        Build and share your own collections, or start from ready-made sets curated by Termorize — then
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
                                    <span class="vi-word"
                                        >🇫🇷 le bonheur<span class="ar">:</span> happiness 🇬🇧</span
                                    >
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
                                    <span class="vi-word"
                                        >🇮🇹 la libreria<span class="ar">:</span> bookshop 🇬🇧</span
                                    >
                                    <span class="vi-pct">55%</span>
                                </div>
                                <div class="bar"><i style="width: 55%"></i></div>
                            </div>
                            <div class="vocab-item">
                                <div class="vi-top">
                                    <span class="vi-word"
                                        >🇷🇺 свобода<span class="ar">:</span> die Freiheit 🇩🇪</span
                                    >
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
                        <button class="btn btn-tg" style="width: 100%" :disabled="isLoading" @click="startTelegramLogin">
                            <svg viewBox="0 0 24 24" fill="none">
                                <path
                                    d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                    fill="currentColor"
                                />
                            </svg>
                            <span v-if="isLoading">{{ isInsideTelegram ? 'Signing in...' : 'Redirecting...' }}</span>
                            <span v-else>Continue via Telegram</span>
                        </button>
                    </div>
                </div>
            </div>
        </section>

        <section class="final">
            <div class="wrap reveal">
                <h2><span class="block">Start remembering</span><span class="block">the words that matter.</span></h2>
                <p>Translate, build your vocabulary and practice on the web, with a Telegram bot for daily learning. One account.</p>
                <div class="final-actions">
                    <button class="btn btn-tg btn-lg" :disabled="isLoading" @click="startTelegramLogin">
                        <svg viewBox="0 0 24 24" fill="none">
                            <path
                                d="M21.5 4.5 2.5 11.8c-1 .4-1 .9-.2 1.1l4.7 1.5 1.8 5.6c.2.6.4.8 1 .8l3-2.3 4.7 3.5c.6.3 1 .1 1.2-.6l3.4-15.9c.2-.9-.3-1.3-1.4-.9z"
                                fill="currentColor"
                            />
                        </svg>
                        <span v-if="isLoading">{{ isInsideTelegram ? 'Signing in...' : 'Redirecting...' }}</span>
                        <span v-else>Continue via Telegram</span>
                    </button>
                    <a href="#features" class="btn btn-outline btn-lg">Explore features</a>
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

<style>
.landing-view {
    /* Colors derive from the global theme tokens (set per data-theme + .dark on
       <html>), so the landing follows whichever palette the user picked. */
    --bg: hsl(var(--background));
    --bg-deep: hsl(var(--background));
    --surface: hsl(var(--card));
    --surface-2: hsl(var(--muted));
    --border: hsl(var(--border));
    --border-soft: hsl(var(--border) / 0.5);
    --text: hsl(var(--foreground));
    --text-muted: hsl(var(--muted-foreground));
    --text-dim: hsl(var(--muted-foreground) / 0.72);
    --green: hsl(var(--primary));
    --green-bright: hsl(var(--ring));
    --green-deep: hsl(var(--primary));
    --tg: #229ed9;
    --tg-bright: #2aabee;
    --font-display: 'Space Grotesk', sans-serif;
    --font-body: 'Hanken Grotesk', sans-serif;
    --maxw: 1200px;

    --shadow-mock: 0 30px 80px -30px rgb(0 0 0 / 0.14), inset 0 1px 0 hsl(0 0% 100% / 0.4);
    --shadow-mock-run: 0 24px 60px -20px rgb(0 0 0 / 0.14);
    --shadow-feat-hover: 0 20px 50px -28px rgb(0 0 0 / 0.14);
    --shadow-brand: 0 4px 12px hsl(var(--primary) / 0.22);
    --shadow-btn-tg: 0 6px 20px rgb(34 158 217 / 0.18);
    --shadow-btn-tg-hover: 0 10px 26px rgb(34 158 217 / 0.26);
    --shadow-btn-green: 0 6px 20px hsl(var(--primary) / 0.2);
    --shadow-btn-green-hover: 0 10px 26px hsl(var(--primary) / 0.3);
    --shadow-tg-ico: 0 8px 24px rgb(34 158 217 / 0.28);
    --bg-mock-dot: hsl(var(--muted-foreground) / 0.4);
    --bg-kbd: hsl(var(--muted));
    --bg-feat-ico: hsl(var(--primary) / 0.1);
    --bg-step-num: hsl(var(--primary) / 0.1);
    --bg-bar: hsl(var(--muted));
    --bg-flag-more: hsl(var(--primary) / 0.1);
    --bg-tg-feat: rgb(34 158 217 / 0.1);
    --bg-nav: hsl(var(--background) / 0.72);
    --bg-nav-links: hsl(var(--background) / 0.96);
    --gradient-feat-end: hsl(var(--secondary));
    --color-error: hsl(var(--destructive));

    font-family: var(--font-body);
    color: var(--text);
    line-height: 1.55;
    -webkit-font-smoothing: antialiased;
    /* `clip` (not `hidden`) prevents horizontal overflow WITHOUT making this a
       scroll container — `overflow-x: hidden` forces overflow-y to compute to
       `auto`, which breaks `position: sticky` on the nav (it scrolls away). */
    overflow-x: clip;
    background: var(--bg);
    min-height: 100vh;
}

.dark .landing-view {
    /* Colors inherit from the global dark tokens automatically (they're var()
       references). Only the dark-specific depth cues need overriding here. */
    --shadow-mock: 0 30px 80px -30px rgb(0 0 0 / 0.8), inset 0 1px 0 hsl(var(--foreground) / 0.06);
    --shadow-mock-run: 0 24px 60px -20px rgb(0 0 0 / 0.85);
    --shadow-feat-hover: 0 20px 50px -28px rgb(0 0 0 / 0.7);
    --shadow-brand: 0 4px 12px hsl(var(--primary) / 0.38);
    --shadow-btn-green: 0 6px 20px hsl(var(--primary) / 0.3);
    --shadow-btn-green-hover: 0 10px 26px hsl(var(--primary) / 0.42);
    --bg-feat-ico: hsl(var(--primary) / 0.14);
    --bg-step-num: hsl(var(--primary) / 0.14);
    --bg-bar: hsl(var(--muted) / 0.6);
    --bg-flag-more: hsl(var(--primary) / 0.16);
}

.landing-view a {
    color: inherit;
    text-decoration: none;
}

.landing-view .wrap {
    max-width: var(--maxw);
    margin: 0 auto;
    padding: 0 28px;
    position: relative;
    z-index: 1;
}

/* ambient green glow */
.landing-view .glow {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    background:
        radial-gradient(900px 600px at 78% -8%, hsl(var(--primary) / 0.1), transparent 60%),
        radial-gradient(1100px 800px at 10% 8%, hsl(var(--primary) / 0.07), transparent 55%);
}

.dark .landing-view .glow {
    background:
        radial-gradient(900px 600px at 78% -8%, hsl(var(--primary) / 0.18), transparent 60%),
        radial-gradient(1100px 800px at 10% 8%, hsl(var(--primary) / 0.12), transparent 55%);
}

/* ---------- nav ---------- */
.landing-view nav {
    position: sticky;
    top: 0;
    z-index: 50;
    backdrop-filter: blur(14px);
    background: var(--bg-nav);
    border-bottom: 1px solid var(--border-soft);
}
.landing-view .nav-inner {
    height: 68px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: relative;
}
.landing-view .brand {
    display: flex;
    align-items: center;
    gap: 11px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 19px;
    letter-spacing: -0.01em;
}
.landing-view .brand .mark {
    width: 32px;
    height: 32px;
    display: block;
    flex-shrink: 0;
    filter: drop-shadow(var(--shadow-brand));
}
.landing-view .brand .mark svg {
    width: 100%;
    height: 100%;
    display: block;
}
.landing-view .nav-links {
    display: flex;
    gap: 34px;
    font-size: 14.5px;
    color: var(--text-muted);
}
.landing-view .nav-links a {
    transition: color 0.18s;
}
.landing-view .nav-links a:hover {
    color: var(--text);
}
.landing-view .nav-cta {
    display: flex;
    align-items: center;
    gap: 14px;
}
.landing-view .nav-toggle {
    display: none;
    background: none;
    border: none;
    color: var(--text);
    cursor: pointer;
    padding: 4px;
    margin-left: auto;
}
.landing-view .nav-toggle svg {
    width: 24px;
    height: 24px;
    display: block;
}

.landing-view .btn {
    display: inline-flex;
    align-items: center;
    gap: 9px;
    justify-content: center;
    font-family: var(--font-body);
    font-weight: 600;
    font-size: 14.5px;
    padding: 11px 20px;
    border-radius: 11px;
    border: 1px solid transparent;
    cursor: pointer;
    transition: transform 0.16s, box-shadow 0.2s, background 0.2s;
    white-space: nowrap;
    background: transparent;
    color: inherit;
}
.landing-view .btn svg {
    width: 17px;
    height: 17px;
}
.landing-view .btn-tg {
    background: var(--tg);
    color: #fff;
    box-shadow: var(--shadow-btn-tg);
}
.landing-view .btn-tg:hover {
    background: var(--tg-bright);
    transform: translateY(-1px);
    box-shadow: var(--shadow-btn-tg-hover);
}
.landing-view .btn-tg:disabled {
    opacity: 0.7;
    cursor: not-allowed;
    transform: none;
}
.landing-view .btn-green {
    background: var(--green);
    color: hsl(var(--primary-foreground));
    box-shadow: var(--shadow-btn-green);
}
.landing-view .btn-green:hover {
    transform: translateY(-1px);
    box-shadow: var(--shadow-btn-green-hover);
}
.landing-view .btn-outline {
    background: transparent;
    border-color: var(--border);
    color: var(--text);
}
.landing-view .btn-outline:hover {
    border-color: var(--green);
    background: hsl(var(--primary) / 0.06);
}
.dark .landing-view .btn-outline:hover {
    background: hsl(var(--primary) / 0.12);
}
.landing-view .btn-lg {
    padding: 14px 26px;
    font-size: 15.5px;
    border-radius: 13px;
}
.landing-view .run-demo .run-label {
    display: inline-block;
}
.landing-view .run-demo.is-busy {
    pointer-events: none;
}
.landing-view .run-demo.is-busy svg {
    animation: spin 0.7s linear infinite;
}
@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}
.landing-view .run-demo.nudge {
    animation: nudge 0.42s ease;
}
@keyframes nudge {
    0%,
    100% {
        transform: none;
    }
    25% {
        transform: translateX(-3px) rotate(-2.5deg);
    }
    75% {
        transform: translateX(3px) rotate(2.5deg);
    }
}

/* ---------- hero ---------- */
.landing-view .hero {
    padding: 92px 0 60px;
}
.landing-view .hero-grid {
    display: grid;
    grid-template-columns: 1.05fr 1fr;
    gap: 56px;
    align-items: center;
}
.landing-view .eyebrow {
    display: inline-flex;
    align-items: center;
    gap: 9px;
    font-size: 12.5px;
    font-weight: 600;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--green-bright);
    margin-bottom: 22px;
}
.landing-view .eyebrow .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--green-bright);
    box-shadow: 0 0 10px var(--green-bright);
}
.landing-view h1.hero-title {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: clamp(40px, 5.2vw, 64px);
    line-height: 1.02;
    letter-spacing: -0.025em;
    margin-bottom: 22px;
}
.landing-view h1.hero-title .accent {
    color: var(--green-bright);
}
.landing-view .hero-sub {
    font-size: 18.5px;
    color: var(--text-muted);
    max-width: 480px;
    margin-bottom: 32px;
}
.landing-view .hero-actions {
    display: flex;
    gap: 14px;
    align-items: center;
    flex-wrap: wrap;
}
.landing-view .hero-error {
    margin-top: 12px;
    font-size: 14px;
    color: var(--color-error);
}
.landing-view .hero-meta {
    margin-top: 26px;
    display: flex;
    align-items: center;
    gap: 11px;
    font-size: 13.5px;
    color: var(--text-dim);
}
.landing-view .hero-meta .flags {
    display: inline-flex;
    align-items: center;
    gap: 9px;
}
.landing-view .hero-meta .flag-set {
    font-size: 16px;
    letter-spacing: 1px;
    white-space: nowrap;
}
.landing-view .hero-meta .flag-more {
    font-size: 11.5px;
    font-weight: 600;
    color: var(--green-bright);
    background: var(--bg-flag-more);
    border: 1px solid var(--border-soft);
    border-radius: 20px;
    padding: 3px 10px;
    white-space: nowrap;
}

/* ---------- product mockup ---------- */
.landing-view .mock {
    background: linear-gradient(165deg, var(--surface), var(--bg-deep));
    border: 1px solid var(--border);
    border-radius: 20px;
    padding: 18px;
    box-shadow: var(--shadow-mock);
    position: relative;
}
.landing-view .mock-top {
    display: flex;
    gap: 7px;
    padding: 0 4px 14px;
}
.landing-view .mock-top span {
    width: 11px;
    height: 11px;
    border-radius: 50%;
    background: var(--bg-mock-dot);
}
.landing-view .mock-panes {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 12px;
    align-items: center;
}
.landing-view .pane-label {
    font-size: 11px;
    color: var(--text-dim);
    font-weight: 600;
    margin-bottom: 7px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}
.landing-view .lang-pill {
    background: var(--surface-2);
    border: 1px solid var(--border-soft);
    border-radius: 8px;
    padding: 5px 9px;
    font-size: 11.5px;
    font-weight: 600;
    display: inline-flex;
    gap: 6px;
    align-items: center;
    color: var(--text);
}
.landing-view .pane-box {
    background: var(--bg-deep);
    border: 1px solid var(--border-soft);
    border-radius: 12px;
    padding: 14px;
    height: 130px;
    font-size: 13px;
    color: var(--text-dim);
}
.landing-view .pane-box.filled {
    color: var(--text);
}
.landing-view .pane-box .typed {
    color: var(--text);
}
.landing-view .swap-btn {
    width: 38px;
    height: 38px;
    border-radius: 50%;
    background: var(--surface-2);
    border: 1px solid var(--border);
    display: grid;
    place-items: center;
    color: var(--text-muted);
    flex-shrink: 0;
}
.landing-view .swap-btn svg {
    width: 16px;
    height: 16px;
}
.landing-view .mock-shortcuts {
    margin-top: 16px;
    padding-top: 14px;
    border-top: 1px solid var(--border-soft);
    display: flex;
    flex-direction: column;
    gap: 9px;
}
.landing-view .sc-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-size: 12.5px;
    color: var(--text-muted);
}
.landing-view kbd {
    font-family: var(--font-body);
    font-size: 10.5px;
    font-weight: 600;
    color: var(--text-muted);
    background: var(--bg-kbd);
    border: 1px solid var(--border-soft);
    border-radius: 6px;
    padding: 3px 8px;
}
.landing-view .hero-mock {
    position: relative;
}
.landing-view .mock-run {
    position: absolute;
    right: -16px;
    bottom: -118px;
    width: 210px;
    background: linear-gradient(165deg, var(--surface), var(--bg-deep));
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 18px 16px;
    text-align: center;
    box-shadow: var(--shadow-mock-run);
}
.landing-view .mock-run h4 {
    font-family: var(--font-display);
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 6px;
}
.landing-view .mock-run p {
    font-size: 11.5px;
    color: var(--text-dim);
    margin-bottom: 13px;
    line-height: 1.4;
}

/* ---------- section frame ---------- */
.landing-view section.block {
    padding: 80px 0;
    position: relative;
    z-index: 1;
}
.landing-view .sec-head {
    max-width: 640px;
    margin-bottom: 48px;
}
.landing-view .sec-title {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: clamp(28px, 3.4vw, 40px);
    letter-spacing: -0.02em;
    line-height: 1.08;
    margin-bottom: 14px;
}
.landing-view .sec-desc {
    font-size: 17px;
    color: var(--text-muted);
}

/* ---------- features ---------- */
.landing-view .feat-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 18px;
}
.landing-view .feat {
    background: var(--surface);
    border: 1px solid var(--border-soft);
    border-radius: 16px;
    padding: 26px 28px;
    transition: transform 0.2s, border-color 0.2s, box-shadow 0.2s;
}
.landing-view .feat:hover {
    transform: translateY(-3px);
    border-color: var(--border);
    box-shadow: var(--shadow-feat-hover);
}
.landing-view .feat-head {
    display: flex;
    align-items: center;
    gap: 13px;
    margin-bottom: 12px;
}
.landing-view .feat-ico {
    width: 26px;
    height: 26px;
    flex-shrink: 0;
    color: var(--green-bright);
}
.landing-view .feat-ico svg {
    width: 26px;
    height: 26px;
}
.landing-view .feat h3 {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 0;
}
.landing-view .feat p {
    font-size: 14.5px;
    color: var(--text-muted);
}

/* ---------- showcase ---------- */
.landing-view .showcase-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 22px;
}
.landing-view .show-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 18px;
    padding: 26px;
    overflow: hidden;
}
.landing-view .show-card .card-title {
    font-family: var(--font-display);
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 4px;
}
.landing-view .show-card .card-sub {
    font-size: 12.5px;
    color: var(--text-dim);
    margin-bottom: 20px;
}

/* collections mini */
.landing-view .coll-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
}
.landing-view .coll-item {
    background: var(--bg-deep);
    border: 1px solid var(--border-soft);
    border-radius: 12px;
    padding: 15px 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
}
.landing-view .coll-item .ci-left {
    display: flex;
    flex-direction: column;
    gap: 7px;
}
.landing-view .coll-item .ci-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--text);
}
.landing-view .coll-item .ci-flags {
    font-size: 14px;
    letter-spacing: 1px;
}
.landing-view .tag {
    font-size: 10.5px;
    font-weight: 600;
    padding: 3px 9px;
    border-radius: 20px;
}
.landing-view .tag.global {
    color: var(--green-bright);
    background: hsl(var(--primary) / 0.14);
}
.landing-view .tag.draft {
    color: hsl(var(--warning));
    background: hsl(var(--warning) / 0.14);
}

/* vocab mini */
.landing-view .vocab-list {
    display: flex;
    flex-direction: column;
    gap: 14px;
}
.landing-view .vocab-item .vi-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
}
.landing-view .vocab-item .vi-word {
    font-size: 14px;
    font-weight: 600;
}
.landing-view .vocab-item .vi-word .ar {
    color: var(--text-dim);
    margin: 0 6px;
    font-weight: 400;
}
.landing-view .vocab-item .vi-pct {
    font-size: 12px;
    color: var(--green-bright);
    font-weight: 600;
}
.landing-view .bar {
    height: 6px;
    border-radius: 4px;
    background: var(--bg-bar);
    overflow: hidden;
}
.landing-view .bar > i {
    display: block;
    height: 100%;
    border-radius: 4px;
    background: linear-gradient(90deg, var(--green-deep), var(--green));
}

/* ---------- steps ---------- */
.landing-view .steps {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 20px;
    counter-reset: step;
}
.landing-view .step {
    padding: 22px 16px 0 0;
    position: relative;
    border-top: 1px solid var(--border);
}
.landing-view .step .num {
    font-family: var(--font-display);
    font-size: 38px;
    font-weight: 700;
    line-height: 1;
    letter-spacing: -0.02em;
    color: var(--green-bright);
    margin: 16px 0 14px;
}
.landing-view .step h3 {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 8px;
}
.landing-view .step p {
    font-size: 14.5px;
    color: var(--text-muted);
}

/* ---------- telegram band ---------- */
.landing-view .tg-band {
    background: linear-gradient(150deg, var(--surface), var(--bg-deep));
    border: 1px solid var(--border);
    border-radius: 22px;
    padding: 44px;
    display: grid;
    grid-template-columns: 1.2fr 1fr;
    gap: 44px;
    align-items: center;
    position: relative;
    overflow: hidden;
}
.landing-view .tg-band .tg-glow {
    position: absolute;
    right: -80px;
    top: -80px;
    width: 320px;
    height: 320px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(34, 158, 217, 0.14), transparent 65%);
    pointer-events: none;
}
.dark .landing-view .tg-band .tg-glow {
    background: radial-gradient(circle, rgba(34, 158, 217, 0.22), transparent 65%);
}
.landing-view .tg-band h2 {
    font-family: var(--font-display);
    font-size: clamp(24px, 3vw, 34px);
    font-weight: 700;
    letter-spacing: -0.02em;
    margin-bottom: 12px;
}
.landing-view .tg-band p {
    font-size: 16px;
    color: var(--text-muted);
    margin-bottom: 24px;
}
.landing-view .tg-feats {
    display: flex;
    flex-direction: column;
    gap: 12px;
}
.landing-view .tg-feat {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    font-size: 14.5px;
    color: var(--text-muted);
}
.landing-view .tg-feat .ck {
    flex-shrink: 0;
    width: 22px;
    height: 22px;
    border-radius: 7px;
    background: var(--bg-tg-feat);
    color: var(--tg-bright);
    display: grid;
    place-items: center;
    margin-top: 1px;
}
.landing-view .tg-feat .ck svg {
    width: 13px;
    height: 13px;
}
.landing-view .tg-card {
    background: var(--bg-deep);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 28px;
    text-align: center;
}
.landing-view .tg-card .tg-ico {
    width: 54px;
    height: 54px;
    border-radius: 15px;
    background: var(--tg);
    display: grid;
    place-items: center;
    margin: 0 auto 16px;
    box-shadow: var(--shadow-tg-ico);
}
.landing-view .tg-card .tg-ico svg {
    width: 26px;
    height: 26px;
    color: #fff;
}
.landing-view .tg-card h4 {
    font-family: var(--font-display);
    font-size: 19px;
    font-weight: 600;
    margin-bottom: 8px;
}
.landing-view .tg-card p {
    font-size: 13.5px;
    color: var(--text-dim);
    margin-bottom: 20px;
}

/* ---------- final cta ---------- */
.landing-view .final {
    text-align: center;
    padding: 96px 0;
}
.landing-view .final h2 {
    font-family: var(--font-display);
    font-size: clamp(32px, 4.4vw, 54px);
    font-weight: 700;
    letter-spacing: -0.025em;
    line-height: 1.05;
    margin-bottom: 18px;
}
.landing-view .final h2 .block {
    display: block;
}
.landing-view .final p {
    font-size: 18px;
    color: var(--text-muted);
    max-width: 500px;
    margin: 0 auto 32px;
}
.landing-view .final-actions {
    display: flex;
    gap: 14px;
    justify-content: center;
    flex-wrap: wrap;
}

/* ---------- footer ---------- */
.landing-view footer {
    border-top: 1px solid var(--border-soft);
    padding: 40px 0;
    position: relative;
    z-index: 1;
}
.landing-view .foot-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 18px;
}
.landing-view .foot-links {
    display: flex;
    gap: 26px;
    font-size: 14px;
    color: var(--text-muted);
}
.landing-view .foot-links a:hover {
    color: var(--text);
}
.landing-view .foot-copy {
    font-size: 13px;
    color: var(--text-dim);
}

/* ---------- reveal animation ---------- */
.landing-view .reveal {
    opacity: 0;
    transform: translateY(24px);
    transition:
        opacity 0.7s cubic-bezier(0.2, 0.7, 0.3, 1),
        transform 0.7s cubic-bezier(0.2, 0.7, 0.3, 1);
}
.landing-view .reveal.in {
    opacity: 1;
    transform: none;
}

/* ---------- focus ---------- */
.landing-view a:focus-visible,
.landing-view .btn:focus-visible {
    outline: 2px solid var(--green-bright);
    outline-offset: 2px;
    border-radius: 4px;
}

/* ---------- responsive ---------- */
@media (max-width: 920px) {
    .landing-view .hero-grid {
        grid-template-columns: 1fr;
        gap: 60px;
    }
    .landing-view .mock-run {
        position: static;
        width: auto;
        margin-top: 16px;
    }
    .landing-view .showcase-grid,
    .landing-view .feat-grid,
    .landing-view .steps,
    .landing-view .tg-band {
        grid-template-columns: 1fr;
    }
}

@media (max-width: 680px) {
    .landing-view .nav-links {
        display: none;
        position: absolute;
        top: 68px;
        left: 0;
        right: 0;
        background: var(--bg-nav-links);
        border-bottom: 1px solid var(--border-soft);
        flex-direction: column;
        padding: 16px 28px;
        gap: 16px;
        z-index: 40;
    }
    .landing-view .nav-links.open {
        display: flex;
    }
    .landing-view .nav-toggle {
        display: flex;
        align-items: center;
        justify-content: center;
    }
}

@media (max-width: 560px) {
    .landing-view .wrap {
        padding: 0 18px;
    }
    .landing-view .hero {
        padding: 56px 0 40px;
    }
    .landing-view .mock-panes {
        grid-template-columns: 1fr;
    }
    .landing-view .swap-btn {
        transform: rotate(90deg);
        margin: 0 auto;
    }
    .landing-view .foot-inner {
        flex-direction: column;
        align-items: flex-start;
    }
}

@media (prefers-reduced-motion: reduce) {
    .landing-view .reveal {
        opacity: 1;
        transform: none;
        transition: none;
    }
    .landing-view .reveal.in {
        opacity: 1;
        transform: none;
    }
    .landing-view .btn,
    .landing-view .feat,
    .landing-view .nav-links a {
        transition: none;
    }
    .landing-view .feat:hover {
        transform: none;
    }
    .landing-view .run-demo.is-busy svg {
        animation: none;
    }
    .landing-view .run-demo.nudge {
        animation: none;
    }
}
</style>
