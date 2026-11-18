const API_URL = import.meta.env.VITE_API_URL.replace(/\/$/, '')

// Keep framed Mini App sessions in memory for the lifetime of the document.
let accessToken: string | undefined

export function setApiAccessToken(token?: string): void {
    accessToken = token
}

export function hasApiAccessToken(): boolean {
    return Boolean(accessToken)
}

export function apiFetch(path: string, options: RequestInit): Promise<Response> {
    const headers = new Headers(options.headers)
    if (accessToken) {
        headers.set('Authorization', `Bearer ${accessToken}`)
    }

    return fetch(`${API_URL}${path}`, { credentials: 'include', ...options, headers })
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

interface ApiResponse<T> {
    status: number
    body: T
}

export const unwrapBody = <T>(response: ApiResponse<T>): T => response.body

export default async function apiCall<T>(
    url: string,
    method: HttpMethod = 'GET',
    data: object = {},
    headers: HeadersInit = {}
): Promise<ApiResponse<T>> {
    const payload: RequestInit = {
        method: method,
        headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
            ...headers,
        },
    }

    const dataToSend = sanitizeData(data)

    if (method !== 'GET') {
        payload.body = JSON.stringify(dataToSend)
    } else if (Object.keys(dataToSend).length > 0) {
        url +=
            '?' +
            new URLSearchParams(
                Object.entries(dataToSend).reduce<Record<string, string>>((acc, [key, value]) => {
                    acc[key] = String(value)
                    return acc
                }, {})
            )
    }

    let response: Response

    try {
        response = await apiFetch(url, payload)
    } catch (error) {
        return Promise.reject({
            status: 0,
            body: {
                message: error instanceof Error ? error.message : 'Network request failed',
            },
        })
    }

    const contentType = response.headers.get('content-type') || ''
    const hasJsonBody = contentType.includes('application/json')
    const json = hasJsonBody ? await response.json().catch(() => null) : null

    if (response.status >= 500) {
        return Promise.reject({
            status: response.status,
            body: json || { message: 'Internal Server Error' },
        })
    }

    if (response.status >= 300) {
        return Promise.reject({
            status: response.status,
            body: json || { message: response.statusText || 'Request failed' },
        })
    }

    return {
        status: response.status,
        body: (json ?? null) as T,
    }
}

function sanitizeData(data: object): Record<string, unknown> {
    const result: Record<string, unknown> = {}

    for (const [key, value] of Object.entries(data)) {
        if (value === undefined) {
            continue
        }

        result[key] = value
    }

    return result
}
