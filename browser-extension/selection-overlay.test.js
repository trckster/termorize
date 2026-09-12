const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { it } = require('node:test')
const vm = require('node:vm')

function overlayHarness() {
    class Element {
        constructor() {
            this.style = {}
            this.dataset = {}
            this.value = ''
            this.hidden = false
            this.listeners = {}
            this.children = new Map()
            this.scrollHeight = 68
            this.offsetHeight = 68
            this.clientHeight = 66
        }
        querySelector(selector) {
            if (!this.children.has(selector)) this.children.set(selector, new Element())
            return this.children.get(selector)
        }
        attachShadow() { return this.shadow = new Element() }
        append() {}
        remove() {}
        replaceChildren() {}
        focus() {}
        getBoundingClientRect() { return { height: 300 } }
        addEventListener(type, fn) { (this.listeners[type] ||= []).push(fn) }
        emit(type, extra = {}) {
            for (const fn of this.listeners[type] || []) fn({ isTrusted: true, ...extra })
        }
    }
    const timers = new Map()
    const requests = []
    let nextTimer = 0
    let receive
    let host
    const windowApi = new Element()
    Object.assign(windowApi, {
        innerWidth: 1000, innerHeight: 800,
        setTimeout(fn) { timers.set(++nextTimer, fn); return nextTimer },
        clearTimeout(id) { timers.delete(id) },
    })
    vm.runInNewContext(readFileSync(require.resolve('./selection-overlay.js'), 'utf8'), {
        window: windowApi,
        document: {
            activeElement: null,
            documentElement: new Element(),
            createElement() { return host = new Element() },
        },
        ResizeObserver: class { observe() {} disconnect() {} },
        chrome: { runtime: {
            id: 'test',
            onMessage: { addListener(fn) { receive = fn } },
            sendMessage(message, reply) {
                if (message.type === 'GET_SESSION') {
                    reply({ ok: true, user: { settings: {} }, languages: [] })
                } else requests.push({ message, reply })
            },
        } },
    })
    function open(text = 'ciao') {
        receive({ type: 'OPEN_SELECTION_OVERLAY', selection: { ok: true, text } }, { id: 'test' }, () => {})
    }
    open()
    return {
        requests, timers,
        field(selector) { return host.shadow.querySelector(selector) },
        edit(text, extra) {
            const input = host.shadow.querySelector('.source')
            input.value = text
            input.emit('input', extra)
        },
        flush() {
            const pending = [...timers.values()]
            timers.clear()
            pending.forEach(fn => fn())
        },
        close() { windowApi.emit('pointerdown', { composedPath: () => [] }) },
        open,
    }
}
const settle = () => new Promise(resolve => setImmediate(resolve))
function respond(request, translated = 'hello') {
    request.reply({ ok: true, translation: {
        original: request.message.text, translated, originalLanguage: 'it', targetLanguage: 'ru', id: 'test',
    } })
}

it('debounces source edits and immediately prevents saving an outdated pair', async () => {
    const ui = overlayHarness()
    await settle()
    respond(ui.requests[0])
    await settle()
    ui.edit('buon')
    ui.edit('buongiorno')
    assert.equal(ui.requests.length, 1)
    assert.equal(ui.timers.size, 1)
    assert.equal(ui.field('.save').disabled, true)
    assert.notEqual(ui.field('.source').disabled, true)
    ui.flush()
    assert.equal(ui.requests[1].message.text, 'buongiorno')
    respond(ui.requests[1], 'good morning')
    await settle()
    assert.equal(ui.field('.translated').value, 'good morning')
    assert.equal(ui.field('.save').disabled, false)
})

it('ignores an older response during the debounce delay and after clearing the source', async () => {
    const ui = overlayHarness()
    await settle()
    ui.edit('new text')
    respond(ui.requests[0], 'stale')
    await settle()
    assert.equal(ui.field('.source').value, 'new text')
    assert.equal(ui.field('.translated').value, '')
    ui.flush()
    ui.edit('')
    respond(ui.requests[1], 'also stale')
    await settle()
    assert.equal(ui.field('.source').value, '')
    assert.equal(ui.field('.translated').value, '')
    assert.equal(ui.field('.save').disabled, true)
    assert.equal(ui.timers.size, 0)
})

it('waits until input composition finishes before requesting a translation', async () => {
    const ui = overlayHarness()
    await settle()
    ui.edit('日本', { isComposing: true })
    assert.equal(ui.timers.size, 0)
    ui.field('.source').emit('compositionend')
    ui.flush()
    assert.equal(ui.requests.length, 2)
    assert.equal(ui.requests[1].message.text, '日本')
})

it('cancels pending edits on dismissal and translates a newly opened selection', async () => {
    const ui = overlayHarness()
    await settle()
    ui.edit('pending')
    ui.close()
    ui.flush()
    assert.equal(ui.requests.length, 1)
    ui.open('new selection')
    await settle()
    assert.equal(ui.requests[1].message.text, 'new selection')
    respond(ui.requests[0], 'old overlay result')
    respond(ui.requests[1], 'new result')
    await settle()
    assert.equal(ui.field('.translated').value, 'new result')
})
