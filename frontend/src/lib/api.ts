let API_URL = import.meta.env.VITE_API_URL
if (API_URL.endsWith('/')) {
    API_URL.replace(/\/$/, '')
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

interface ApiResponse<T> {
    status: number
    body: T
}

async function apiCall<T>(
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

export interface TelegramAuthData {
    id: number
    auth_date: number
    username: string
    first_name: string
    last_name: string
    photo_url: string
    hash: string
}

export interface User {
    id: number
    username: string
    name: string
    photo_url: string
    created_at: string
}

export const authApi = {
    async login(authData: TelegramAuthData): Promise<User | null> {
        const response = await apiCall<User>('/telegram/login', 'POST', authData)

        return response.body
    },

    async getCurrentUser(): Promise<User | null> {
        return await apiCall<User>('/me').then((r) => r.body)
    },

    async logout(): Promise<void> {
        await apiCall<void>('/logout', 'POST')
    },
}
