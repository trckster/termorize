const TERMORIZE_SUPPORTED_LANGUAGES = new Set(['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk'])
const TERMORIZE_LANGUAGE_NAMES = {
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

function normalizedText(value) {
    return typeof value === 'string' ? value.trim() : ''
}

function elementValue(element) {
    return normalizedText(element?.value || element?.textContent)
}

function firstValue(documentApi, selectors) {
    for (const selector of selectors) {
        const value = elementValue(documentApi.querySelector(selector))
        if (value) return value
    }
    return ''
}

function joinedValues(documentApi, selectors) {
    for (const selector of selectors) {
        const values = [...documentApi.querySelectorAll(selector)].map(elementValue).filter(Boolean)
        if (values.length > 0) return values.join(' ')
    }
    return ''
}

function extractGoogleTranslation(documentApi = document, href = location.href) {
    const source = firstValue(documentApi, ['textarea[aria-label="Source text"]', 'textarea[jsname="BJE2fc"]'])
    const target =
        firstValue(documentApi, [
            'textarea[jsname="YPqjbf"][lang]',
            'textarea[lang]:not([aria-label="Source text"])',
        ]) ||
        joinedValues(documentApi, [
            '[data-language-for-alternatives] span.ryNqvb',
            '[data-result-index="0"] span[lang]',
            'span[jsname="W297wb"]',
        ])

    const url = new URL(href)
    const targetElement = documentApi.querySelector(
        'textarea[jsname="YPqjbf"][lang], textarea[lang]:not([aria-label="Source text"])'
    )
    const sourceLanguage = normalizedText(url.searchParams.get('sl')).toLowerCase()
    const targetLanguage = normalizedText(
        url.searchParams.get('tl') || targetElement?.getAttribute('lang')
    ).toLowerCase()

    return {
        original: source,
        translation: target,
        original_language: sourceLanguage,
        translation_language: targetLanguage,
    }
}

function shortcutAction(event) {
    if (!event.isTrusted || event.altKey || event.metaKey || !event.ctrlKey || event.shiftKey) return null
    if (event.code === 'KeyE') return 'edit'
    if (event.code === 'KeyS') return 'save'
    return null
}

function validatePair(pair) {
    if (!pair.original || !pair.translation) {
        return {
            ok: false,
            message: 'Enter a word and wait for its translation before saving.',
        }
    }

    if (pair.original_language === 'auto') {
        return {
            ok: false,
            message: 'Choose the source language in Google Translate before saving.',
        }
    }

    if (!TERMORIZE_SUPPORTED_LANGUAGES.has(pair.original_language)) {
        return {
            ok: false,
            message: `Termorize does not support source language “${pair.original_language || 'unknown'}” yet.`,
        }
    }

    if (!TERMORIZE_SUPPORTED_LANGUAGES.has(pair.translation_language)) {
        return {
            ok: false,
            message: `Termorize does not support target language “${pair.translation_language || 'unknown'}” yet.`,
        }
    }

    if (pair.original_language === pair.translation_language) {
        return {
            ok: false,
            message: 'Choose two different languages before saving.',
        }
    }

    return { ok: true }
}

function sendRuntimeMessage(message) {
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

function createElement(tag, className, text) {
    const element = document.createElement(tag)
    if (className) element.className = className
    if (text !== undefined) element.textContent = text
    return element
}

function createButton(label, className, onClick) {
    const button = createElement('button', className, label)
    button.type = 'button'
    button.addEventListener('click', onClick)
    return button
}

function createShortcutHints() {
    const hints = createElement('aside', 'termorize-shortcut-hints')
    hints.setAttribute('aria-label', 'Termorize keyboard shortcuts')

    const heading = createElement('div', 'termorize-shortcut-heading')
    const mark = createElement('span', 'termorize-shortcut-mark', 'T')
    mark.setAttribute('aria-hidden', 'true')
    heading.append(mark, createElement('strong', 'termorize-shortcut-title', 'Save to Termorize'))

    const list = createElement('div', 'termorize-shortcut-list')
    const shortcuts = [
        ['Review before saving', 'Ctrl', 'E'],
        ['Save immediately', 'Ctrl', 'S'],
    ]
    for (const [label, modifier, key] of shortcuts) {
        const row = createElement('div', 'termorize-shortcut-row')
        const keys = createElement('span', 'termorize-shortcut-keys')
        keys.append(createElement('kbd', 'termorize-kbd', modifier), ' + ', createElement('kbd', 'termorize-kbd', key))
        row.append(createElement('span', 'termorize-shortcut-label', label), keys)
        list.append(row)
    }

    hints.append(heading, list)
    return hints
}

function createUi() {
    const root = createElement('div', 'termorize-extension-root')
    const toastViewport = createElement('div', 'termorize-toast-viewport')
    toastViewport.setAttribute('aria-live', 'polite')
    toastViewport.setAttribute('aria-atomic', 'true')
    root.append(toastViewport, createShortcutHints())
    document.body.append(root)

    let toastTimer = null
    let dialog = null
    let previouslyFocused = null
    let isSaving = false

    function dismissToast() {
        if (toastTimer) window.clearTimeout(toastTimer)
        toastTimer = null
        toastViewport.replaceChildren()
    }

    function showToast({ title, description, variant = 'default', action }) {
        dismissToast()

        const toast = createElement('section', `termorize-toast termorize-toast--${variant}`)
        toast.setAttribute('role', variant === 'error' ? 'alert' : 'status')

        const mark = createElement('span', 'termorize-mark', 'T')
        mark.setAttribute('aria-hidden', 'true')
        const copy = createElement('div', 'termorize-toast-copy')
        copy.append(createElement('strong', 'termorize-toast-title', title))
        copy.append(createElement('p', 'termorize-toast-description', description))

        const controls = createElement('div', 'termorize-toast-controls')
        if (action) {
            controls.append(createButton(action.label, 'termorize-toast-action', action.onClick))
        }
        const close = createButton('Close', 'termorize-icon-button', dismissToast)
        close.setAttribute('aria-label', 'Close Termorize notification')
        controls.append(close)

        toast.append(mark, copy, controls)
        toastViewport.append(toast)
        toastTimer = window.setTimeout(dismissToast, action ? 9000 : 4500)
    }

    function closeDialog() {
        if (!dialog || isSaving) return
        dialog.remove()
        dialog = null
        previouslyFocused?.focus?.()
        previouslyFocused = null
    }

    function handleResponse(response) {
        if (response.ok) {
            showToast({
                title: 'Saved to Termorize',
                description: 'The word pair is ready in your vocabulary.',
                variant: 'success',
            })
            return
        }

        if (response.reason === 'unauthorized') {
            showToast({
                title: 'Sign in to save words',
                description: 'Open Termorize, sign in, then try the shortcut again.',
                variant: 'warning',
                action: {
                    label: 'Open Termorize',
                    onClick: () => void sendRuntimeMessage({ type: 'OPEN_TERMORIZE' }),
                },
            })
            return
        }

        if (response.reason === 'duplicate') {
            showToast({
                title: 'Already in your vocabulary',
                description: 'This word pair was saved earlier.',
            })
            return
        }

        showToast({
            title: 'Could not save the word',
            description:
                response.reason === 'network'
                    ? 'Check your connection and try again.'
                    : 'Termorize could not accept this word pair. Try again in a moment.',
            variant: 'error',
        })
    }

    async function savePair(pair, button) {
        if (isSaving) return false
        isSaving = true
        if (button) {
            button.disabled = true
            button.textContent = 'Saving…'
        }

        const response = await sendRuntimeMessage({
            type: 'SAVE_VOCABULARY',
            payload: pair,
        })
        isSaving = false
        if (button) {
            button.disabled = false
            button.textContent = 'Save to vocabulary'
        }
        handleResponse(response)
        return response.ok
    }

    async function saveDirect(pair) {
        showToast({
            title: 'Saving to Termorize',
            description: `${pair.original} → ${pair.translation}`,
        })
        await savePair(pair)
    }

    function openEditor(pair) {
        if (dialog || isSaving) return
        previouslyFocused = document.activeElement

        const overlay = createElement('div', 'termorize-dialog-overlay')
        const panel = createElement('section', 'termorize-dialog')
        panel.setAttribute('role', 'dialog')
        panel.setAttribute('aria-modal', 'true')
        panel.setAttribute('aria-labelledby', 'termorize-dialog-title')

        const header = createElement('header', 'termorize-dialog-header')
        const headingGroup = createElement('div', 'termorize-dialog-heading')
        const title = createElement('h2', 'termorize-dialog-title', 'Review before saving')
        title.id = 'termorize-dialog-title'
        headingGroup.append(
            title,
            createElement(
                'p',
                'termorize-dialog-description',
                'Edit either side of the word pair. Languages stay fixed.'
            )
        )
        const close = createButton('Close', 'termorize-icon-button termorize-dialog-close', closeDialog)
        close.setAttribute('aria-label', 'Close review dialog')
        header.append(headingGroup, close)

        const form = createElement('form', 'termorize-form')
        const originalField = createElement('label', 'termorize-field')
        const originalLabel = createElement('span', 'termorize-field-header')
        originalLabel.append(
            createElement('span', 'termorize-field-label', 'Original'),
            createElement(
                'span',
                'termorize-language-chip',
                TERMORIZE_LANGUAGE_NAMES[pair.original_language] || pair.original_language
            )
        )
        const originalInput = createElement('textarea', 'termorize-textarea')
        originalInput.value = pair.original
        originalInput.maxLength = 5000
        originalInput.rows = 3
        originalField.append(originalLabel, originalInput)

        const translationField = createElement('label', 'termorize-field')
        const translationLabel = createElement('span', 'termorize-field-header')
        translationLabel.append(
            createElement('span', 'termorize-field-label', 'Translation'),
            createElement(
                'span',
                'termorize-language-chip',
                TERMORIZE_LANGUAGE_NAMES[pair.translation_language] || pair.translation_language
            )
        )
        const translationInput = createElement('textarea', 'termorize-textarea')
        translationInput.value = pair.translation
        translationInput.maxLength = 5000
        translationInput.rows = 3
        translationField.append(translationLabel, translationInput)

        const footer = createElement('footer', 'termorize-dialog-footer')
        const hint = createElement('p', 'termorize-keyboard-hint')
        const shiftKey = createElement('kbd', 'termorize-kbd', 'Shift')
        const enterKey = createElement('kbd', 'termorize-kbd', 'Enter')
        hint.append('Press ', shiftKey, ' + ', enterKey, ' to save')
        const actions = createElement('div', 'termorize-dialog-actions')
        const cancel = createButton('Cancel', 'termorize-button termorize-button--secondary', closeDialog)
        const save = createElement('button', 'termorize-button termorize-button--primary', 'Save to vocabulary')
        save.type = 'submit'
        actions.append(cancel, save)
        footer.append(hint, actions)

        form.append(originalField, translationField, footer)
        panel.append(header, form)
        overlay.append(panel)
        root.append(overlay)
        dialog = overlay

        form.addEventListener('submit', async (event) => {
            event.preventDefault()
            const editedPair = {
                ...pair,
                original: normalizedText(originalInput.value),
                translation: normalizedText(translationInput.value),
            }
            const validation = validatePair(editedPair)
            if (!validation.ok) {
                showToast({
                    title: 'Word pair is not ready',
                    description: validation.message,
                    variant: 'warning',
                })
                return
            }

            const saved = await savePair(editedPair, save)
            if (saved) {
                dialog?.remove()
                dialog = null
                previouslyFocused?.focus?.()
                previouslyFocused = null
            }
        })

        overlay.addEventListener('mousedown', (event) => {
            if (event.target === overlay) closeDialog()
        })

        panel.addEventListener('keydown', (event) => {
            if (event.key === 'Escape') {
                event.preventDefault()
                closeDialog()
                return
            }

            if (event.key === 'Enter' && event.shiftKey && !event.altKey && !event.ctrlKey && !event.metaKey) {
                event.preventDefault()
                form.requestSubmit()
                return
            }

            if (event.key === 'Tab') {
                const focusable = [...panel.querySelectorAll('button:not([disabled]), textarea:not([disabled])')]
                const first = focusable[0]
                const last = focusable[focusable.length - 1]
                if (event.shiftKey && document.activeElement === first) {
                    event.preventDefault()
                    last.focus()
                } else if (!event.shiftKey && document.activeElement === last) {
                    event.preventDefault()
                    first.focus()
                }
            }
        })

        translationInput.focus()
        translationInput.setSelectionRange(translationInput.value.length, translationInput.value.length)
    }

    return {
        showToast,
        saveDirect,
        openEditor,
        isDialogOpen: () => Boolean(dialog),
    }
}

function bootstrap() {
    if (document.querySelector('.termorize-extension-root')) return
    const ui = createUi()

    function handleAction(action) {
        if (ui.isDialogOpen()) return
        const pair = extractGoogleTranslation()
        const validation = validatePair(pair)
        if (!validation.ok) {
            ui.showToast({
                title: 'Word pair is not ready',
                description: validation.message,
                variant: 'warning',
            })
            return
        }

        if (action === 'edit') ui.openEditor(pair)
        else if (action === 'save') void ui.saveDirect(pair)
    }

    window.addEventListener(
        'keydown',
        (event) => {
            const action = shortcutAction(event)
            if (!action) return

            event.preventDefault()
            event.stopImmediatePropagation()

            handleAction(action)
        },
        true
    )

    if (chrome.runtime.onMessage) {
        chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
            if (sender.id !== chrome.runtime.id || message?.type !== 'TRIGGER_SHORTCUT') return false
            if (message.action !== 'edit' && message.action !== 'save') return false

            handleAction(message.action)
            sendResponse({ ok: true })
            return false
        })
    }
}

if (typeof chrome !== 'undefined' && chrome.runtime && typeof document !== 'undefined') {
    bootstrap()
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        extractGoogleTranslation,
        shortcutAction,
        validatePair,
    }
}
