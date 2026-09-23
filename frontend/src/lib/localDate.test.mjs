import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { localDateInTimezone } from './localDate.ts'

describe('localDateInTimezone', () => {
    it('uses the account timezone on either side of midnight', () => {
        const now = new Date('2026-09-23T12:00:00Z')
        assert.equal(localDateInTimezone(now, 'Pacific/Kiritimati'), '2026-09-24')
        assert.equal(localDateInTimezone(now, 'America/Los_Angeles'), '2026-09-23')
        assert.equal(localDateInTimezone(new Date('2026-09-23T06:59:59Z'), 'America/Los_Angeles'), '2026-09-22')
        assert.equal(localDateInTimezone(new Date('2026-09-23T07:00:00Z'), 'America/Los_Angeles'), '2026-09-23')
    })
    it('pads single-digit months and days', () => {
        assert.equal(localDateInTimezone(new Date('2026-01-02T00:00:00Z'), 'UTC'), '2026-01-02')
    })
    it('matches the backend UTC fallback for missing or invalid timezones', () => {
        for (const zone of ['', 'invalid/timezone']) {
            assert.equal(localDateInTimezone(new Date('2026-09-23T23:59:00Z'), zone), '2026-09-23')
        }
    })
    it('handles midnight across daylight-saving changes', () => {
        assert.equal(localDateInTimezone(new Date('2026-03-28T23:00:00Z'), 'Europe/Rome'), '2026-03-29')
        assert.equal(localDateInTimezone(new Date('2026-03-29T22:00:00Z'), 'Europe/Rome'), '2026-03-30')
    })
})
