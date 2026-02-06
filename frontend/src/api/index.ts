let API_URL = import.meta.env.VITE_API_URL
if (API_URL.endsWith('/')) {
    API_URL.replace(/\/$/, '')
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

interface ApiResponse<T> {
    status: number
    body: T
}

export default async function apiCall<T>(
    url: string,
    method: HttpMethod = 'GET',
    data: any = {},
    headers: object = {}
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
    } else {
        fullUrl += '?' + new URLSearchParams(dataToSend as Record<string, string>)
    }

    const response = await fetch(fullUrl, payload)
    const json = await response.json().catch(() => '')

    if (response.status >= 500) {
        return Promise.reject({
            status: response.status,
            body: { message: 'Internal Server Error' },
        })
    }

    if (response.status >= 300) {
        return Promise.reject({
            status: response.status,
            body: json,
        })
    }

    return {
        status: response.status,
        body: json,
    }
}

function sanitizeData(data: any): object {
    const result = { ...data }

    for (const key in result) {
        if (result[key] === undefined) {
            delete result[key]
        }

        if (result[key] === true || result[key] === false) {
            result[key] = +result[key]
        }
    }

    return result
}
