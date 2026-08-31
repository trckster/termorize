const TERMORIZE_ORIGIN = 'https://termorize.daniil.online'
const VOCABULARY_ENDPOINT = `${TERMORIZE_ORIGIN}/api/vocabulary`
const SUPPORTED_LANGUAGES = new Set(['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk'])
const COMMAND_ACTIONS = {
    'save-with-editing': 'edit',
    'save-without-editing': 'save',
}

function isValidVocabularyPayload(payload) {
    if (!payload || typeof payload !== 'object') return false

    const original = typeof payload.original === 'string' ? payload.original.trim() : ''
    const translation = typeof payload.translation === 'string' ? payload.translation.trim() : ''

    return (
        original.length > 0 &&
        original.length <= 5000 &&
        translation.length > 0 &&
        translation.length <= 5000 &&
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

async function saveVocabulary(payload, dependencies = {}) {
    if (!isValidVocabularyPayload(payload)) {
        return { ok: false, reason: 'invalid' }
    }

    const chromeApi = dependencies.chromeApi || chrome
    const fetchApi = dependencies.fetchApi || fetch
    const cookie = await getAuthCookie(chromeApi)

    if (!cookie?.value) {
        return { ok: false, reason: 'unauthorized' }
    }

    let response
    try {
        response = await fetchApi(VOCABULARY_ENDPOINT, {
            method: 'POST',
            headers: {
                Accept: 'application/json',
                Authorization: `Bearer ${cookie.value}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                original: payload.original.trim(),
                translation: payload.translation.trim(),
                original_language: payload.original_language,
                translation_language: payload.translation_language,
            }),
        })
    } catch {
        return { ok: false, reason: 'network' }
    }

    if (response.status === 401) return { ok: false, reason: 'unauthorized' }
    if (response.status === 409) return { ok: false, reason: 'duplicate' }
    if (!response.ok) return { ok: false, reason: 'server' }

    return { ok: true }
}

function commandAction(command, tabUrl) {
    const action = COMMAND_ACTIONS[command]
    if (!action || typeof tabUrl !== 'string') return null

    try {
        const url = new URL(tabUrl)
        return url.protocol === 'https:' && url.hostname === 'translate.google.com' ? action : null
    } catch {
        return null
    }
}

function registerMessageHandler(chromeApi = chrome) {
    chromeApi.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (sender.id !== chromeApi.runtime.id) {
            sendResponse({ ok: false, reason: 'forbidden' })
            return false
        }

        if (message?.type === 'SAVE_VOCABULARY') {
            saveVocabulary(message.payload, { chromeApi })
                .then(sendResponse)
                .catch(() => sendResponse({ ok: false, reason: 'server' }))
            return true
        }

        if (message?.type === 'OPEN_TERMORIZE') {
            chromeApi.tabs.create({ url: `${TERMORIZE_ORIGIN}/login` })
            sendResponse({ ok: true })
        }

        return false
    })
}

function registerCommandHandler(chromeApi = chrome) {
    chromeApi.commands.onCommand.addListener((command, tab) => {
        const action = commandAction(command, tab?.url)
        if (!action || !tab?.id) return

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
        TERMORIZE_ORIGIN,
        VOCABULARY_ENDPOINT,
        commandAction,
        isValidVocabularyPayload,
        saveVocabulary,
    }
}
