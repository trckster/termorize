const API_URL = import.meta.env.VITE_API_URL.replace(/\/$/, '')

export class PronunciationLimitError extends Error {
    constructor(public readonly retryAt: string) {
        super('Pronunciation spending limit reached')
    }
}

export const pronunciationApi = {
    async getWordAudio(wordId: string): Promise<Blob> {
        const response = await fetch(`${API_URL}/words/${encodeURIComponent(wordId)}/pronunciation`, {
            credentials: 'include',
            headers: { Accept: 'audio/mpeg' },
        })

        if (!response.ok) {
            if (response.status === 429) {
                const body = (await response.json().catch(() => null)) as { retry_at?: string } | null
                if (body?.retry_at) {
                    throw new PronunciationLimitError(body.retry_at)
                }
            }
            throw new Error(`Pronunciation request failed with status ${response.status}`)
        }

        return response.blob()
    },
}
