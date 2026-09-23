export function localDateInTimezone(now: Date, timezone: string): string {
    let formatter: Intl.DateTimeFormat
    try {
        formatter = new Intl.DateTimeFormat('en-CA', { timeZone: timezone || 'UTC' })
    } catch {
        formatter = new Intl.DateTimeFormat('en-CA', { timeZone: 'UTC' })
    }
    const parts = formatter.formatToParts(now)
    const part = (type: string) => parts.find((value) => value.type === type)?.value
    return `${part('year')}-${part('month')}-${part('day')}`
}
