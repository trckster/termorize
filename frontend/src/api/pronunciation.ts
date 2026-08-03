const API_URL = import.meta.env.VITE_API_URL.replace(/\/$/, '')

export const pronunciationApi = {
    async getWordAudio(wordId: string): Promise<Blob> {
        const response = await fetch(`${API_URL}/words/${encodeURIComponent(wordId)}/pronunciation`, {
            credentials: 'include',
            headers: { Accept: 'audio/mpeg' },
        })

        if (!response.ok) {
            throw new Error(`Pronunciation request failed with status ${response.status}`)
        }

        return response.blob()
    },
}
