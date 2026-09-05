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

const elements = {
    loading: document.querySelector('#loading-state'),
    signedOut: document.querySelector('#signed-out-state'),
    workspace: document.querySelector('#workspace'),
    accountName: document.querySelector('#account-name'),
    targetLanguage: document.querySelector('#target-language'),
    empty: document.querySelector('#selection-empty'),
    form: document.querySelector('#translation-form'),
    sourceText: document.querySelector('#source-text'),
    translatedText: document.querySelector('#translated-text'),
    sourceLanguage: document.querySelector('#source-language'),
    translatedLanguage: document.querySelector('#translated-language'),
    message: document.querySelector('#message'),
    retry: document.querySelector('#retry-translation'),
    save: document.querySelector('#save-translation'),
    signIn: document.querySelector('#sign-in'),
    checkSession: document.querySelector('#check-session'),
    openApp: document.querySelector('#open-app'),
}

let currentTranslation = null
let latestRequest = 0

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

function setMessage(text = '', variant = '') {
    elements.message.textContent = text
    if (variant) elements.message.dataset.variant = variant
    else delete elements.message.dataset.variant
}

function languageName(code) {
    return LANGUAGE_NAMES[code] || String(code || '').toUpperCase()
}

function setBusy(busy, label = 'Translating…') {
    elements.targetLanguage.disabled = busy
    elements.retry.disabled = busy
    elements.sourceText.disabled = busy
    elements.translatedText.disabled = busy
    elements.save.disabled = busy || !currentTranslation
    if (busy) {
        elements.translatedText.value = ''
        elements.translatedText.placeholder = label
        setMessage(label)
    } else {
        elements.translatedText.placeholder = ''
    }
}

function setSaving(saving) {
    elements.targetLanguage.disabled = saving
    elements.retry.disabled = saving
    elements.sourceText.disabled = saving
    elements.translatedText.disabled = saving
    elements.save.disabled = saving || !currentTranslation
}

function showSignedOut() {
    elements.loading.hidden = true
    elements.workspace.hidden = true
    elements.signedOut.hidden = false
}

function fillLanguageOptions(languages, selected) {
    elements.targetLanguage.replaceChildren()
    for (const code of languages) {
        const option = document.createElement('option')
        option.value = code
        option.textContent = languageName(code)
        option.selected = code === selected
        elements.targetLanguage.append(option)
    }
}

function showWorkspace(session, selection) {
    elements.loading.hidden = true
    elements.signedOut.hidden = true
    elements.workspace.hidden = false
    elements.accountName.textContent = `Signed in as ${session.user.name}`

    const target = session.user.settings.translation_target_language || 'ru'
    fillLanguageOptions(session.languages, target)

    if (!selection.ok) {
        elements.empty.hidden = false
        elements.form.hidden = true
        if (selection.reason === 'too-long') {
            elements.empty.querySelector('h2').textContent = 'Selection is too long'
            elements.empty.querySelector('p').textContent = 'Select up to 5,000 characters and open TermoClip again.'
        } else if (selection.reason === 'frame-unavailable') {
            elements.empty.querySelector('h2').textContent = 'Embedded selection'
            elements.empty.querySelector('p').textContent =
                'Right-click the selected text and choose “Translate selection with TermoClip.”'
        }
        return
    }

    elements.empty.hidden = true
    elements.form.hidden = false
    elements.sourceText.value = selection.text
    void translate()
}

function translationErrorMessage(response) {
    if (response.reason === 'same-language') {
        return `This text is already ${languageName(response.detectedLanguage)}. Choose another target language.`
    }
    if (response.reason === 'unsupported-language') {
        return `Detected ${languageName(response.detectedLanguage)}, which Termorize does not support yet.`
    }
    if (response.reason === 'network') return 'Could not reach Termorize. Check your connection and try again.'
    if (response.reason === 'unauthorized') return 'Your session expired. Sign in again to continue.'
    return 'Termorize could not translate this selection. Try again in a moment.'
}

