;(() => {
    if (globalThis.__termorizeSelectionOverlayInstalled) return
    globalThis.__termorizeSelectionOverlayInstalled = true

    const LANGUAGE_NAMES = {
        en: 'English',
        ru: 'Russian',
        it: 'Italian',
        de: 'German',
        es: 'Spanish',
        fr: 'French',
        pl: 'Polish',
        tr: 'Turkish',
        pt: 'Portuguese',
        uk: 'Ukrainian',
    }

    let host = null
    let currentTranslation = null
    let latestRequest = 0
    let elements = null
    let anchorRect = null
    let resizeObserver = null

    function runtimeMessage(message) {
        return new Promise((resolve) => {
            chrome.runtime.sendMessage(message, (response) => {
                if (chrome.runtime.lastError) {
                    resolve({ ok: false, reason: 'network' })
                    return
                }
                resolve(response || { ok: false, reason: 'server' })
            })
        })
    }

    function languageName(code) {
        return LANGUAGE_NAMES[code] || String(code || '').toUpperCase()
    }

    function close() {
        resizeObserver?.disconnect()
        resizeObserver = null
        host?.remove()
        host = null
        elements = null
        currentTranslation = null
        anchorRect = null
        latestRequest += 1
    }

    function positionHost(rect) {
        if (!host) return
        const margin = 12
        const width = Math.min(380, window.innerWidth - margin * 2)
        let left = rect?.left ?? window.innerWidth - width - 20
        left = Math.max(margin, Math.min(left, window.innerWidth - width - margin))
        let top = rect?.bottom ? rect.bottom + 10 : 72

        host.style.width = `${width}px`
        host.style.left = `${left}px`
        host.style.top = `${Math.max(margin, top)}px`

        const height = host.getBoundingClientRect().height
        if (top + height > window.innerHeight - margin && rect?.top) {
            top = Math.max(margin, rect.top - height - 10)
            host.style.top = `${top}px`
        } else if (top + height > window.innerHeight - margin) {
            host.style.top = `${Math.max(margin, window.innerHeight - height - margin)}px`
        }
    }

    function template() {
        return `
            <style>
                :host { all: initial; color-scheme: light dark; }
                *, *::before, *::after { box-sizing: border-box; }
                .panel {
                    --bg: #fbfdfc; --surface: #fff; --muted-surface: #edf3ef; --fg: #17211d;
                    --muted: #596960; --border: #d5e0d9; --strong-border: #c5d3cb;
                    --primary: #217a4b; --primary-hover: #19663d; --primary-fg: #f6fff9;
                    --error: #a4312b; --warning: #835b10; --success: #17643b;
                    width: 100%; max-height: min(540px, calc(100vh - 24px)); overflow: auto; padding: 16px;
                    color: var(--fg); background: var(--bg); border: 1px solid var(--border); border-radius: 13px;
                    box-shadow: 0 18px 50px rgb(10 31 20 / 24%); font: 14px/1.45 ui-sans-serif,
                        system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
                    animation: enter 180ms cubic-bezier(.16, 1, .3, 1);
                }
                button, select, textarea { color: inherit; font: inherit; }
                button:focus-visible, select:focus-visible, textarea:focus-visible { outline: 2px solid #2f8c5a; outline-offset: 2px; }
                textarea::selection { color: #10231a; background: #cfe9d9; }
                h2, p { margin: 0; }
                .header, .brand, .field-head, .actions, .signed-in { display: flex; align-items: center; }
                .header { justify-content: space-between; margin-bottom: 14px; }
                .brand { gap: 9px; }
                .mark { display: grid; width: 30px; height: 30px; place-items: center; color: var(--primary-fg);
                    background: var(--primary); border-radius: 8px; font-size: 14px; font-weight: 800; }
                h2 { font-size: 15px; font-weight: 750; letter-spacing: -.015em; }
                .subtitle { margin-top: 1px; color: var(--muted); font-size: 11px; }
                .close { position: relative; width: 34px; height: 34px; padding: 0; color: transparent; background: transparent;
                    border: 0; border-radius: 8px; cursor: pointer; }
                .close:hover { background: var(--muted-surface); }
                .close::before, .close::after { position: absolute; top: 16px; left: 9px; width: 16px; height: 1.5px;
                    content: ""; background: var(--muted); border-radius: 999px; }
                .close::before { transform: rotate(45deg); } .close::after { transform: rotate(-45deg); }
                .signed-in { gap: 7px; margin: -2px 0 12px; color: var(--muted); font-size: 11px; }
                .dot { width: 6px; height: 6px; background: var(--primary); border-radius: 50%; }
                .target-field { display: grid; grid-template-columns: 1fr 176px; gap: 12px; align-items: center;
                    margin-bottom: 14px; font-size: 12px; font-weight: 700; }
                select, textarea { width: 100%; color: var(--fg); background: var(--surface); border: 1px solid var(--strong-border); border-radius: 8px; }
                select { min-height: 40px; padding: 0 32px 0 10px; cursor: pointer; }
                .field { display: grid; gap: 6px; }
                .field-head { justify-content: space-between; font-size: 11px; font-weight: 700; }
                .language { color: var(--muted); font-weight: 600; }
                textarea { min-height: 68px; resize: vertical; padding: 9px 10px; caret-color: var(--primary); font-size: 13px; line-height: 1.45; }
                .arrow { display: flex; align-items: center; gap: 7px; margin: 8px 0; color: var(--muted); }
                .arrow::before, .arrow::after { height: 1px; flex: 1; content: ""; background: var(--border); }
                .arrow svg { width: 16px; height: 16px; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
                .message { min-height: 18px; margin-top: 8px; color: var(--muted); font-size: 11px; }
                .message[data-variant="error"] { color: var(--error); }
                .message[data-variant="warning"] { color: var(--warning); }
                .message[data-variant="success"] { color: var(--success); }
                .actions { gap: 8px; margin-top: 8px; }
                .button { min-height: 40px; padding: 0 13px; border: 1px solid transparent; border-radius: 8px;
                    font-size: 12px; font-weight: 700; cursor: pointer; }
                .button.primary { margin-left: auto; color: var(--primary-fg); background: var(--primary); }
                .state .button.primary { margin-left: 0; }
                .button.primary:hover { background: var(--primary-hover); }
                .button.secondary { color: var(--fg); background: var(--surface); border-color: var(--strong-border); }
                .button.secondary:hover { background: var(--muted-surface); }
                .button:disabled { cursor: not-allowed; opacity: .58; }
                .state { padding: 22px 10px 18px; text-align: center; }
                .state svg { width: 25px; height: 25px; margin-bottom: 10px; fill: none; stroke: var(--muted); stroke-width: 1.7;
                    stroke-linecap: round; stroke-linejoin: round; }
                .state p { max-width: 34ch; margin: 6px auto 15px; color: var(--muted); font-size: 12px; line-height: 1.5; }
                [hidden] { display: none !important; }
                @keyframes enter { from { opacity: 0; transform: translateY(7px) scale(.99); } }
                @media (prefers-color-scheme: dark) {
                    .panel { --bg: #142019; --surface: #101a14; --muted-surface: #223229; --fg: #edf5f0;
                        --muted: #adbbb3; --border: #30463a; --strong-border: #405348; --primary: #35a568;
                        --primary-hover: #43b677; --primary-fg: #071b10; --error: #f0918a; --warning: #e4bf70; --success: #74d49c; }
                    textarea::selection { color: #edf5f0; background: #245c3b; }
                }
                @media (prefers-reduced-motion: reduce) { .panel { animation: none; } }
            </style>
            <section class="panel" role="dialog" aria-modal="false" aria-labelledby="termorize-selection-title">
                <header class="header">
                    <div class="brand">
                        <span class="mark" aria-hidden="true">T</span>
                        <div><h2 id="termorize-selection-title">Termorize</h2><p class="subtitle">Selected-text translation</p></div>
                    </div>
                    <button class="close" type="button" aria-label="Close Termorize"></button>
                </header>
                <div class="loading state"><p>Loading your language settings…</p></div>
                <div class="signed-out state" hidden>
                    <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 15v2M8 10V7a4 4 0 0 1 8 0v3M6 10h12v10H6z" /></svg>
                    <h2>Sign in to translate</h2>
                    <p>Open Termorize, sign in, then press Alt + T again.</p>
                    <button class="button primary login" type="button">Sign in to Termorize</button>
                </div>
                <div class="empty state" hidden>
                    <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 4h10M7 8h8M7 12h10M7 16h6M5 2v20M19 2v20" /></svg>
                    <h2>No text selected</h2>
                    <p>Highlight a word or phrase on this page, then press Alt + T.</p>
                </div>
                <div class="workspace" hidden>
                    <div class="signed-in"><span class="dot" aria-hidden="true"></span><span class="account">Signed in</span></div>
                    <label class="target-field"><span>Translate to</span><select class="target"></select></label>
                    <div class="field">
                        <div class="field-head"><span>Selected text</span><span class="language source-language">Detecting…</span></div>
                        <textarea class="source" rows="2" maxlength="5000"></textarea>
                    </div>
                    <div class="arrow" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m8 10 4 4 4-4" /></svg></div>
                    <div class="field">
                        <div class="field-head"><span>Translation</span><span class="language translated-language"></span></div>
                        <textarea class="translated" rows="2" maxlength="5000"></textarea>
                    </div>
                    <div class="message" role="status" aria-live="polite"></div>
                    <div class="actions">
                        <button class="button secondary retry" type="button">Translate again</button>
                        <button class="button primary save" type="button">Save to vocabulary</button>
                    </div>
                </div>
            </section>`
    }

    function setMessage(text = '', variant = '') {
        elements.message.textContent = text
        if (variant) elements.message.dataset.variant = variant
        else delete elements.message.dataset.variant
    }

    function showState(state) {
        elements.loading.hidden = state !== 'loading'
        elements.signedOut.hidden = state !== 'signed-out'
        elements.empty.hidden = state !== 'empty'
        elements.workspace.hidden = state !== 'workspace'
        positionHost(anchorRect)
    }

    function setBusy(busy) {
        elements.target.disabled = busy
        elements.retry.disabled = busy
        elements.save.disabled = busy || !currentTranslation
        if (busy) {
            elements.translated.value = ''
            elements.translated.placeholder = 'Translating…'
            setMessage('Detecting the source language and translating…')
        } else {
            elements.translated.placeholder = ''
        }
    }

    function fillLanguages(languages, selected) {
        elements.target.replaceChildren()
        for (const code of languages) {
            const option = document.createElement('option')
            option.value = code
            option.textContent = languageName(code)
            option.selected = code === selected
            elements.target.append(option)
        }
    }

    function errorMessage(response) {
        if (response.reason === 'same-language') {
            return `This text is already ${languageName(response.detectedLanguage)}. Choose another target language.`
        }
        if (response.reason === 'unsupported-language') {
            return `Detected ${languageName(response.detectedLanguage)}, which Termorize does not support yet.`
        }
        if (response.reason === 'network') return 'Could not reach Termorize. Check your connection and try again.'
        return 'Termorize could not translate this selection. Try again in a moment.'
    }

    async function translate() {
        const text = elements.source.value.trim()
        if (!text) {
            currentTranslation = null
            elements.translated.value = ''
            elements.save.disabled = true
            setMessage('Enter or select text to translate.', 'warning')
            return
        }

        const requestId = ++latestRequest
        currentTranslation = null
        elements.sourceLanguage.textContent = 'Detecting…'
        elements.translatedLanguage.textContent = languageName(elements.target.value)
        setBusy(true)

        const response = await runtimeMessage({
            type: 'TRANSLATE_SELECTION',
            text,
            targetLanguage: elements.target.value,
        })
        if (requestId !== latestRequest || !elements) return
        setBusy(false)

        if (!response.ok) {
            if (response.reason === 'unauthorized') {
                showState('signed-out')
                return
            }
            elements.sourceLanguage.textContent = response.detectedLanguage
                ? languageName(response.detectedLanguage)
                : 'Auto-detected'
            elements.translated.value = ''
            setMessage(errorMessage(response), response.reason === 'same-language' ? 'warning' : 'error')
            return
        }

        currentTranslation = response.translation
        elements.source.value = response.translation.original
        elements.translated.value = response.translation.translated
        elements.sourceLanguage.textContent = languageName(response.translation.originalLanguage)
        elements.translatedLanguage.textContent = languageName(response.translation.targetLanguage)
        elements.save.disabled = false
        setMessage('Ready to save. You can edit either field first.')
        positionHost(anchorRect)
    }

    async function changeTarget() {
        currentTranslation = null
        elements.save.disabled = true
        elements.save.textContent = 'Save to vocabulary'
        const targetLanguage = elements.target.value
        const [settingsResponse] = await Promise.all([
            runtimeMessage({ type: 'UPDATE_TARGET_LANGUAGE', targetLanguage }),
            translate(),
        ])
        if (!elements) return
        if (!settingsResponse.ok && settingsResponse.reason === 'unauthorized') showState('signed-out')
        else if (!settingsResponse.ok)
            setMessage('Translation updated, but the language preference was not saved.', 'warning')
    }

    async function save() {
        if (!currentTranslation) return
        const original = elements.source.value.trim()
        const translated = elements.translated.value.trim()
        if (!original || !translated) {
            setMessage('Both fields are required before saving.', 'warning')
            return
        }

        const edited = original !== currentTranslation.original || translated !== currentTranslation.translated
        elements.save.disabled = true
        elements.save.textContent = 'Saving…'
        setMessage('Saving to your vocabulary…')

        const response = await runtimeMessage({
            type: 'SAVE_SELECTION',
            payload: {
                translation_id: currentTranslation.id,
                edited,
                original,
                translation: translated,
                original_language: currentTranslation.originalLanguage,
                translation_language: currentTranslation.targetLanguage,
            },
        })

        if (!elements) return
        elements.save.textContent = response.ok ? 'Saved' : 'Save to vocabulary'
        elements.save.disabled = response.ok
        if (response.ok) setMessage('Saved to your Termorize vocabulary.', 'success')
        else if (response.reason === 'duplicate') setMessage('This word pair is already in your vocabulary.', 'warning')
        else if (response.reason === 'unauthorized') showState('signed-out')
        else {
            setMessage(
                response.reason === 'network' ? 'Connection failed. Try again.' : 'Could not save this word pair.',
                'error'
            )
            elements.save.disabled = false
        }
    }

    async function initialize(selection) {
        if (!selection?.ok) {
            showState('empty')
            if (selection?.reason === 'too-long') {
                elements.empty.querySelector('h2').textContent = 'Selection is too long'
                elements.empty.querySelector('p').textContent =
                    'Select up to 5,000 characters, then press Alt + T again.'
                positionHost(anchorRect)
            }
            return
        }

        const session = await runtimeMessage({ type: 'GET_SESSION' })
        if (!elements) return
        if (!session.ok) {
            showState('signed-out')
            return
        }

        showState('workspace')
        elements.account.textContent = `Signed in as ${session.user.name}`
        fillLanguages(session.languages, session.user.settings.translation_target_language || 'ru')
        elements.source.value = selection.text
        await translate()
    }

    function open(selection) {
        close()
        anchorRect = selection?.rect || null
        host = document.createElement('div')
        host.id = 'termorize-selection-overlay'
        host.style.position = 'fixed'
        host.style.zIndex = '2147483647'
        const shadow = host.attachShadow({ mode: 'open' })
        shadow.innerHTML = template()
        document.documentElement.append(host)

        resizeObserver = new ResizeObserver(() => positionHost(anchorRect))
        resizeObserver.observe(shadow.querySelector('.panel'))

        elements = {
            loading: shadow.querySelector('.loading'),
            signedOut: shadow.querySelector('.signed-out'),
            empty: shadow.querySelector('.empty'),
            workspace: shadow.querySelector('.workspace'),
            account: shadow.querySelector('.account'),
            target: shadow.querySelector('.target'),
            source: shadow.querySelector('.source'),
            translated: shadow.querySelector('.translated'),
            sourceLanguage: shadow.querySelector('.source-language'),
            translatedLanguage: shadow.querySelector('.translated-language'),
            message: shadow.querySelector('.message'),
            retry: shadow.querySelector('.retry'),
            save: shadow.querySelector('.save'),
        }

        shadow.querySelector('.close').addEventListener('click', close)
        shadow.querySelector('.login').addEventListener('click', () => void runtimeMessage({ type: 'OPEN_TERMORIZE' }))
        elements.retry.addEventListener('click', translate)
        elements.save.addEventListener('click', save)
        elements.target.addEventListener('change', changeTarget)
        shadow.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                event.preventDefault()
                close()
            }
        })

        positionHost(anchorRect)
        shadow.querySelector('.close').focus()
        void initialize(selection)
    }

    chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (sender.id !== chrome.runtime.id || message?.type !== 'OPEN_SELECTION_OVERLAY') return false
        open(message.selection)
        sendResponse({ ok: true })
        return false
    })
})()
