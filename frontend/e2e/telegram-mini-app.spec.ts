import { expect, test } from '@playwright/test'
import { createServer, request, type IncomingMessage, type ServerResponse } from 'node:http'
import type { AddressInfo } from 'node:net'

const initData = 'signed-telegram-launch-data'
const session = 'test-session'

const user = {
    id: 1,
    name: 'Test user',
    username: 'test_user',
    guest_expires_at: null,
    settings: {
        system_language: 'en',
        translation_source_language: 'en',
        translation_target_language: 'it',
        ignored_audio_languages: [],
        ignored_description_languages: [],
        telegram: { daily_questions_schedule: [] },
    },
}

const vocabulary = {
    data: [
        {
            id: 'saved-word',
            translation: {
                original: { id: 'word-one', word: 'forest', language: 'en', part_of_speech: 'noun' },
                translation: { id: 'word-two', word: 'foresta', language: 'it', part_of_speech: 'noun' },
            },
            progress: [],
            created_at: '2026-10-27T12:00:00Z',
            mastered_at: null,
        },
    ],
    pagination: { page: 1, page_size: 20, total: 1, total_pages: 1 },
}

const privateCollection = {
    id: 'private',
    title: 'Private vocabulary collection',
    is_admin: false,
    is_owner: true,
    is_published: false,
    languages: [],
    translation_count: 0,
    vocabulary_count: 0,
    user_add_count: 0,
    created_at: '2026-10-27T12:00:00Z',
    translations: [],
}

function json(response: ServerResponse, body: object) {
    response.setHeader('Content-Type', 'application/json')
    response.end(JSON.stringify(body))
}

async function handleAPI(req: IncomingMessage, response: ServerResponse) {
    switch (req.url?.split('?')[0]) {
        case '/api/telegram/login/callback': {
            let body = ''
            for await (const chunk of req) body += chunk
            const payload = JSON.parse(body)
            if (payload.init_data !== initData) return response.writeHead(401).end()
            if (payload.bearer_session) return json(response, { ...user, access_token: session })
            response.setHeader('Set-Cookie', `auth=${session}; Path=/; HttpOnly; SameSite=Strict`)
            return json(response, user)
        }
        case '/api/settings':
            return json(response, { languages: ['en', 'it'] })
        case '/api/public/collections/shared':
            return json(response, {
                ...privateCollection,
                id: 'shared',
                title: 'Shared vocabulary collection',
                is_owner: false,
                is_published: true,
                share_path: '/collections/shared',
            })
        case '/api/me':
        case '/api/vocabulary':
        case '/api/collections/private': {
            const hasBearer = req.headers.authorization === `Bearer ${session}`
            const hasCookie = req.headers.cookie === `auth=${session}`
            if (!hasCookie && !hasBearer) return response.writeHead(401).end()
            response.setHeader('X-Test-Session', hasBearer ? 'bearer' : 'cookie')
            if (req.url === '/api/me') return json(response, user)
            if (req.url === '/api/collections/private') return json(response, privateCollection)
            return json(response, vocabulary)
        }
        default:
            return response.writeHead(404).end()
    }
}

// Serve real HTTP responses for auth so both engines apply their cookie policy.
const server = createServer((req, response) => {
    const url = new URL(req.url ?? '/', 'http://localhost')
    if (url.pathname === '/telegram-shell') {
        const collection = url.searchParams.get('collection')
        const path = collection ? `/collections/${encodeURIComponent(collection)}` : '/'
        response.setHeader('Content-Type', 'text/html')
        return response.end(`<iframe src="${appURL}${path}" width="1200" height="900"></iframe>`)
    }
    if (req.url?.startsWith('/api/')) {
        void handleAPI(req, response).catch(() => response.writeHead(500).end())
        return
    }

    const proxy = request({ hostname: '127.0.0.1', port: 4173, path: req.url, method: req.method }, (upstream) => {
        response.writeHead(upstream.statusCode ?? 502, upstream.headers)
        upstream.pipe(response)
    })
    proxy.on('error', () => response.writeHead(502).end())
    req.pipe(proxy)
})

let appURL: string
let shellURL: string

test.beforeAll(async () => {
    await new Promise<void>((resolve, reject) => {
        server.once('error', reject)
        server.listen(0, '127.0.0.1', resolve)
    })
    const { port } = server.address() as AddressInfo
    appURL = `http://127.0.0.1:${port}`
    shellURL = `http://localhost:${port}/telegram-shell`
})

test.afterAll(async () => {
    await new Promise<void>((resolve, reject) => server.close((error) => (error ? reject(error) : resolve())))
})

for (const embedded of [true, false]) {
    test(`Telegram Mini App loads saved words ${embedded ? 'in a cross-site frame' : 'as a top-level page'}`, async ({
        page,
    }) => {
        await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
            route.fulfill({
                contentType: 'application/javascript',
                body: `window.Telegram = { WebApp: { initData: '${initData}' } }`,
            })
        )
        await page.goto(embedded ? shellURL : appURL)
        const app = embedded ? page.frameLocator('iframe') : page
        const [response] = await Promise.all([
            page.waitForResponse((res) => new URL(res.url()).pathname === '/api/vocabulary'),
            app.getByRole('link', { name: 'Vocabulary' }).click(),
        ])
        expect(response.status()).toBe(200)
        expect(response.headers()['x-test-session']).toBe(embedded ? 'bearer' : 'cookie')
        await expect(app.getByRole('heading', { name: 'Saved Words' })).toBeVisible()
        await expect(app.getByText('foresta', { exact: true })).toBeVisible()

        if (embedded) {
            const vocabularyURL = `${appURL}/vocabulary`
            await Promise.all([
                page.waitForEvent('framenavigated', (frame) => frame.url() === vocabularyURL),
                page.locator('iframe').evaluate((frame: HTMLIFrameElement, src) => (frame.src = src), vocabularyURL),
            ])
            await expect(app.getByText('foresta', { exact: true })).toBeVisible()
        }
    })
}

test('Telegram Mini App restores the session before opening a private collection', async ({ page }) => {
    await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
        route.fulfill({
            contentType: 'application/javascript',
            body: `window.Telegram = { WebApp: { initData: '${initData}' } }`,
        })
    )
    await page.goto(`${shellURL}?collection=private`)
    const app = page.frameLocator('iframe')
    await expect(app.getByRole('heading', { name: privateCollection.title })).toBeVisible()
    await expect(app.getByRole('button', { name: 'Edit title' })).toBeVisible()
})

test('Telegram Mini App can preview a shared collection when session restoration fails', async ({ page }) => {
    await page.route('https://telegram.org/js/telegram-web-app.js', (route) =>
        route.fulfill({
            contentType: 'application/javascript',
            body: "window.Telegram = { WebApp: { initData: 'expired-launch-data' } }",
        })
    )
    const [login] = await Promise.all([
        page.waitForResponse((response) => new URL(response.url()).pathname === '/api/telegram/login/callback'),
        page.goto(`${shellURL}?collection=shared`),
    ])
    expect(login.status()).toBe(401)
    await expect(
        page.frameLocator('iframe').getByRole('heading', { name: 'Shared vocabulary collection' })
    ).toBeVisible()
})