async function translate() {
    const source = elements.sourceText.value.trim()
    if (!source) {
        currentTranslation = null
        elements.sourceLanguage.textContent = 'Auto-detected'
        elements.translatedText.value = ''
        elements.save.disabled = true
        setMessage('Enter or select text to translate.', 'warning')
        return
    }

    const requestId = ++latestRequest
    currentTranslation = null
    elements.sourceLanguage.textContent = 'Detecting…'
    elements.translatedLanguage.textContent = languageName(elements.targetLanguage.value)
    setBusy(true)

    const response = await runtimeMessage({
        type: 'TRANSLATE_SELECTION',
        text: source,
        targetLanguage: elements.targetLanguage.value,
    })

    if (requestId !== latestRequest) return
    setBusy(false)

    if (!response.ok) {
        elements.sourceLanguage.textContent = response.detectedLanguage
            ? languageName(response.detectedLanguage)
            : 'Auto-detected'
        elements.translatedText.value = ''
        setMessage(translationErrorMessage(response), response.reason === 'same-language' ? 'warning' : 'error')
        if (response.reason === 'unauthorized') showSignedOut()
        return
    }

    currentTranslation = response.translation
    elements.sourceText.value = response.translation.original
    elements.translatedText.value = response.translation.translated
    elements.sourceLanguage.textContent = languageName(response.translation.originalLanguage)
    elements.translatedLanguage.textContent = languageName(response.translation.targetLanguage)
    elements.save.disabled = false
    setMessage('Ready to save. You can edit either field first.')
}

async function save(event) {
    event.preventDefault()
    if (!currentTranslation) return

    const original = elements.sourceText.value.trim()
    const translated = elements.translatedText.value.trim()
    if (!original || !translated) {
        setMessage('Both fields are required before saving.', 'warning')
        return
    }

    const edited = original !== currentTranslation.original || translated !== currentTranslation.translated
    setSaving(true)
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

    setSaving(false)
    elements.save.textContent = response.ok ? 'Saved' : 'Save to vocabulary'
    elements.save.disabled = response.ok

    if (response.ok) {
        setMessage('Saved to your Termorize vocabulary.', 'success')
    } else if (response.reason === 'duplicate') {
        setMessage('This word pair is already in your vocabulary.', 'warning')
    } else if (response.reason === 'unauthorized') {
        showSignedOut()
    } else {
        setMessage(
            response.reason === 'network' ? 'Connection failed. Try again.' : 'Could not save this word pair.',
            'error'
        )
        elements.save.disabled = false
    }
}

async function changeTargetLanguage() {
    const targetLanguage = elements.targetLanguage.value
    currentTranslation = null
    elements.save.disabled = true
    elements.save.textContent = 'Save to vocabulary'

    const [settingsResponse] = await Promise.all([
        runtimeMessage({ type: 'UPDATE_TARGET_LANGUAGE', targetLanguage }),
        translate(),
    ])

    if (!settingsResponse.ok && settingsResponse.reason === 'unauthorized') showSignedOut()
    else if (!settingsResponse.ok)
        setMessage('Translation updated, but the language preference was not saved.', 'warning')
}

async function initialize() {
    elements.loading.hidden = false
    elements.signedOut.hidden = true
    elements.workspace.hidden = true
    const [session, selection] = await Promise.all([
        runtimeMessage({ type: 'GET_SESSION' }),
        runtimeMessage({ type: 'GET_ACTIVE_SELECTION' }),
    ])

    if (!session.ok) {
        showSignedOut()
        return
    }

    showWorkspace(session, selection)
}

elements.form.addEventListener('submit', save)
elements.targetLanguage.addEventListener('change', changeTargetLanguage)
elements.retry.addEventListener('click', translate)
elements.signIn.addEventListener('click', () => void runtimeMessage({ type: 'OPEN_TERMORIZE' }))
elements.checkSession.addEventListener('click', initialize)
elements.openApp.addEventListener('click', () => void runtimeMessage({ type: 'OPEN_TERMORIZE' }))

void initialize()
