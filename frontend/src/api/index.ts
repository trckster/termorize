const API_URL = import.meta.env.VITE_API_URL.replace(/\/$/, '')

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
        credentials: 'include',
        headers: {
            Accept: 'application/json',
            'Content-Type': 'application/json',
            ...headers,
        },
    }

    let fullUrl = `${API_URL}${url}`

    const dataToSend = sanitizeData(data)

    if (method !== 'GET') {
        payload.body = JSON.stringify(dataToSend)
    } else if (Object.keys(dataToSend).length > 0) {
        fullUrl +=
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
        response = await fetch(fullUrl, payload)
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

        result[key] = typeof value === 'boolean' ? +value : value
    }

    return result
}
