import type { Plugin } from 'vite'

/** Opt-in, in-memory sample API for local design review. Never used in a build. */
export function designPreview(): Plugin {
    const languages = ['en', 'ru', 'it', 'de', 'es', 'fr', 'pl', 'tr', 'pt', 'uk']
    const user = {
        id: 1,
        name: 'Design preview',
        username: 'preview',
        is_admin: false,
        guest_expires_at: null,
        created_at: '2026-01-01T12:00:00Z',
        settings: {
            system_language: 'en',
            main_learning_language: 'en',
            translation_source_language: 'en',
            translation_target_language: 'it',
            time_zone: 'Europe/Rome',
            ignored_audio_languages: [],
            ignored_description_languages: [],
            telegram: {
                bot_enabled: true,
                daily_idiom_enabled: false,
                daily_questions_enabled: false,
                daily_questions_count: 10,
                daily_questions_schedule: [],
            },
        },
    }
    const samples: Record<string, [string, string]> = {
        en: ['break the ice', 'To make people feel more relaxed in a new situation.'],
        it: ['rompere il ghiaccio', 'Mettere le persone a proprio agio in una situazione nuova.'],
        ru: ['растопить лёд', 'Помочь людям почувствовать себя свободнее в новой обстановке.'],
    }
    return {
        name: 'local-design-preview',
        apply: 'serve',
        configureServer(server) {
            server.middlewares.use('/api', async (req, res) => {
                res.setHeader('Content-Type', 'application/json')
                const path = req.url?.split('?')[0]
                let data: unknown
                if (path === '/me' || path === '/guest/login') data = user
                else if (path === '/settings' && req.method === 'PUT') {
                    try {
                        let body = ''
                        for await (const chunk of req) body += chunk
                        user.settings = JSON.parse(body)
                        data = user
                    } catch {
                        res.statusCode = 400
                        data = { message: 'Invalid preview settings.' }
                    }
                } else if (path === '/settings') data = { languages }
                else if (path === '/daily-idiom') {
                    const language = user.settings.main_learning_language
                    const sample = samples[language]
                    data = {
                        date: new Date().toISOString().slice(0, 10),
                        language,
                        idiom: sample ? { id: 'preview-idiom', word_id: 'preview-idiom', word: sample[0] } : null,
                    }
                } else if (path === '/daily-idiom/preview-idiom/description') {
                    data = { description: samples[user.settings.main_learning_language]?.[1] ?? '' }
                } else {
                    res.statusCode = 404
                    data = { message: 'This action is not available with the local sample data.' }
                }
                res.end(JSON.stringify(data))
            })
        },
    }
}
