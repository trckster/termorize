const TERMORIZE_ORIGIN = 'https://termorize.daniil.online'
const VOCABULARY_ENDPOINT = `${TERMORIZE_ORIGIN}/api/vocabulary`
const TRANSLATE_SELECTION_ENDPOINT = `${TERMORIZE_ORIGIN}/api/translate/selection`
const ME_ENDPOINT = `${TERMORIZE_ORIGIN}/api/me`
const SETTINGS_ENDPOINT = `${TERMORIZE_ORIGIN}/api/settings`
const SUPPORTED_LANGUAGE_CODES = ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk']
const SUPPORTED_LANGUAGES = new Set(SUPPORTED_LANGUAGE_CODES)
const MAX_TEXT_LENGTH = 5000
const COMMAND_ACTIONS = {
    'save-with-editing': 'edit',
    'save-without-editing': 'save',
    'translate-selection': 'translate',
}

function normalizedText(value) {
    return typeof value === 'string' ? value.trim() : ''
}

function isValidVocabularyPayload(payload) {
    if (!payload || typeof payload !== 'object') return false

    const original = normalizedText(payload.original)
    const translation = normalizedText(payload.translation)

    return (
        original.length > 0 &&
        original.length <= MAX_TEXT_LENGTH &&
        translation.length > 0 &&
        translation.length <= MAX_TEXT_LENGTH &&
        SUPPORTED_LANGUAGES.has(payload.original_language) &&
        SUPPORTED_LANGUAGES.has(payload.translation_language) &&
        payload.original_language !== payload.translation_language
    )
}

function getAuthCookie(chromeApi = chrome) {
    return new Promise((resolve) => {
        chromeApi.cookies.get({ url: TERMORIZE_ORIGIN, name: 'auth' }, (cookie) => {
            if (chromeApi.runtime.lastError) {
                resolve(null)
                return
            }
            resolve(cookie || null)
        })
    })
}

async function readJson(response) {
    if (typeof response?.json !== 'function') return null
    try {
        return await response.json()
    } catch {
        return null
    }
}

async function authenticatedRequest(url, options = {}, dependencies = {}) {
    const chromeApi = dependencies.chromeApi || chrome
    const fetchApi = dependencies.fetchApi || fetch
    const cookie = await getAuthCookie(chromeApi)

    if (!cookie?.value) return { ok: false, reason: 'unauthorized' }

    let response
    try {
        response = await fetchApi(url, {
            ...options,
            headers: {
                Accept: 'application/json',
                Authorization: `Bearer ${cookie.value}`,
                ...(options.body ? { 'Content-Type': 'application/json' } : {}),
                ...options.headers,
            },
        })
    } catch {
        return { ok: false, reason: 'network' }
    }

    const body = await readJson(response)
    if (response.status === 401) return { ok: false, reason: 'unauthorized' }

    return { ok: Boolean(response.ok), status: response.status, body }
}

async function getSession(dependencies = {}) {
    const response = await authenticatedRequest(ME_ENDPOINT, { method: 'GET' }, dependencies)
    if (!response.ok) return { ok: false, reason: response.reason || 'server' }

    if (!response.body?.settings || typeof response.body.settings !== 'object') {
        return { ok: false, reason: 'server' }
    }

    return {
        ok: true,
        user: {
            name: normalizedText(response.body.name) || normalizedText(response.body.username) || 'Termorize user',
            settings: response.body.settings,
        },
        languages: SUPPORTED_LANGUAGE_CODES,
    }
}

async function translateSelectedText(text, targetLanguage, dependencies = {}) {
    const fromWord = normalizedText(text)
    if (!fromWord || fromWord.length > MAX_TEXT_LENGTH || !SUPPORTED_LANGUAGES.has(targetLanguage)) {
        return { ok: false, reason: 'invalid' }
    }

    const response = await authenticatedRequest(
        TRANSLATE_SELECTION_ENDPOINT,
        {
            method: 'POST',
            body: JSON.stringify({
                from_word: fromWord,
                to_language: targetLanguage,
            }),
        },
        dependencies
    )

    if (!response.ok) {
        if (response.reason) return { ok: false, reason: response.reason }
        if (response.status === 422 && response.body?.error === 'source language matches target language') {
            return {
                ok: false,
                reason: 'same-language',
                detectedLanguage: response.body.detected_language,
            }
        }
        if (response.status === 422 && response.body?.error === 'unsupported source language') {
            return {
                ok: false,
                reason: 'unsupported-language',
                detectedLanguage: response.body.detected_language,
            }
        }
        return { ok: false, reason: 'server' }
    }

    if (!response.body?.id || !SUPPORTED_LANGUAGES.has(response.body.original_language)) {
        return { ok: false, reason: 'server' }
    }

    return {
        ok: true,
        translation: {
            id: response.body.id,
            original: fromWord,
            translated: normalizedText(response.body.translation),
            originalLanguage: response.body.original_language,
            targetLanguage,
            source: response.body.source,
        },
    }
}

