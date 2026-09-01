const POST_AUTH_PATH_KEY = 'termorize:post-auth-path'

const safeCollectionPath = (value: string): string | null => {
    if (!value.startsWith('/collections/') || value.startsWith('//')) {
        return null
    }

    try {
        const parsed = new URL(value, window.location.origin)
        if (parsed.origin !== window.location.origin || !parsed.pathname.startsWith('/collections/')) {
            return null
        }
        return `${parsed.pathname}${parsed.search}${parsed.hash}`
    } catch {
        return null
    }
}

export const rememberPostAuthPath = (path: string) => {
    const safePath = safeCollectionPath(path)
    if (safePath) {
        window.sessionStorage.setItem(POST_AUTH_PATH_KEY, safePath)
    }
}

export const clearPostAuthPath = () => {
    window.sessionStorage.removeItem(POST_AUTH_PATH_KEY)
}

export const takePostAuthPath = (): string | null => {
    const storedPath = window.sessionStorage.getItem(POST_AUTH_PATH_KEY)
    clearPostAuthPath()
    return storedPath ? safeCollectionPath(storedPath) : null
}
