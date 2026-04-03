import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs))
}

function getPreferredLocale() {
    if (typeof document !== 'undefined' && document.documentElement.lang) {
        return document.documentElement.lang
    }

    if (typeof navigator !== 'undefined' && navigator.language) {
        return navigator.language
    }

    return 'en'
}

export function formatRelativeTime(dateString: string) {
    const date = new Date(dateString)

    if (Number.isNaN(date.getTime())) {
        return dateString
    }

    const now = new Date()
    const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    const rtf = new Intl.RelativeTimeFormat(getPreferredLocale(), { numeric: 'auto' })

    if (diffInSeconds < 60) {
        return rtf.format(0, 'second')
    }

    const diffInMinutes = Math.floor(diffInSeconds / 60)
    if (diffInMinutes < 60) {
        return rtf.format(-diffInMinutes, 'minute')
    }

    const diffInHours = Math.floor(diffInMinutes / 60)
    if (diffInHours < 24) {
        return rtf.format(-diffInHours, 'hour')
    }

    const diffInDays = Math.floor(diffInHours / 24)
    if (diffInDays < 30) {
        return rtf.format(-diffInDays, 'day')
    }

    const diffInMonths = Math.floor(diffInDays / 30)
    if (diffInMonths < 12) {
        return rtf.format(-diffInMonths, 'month')
    }

    const diffInYears = Math.floor(diffInDays / 365)
    return rtf.format(-diffInYears, 'year')
}

export function formatDate(dateString: string) {
    const date = new Date(dateString)

    if (Number.isNaN(date.getTime())) {
        return dateString
    }

    return new Intl.DateTimeFormat(getPreferredLocale(), {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    }).format(date)
}

export function formatNumber(value: number) {
    if (!Number.isFinite(value)) {
        return String(value)
    }

    return new Intl.NumberFormat(getPreferredLocale()).format(value)
}