async function updateTargetLanguage(targetLanguage, dependencies = {}) {
    if (!SUPPORTED_LANGUAGES.has(targetLanguage)) return { ok: false, reason: 'invalid' }

    const session = await getSession(dependencies)
    if (!session.ok) return session

    const currentSettings = session.user.settings
    if (currentSettings.translation_target_language === targetLanguage) {
        return { ok: true, settings: currentSettings }
    }

    const nextSourceLanguage =
        currentSettings.translation_source_language === targetLanguage
            ? currentSettings.translation_target_language
            : currentSettings.translation_source_language
    const nextSettings = {
        ...currentSettings,
        translation_source_language: nextSourceLanguage,
        translation_target_language: targetLanguage,
    }

    const response = await authenticatedRequest(
        SETTINGS_ENDPOINT,
        {
            method: 'PUT',
            body: JSON.stringify(nextSettings),
        },
        dependencies
    )

    if (!response.ok) return { ok: false, reason: response.reason || 'server' }
    return {
        ok: true,
        settings: response.body?.settings || nextSettings,
    }
}

async function saveVocabulary(payload, dependencies = {}) {
    if (!isValidVocabularyPayload(payload)) return { ok: false, reason: 'invalid' }

    const response = await authenticatedRequest(
        VOCABULARY_ENDPOINT,
        {
            method: 'POST',
            body: JSON.stringify({
                original: payload.original.trim(),
                translation: payload.translation.trim(),
                original_language: payload.original_language,
                translation_language: payload.translation_language,
            }),
        },
        dependencies
    )

    if (response.ok) return { ok: true }
    if (response.status === 409) return { ok: false, reason: 'duplicate' }
    return { ok: false, reason: response.reason || 'server' }
}

async function saveVocabularyByTranslation(translationId, dependencies = {}) {
    if (typeof translationId !== 'string' || !translationId.trim()) return { ok: false, reason: 'invalid' }

    const response = await authenticatedRequest(
        `${VOCABULARY_ENDPOINT}/translation`,
        {
            method: 'POST',
            body: JSON.stringify({ translation_id: translationId }),
        },
        dependencies
    )

    if (response.ok) return { ok: true }
    if (response.status === 409) return { ok: false, reason: 'duplicate' }
    return { ok: false, reason: response.reason || 'server' }
}

async function saveSelection(payload, dependencies = {}) {
    if (!payload?.edited && payload?.translation_id) {
        return saveVocabularyByTranslation(payload.translation_id, dependencies)
    }

    return saveVocabulary(payload, dependencies)
}

function commandAction(command, tabUrl) {
    const action = COMMAND_ACTIONS[command]
    if (!action) return null
    if (action === 'translate') return action
    if (typeof tabUrl !== 'string') return null

    try {
        const url = new URL(tabUrl)
        return url.protocol === 'https:' && url.hostname === 'translate.google.com' ? action : null
    } catch {
        return null
    }
}

function selectedTextInPage() {
    let text = ''
    let rect = null
    const activeElement = document.activeElement

    if (
        (activeElement instanceof HTMLInputElement || activeElement instanceof HTMLTextAreaElement) &&
        typeof activeElement.selectionStart === 'number' &&
        typeof activeElement.selectionEnd === 'number' &&
        activeElement.selectionEnd > activeElement.selectionStart
    ) {
        text = activeElement.value.slice(activeElement.selectionStart, activeElement.selectionEnd)
        const bounds = activeElement.getBoundingClientRect()
        rect = {
            left: bounds.left,
            top: bounds.top,
            right: bounds.right,
            bottom: bounds.bottom,
        }
    } else {
        const selection = window.getSelection()
        text = selection?.toString() || ''
        if (selection?.rangeCount) {
            const bounds = selection.getRangeAt(0).getBoundingClientRect()
            rect = {
                left: bounds.left,
                top: bounds.top,
                right: bounds.right,
                bottom: bounds.bottom,
            }
        }
    }

    return { text: text.trim().slice(0, 5001), rect }
}

async function captureTabSelection(tabId, chromeApi = chrome) {
    if (!tabId || !chromeApi.scripting?.executeScript) return { ok: false, reason: 'unavailable' }

    let results
    try {
        results = await chromeApi.scripting.executeScript({
            target: { tabId, allFrames: true },
            func: selectedTextInPage,
        })
    } catch {
        return { ok: false, reason: 'unavailable' }
    }

    const selected = results?.find((entry) => normalizedText(entry.result?.text))
    if (!selected) return { ok: false, reason: 'empty' }
    if (selected.result.text.length > MAX_TEXT_LENGTH) return { ok: false, reason: 'too-long' }

    return {
        ok: true,
        text: selected.result.text,
        rect: selected.result.rect,
        frameId: selected.frameId || 0,
    }
}

async function getActiveSelection(chromeApi = chrome) {
    let tabs
    try {
        tabs = await chromeApi.tabs.query({ active: true, currentWindow: true })
    } catch {
        return { ok: false, reason: 'unavailable' }
    }

    const tab = tabs?.[0]
    if (!tab?.id) return { ok: false, reason: 'unavailable' }
    return captureTabSelection(tab.id, chromeApi)
}

function sendTabMessage(chromeApi, tabId, frameId, message) {
    return new Promise((resolve) => {
        chromeApi.tabs.sendMessage(tabId, message, { frameId }, (response) => {
            if (chromeApi.runtime.lastError) {
                resolve({ ok: false })
                return
            }
            resolve(response || { ok: true })
        })
    })
}

async function openSelectionOverlay(tab, chromeApi = chrome) {
    if (!tab?.id || !chromeApi.scripting?.executeScript) return

    const selection = await captureTabSelection(tab.id, chromeApi)
    const frameId = selection.frameId || 0

    try {
        await chromeApi.scripting.executeScript({
            target: { tabId: tab.id, frameIds: [frameId] },
            files: ['selection-overlay.js'],
        })
    } catch {
        return
    }

    await sendTabMessage(chromeApi, tab.id, frameId, {
        type: 'OPEN_SELECTION_OVERLAY',
        selection,
    })
}

function registerMessageHandler(chromeApi = chrome) {
    chromeApi.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (sender.id !== chromeApi.runtime.id) {
            sendResponse({ ok: false, reason: 'forbidden' })
            return false
        }

        const handlers = {
            SAVE_VOCABULARY: () => saveVocabulary(message.payload, { chromeApi }),
            GET_SESSION: () => getSession({ chromeApi }),
            GET_ACTIVE_SELECTION: () => getActiveSelection(chromeApi),
            TRANSLATE_SELECTION: () =>
                translateSelectedText(message.text, message.targetLanguage, {
                    chromeApi,
                }),
            UPDATE_TARGET_LANGUAGE: () => updateTargetLanguage(message.targetLanguage, { chromeApi }),
            SAVE_SELECTION: () => saveSelection(message.payload, { chromeApi }),
        }

        if (handlers[message?.type]) {
            handlers[message.type]()
                .then(sendResponse)
                .catch(() => sendResponse({ ok: false, reason: 'server' }))
            return true
        }

        if (message?.type === 'OPEN_TERMORIZE') {
            chromeApi.tabs.create({ url: `${TERMORIZE_ORIGIN}/login` })
            sendResponse({ ok: true })
            return false
        }

        return false
    })
}

function registerCommandHandler(chromeApi = chrome) {
    chromeApi.commands.onCommand.addListener((command, tab) => {
        const action = commandAction(command, tab?.url)
        if (!action || !tab?.id) return

        if (action === 'translate') {
            void openSelectionOverlay(tab, chromeApi)
            return
        }

        chromeApi.tabs.sendMessage(tab.id, { type: 'TRIGGER_SHORTCUT', action }, () => {
            void chromeApi.runtime.lastError
        })
    })
}

if (typeof chrome !== 'undefined' && chrome.runtime?.onMessage) {
    registerMessageHandler()
    if (chrome.commands?.onCommand) registerCommandHandler()
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        ME_ENDPOINT,
        SETTINGS_ENDPOINT,
        TERMORIZE_ORIGIN,
        TRANSLATE_SELECTION_ENDPOINT,
        VOCABULARY_ENDPOINT,
        captureTabSelection,
        commandAction,
        getSession,
        isValidVocabularyPayload,
        saveSelection,
        saveVocabulary,
        translateSelectedText,
        updateTargetLanguage,
    }
}
