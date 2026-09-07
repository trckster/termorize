import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { createPieSegments } from './pieChart.ts'

describe('pie geometry', () => {
    it('covers one full turn in the learning order, with exact shares', () => {
        const counts = [8, 10, 14, 8, 10]
        const segments = createPieSegments(counts)
        assert.equal(segments[0].startAngle, -Math.PI / 2)
        assert.equal(segments.at(-1).endAngle, Math.PI * 1.5)
        segments.forEach((segment, index) => {
            assert.equal(segment.share, counts[index] / 50)
            assert.ok(Math.abs(segment.endAngle - segment.startAngle - segment.share * Math.PI * 2) < 1e-12)
            if (index > 0) assert.equal(segment.startAngle, segments[index - 1].endAngle)
        })
    })

    it('retains tiny slices without minimum-angle inflation', () => {
        const segments = createPieSegments([1, 999999, 0, 0, 0])
        assert.equal(segments[0].share, 0.000001)
        assert.notEqual(segments[0].path, '')
        assert.ok(segments[0].endAngle - segments[0].startAngle < 0.00001)
        assert.equal(segments[2].path, '')
    })

    it('renders a single populated range as a closed full circle', () => {
        for (let index = 0; index < 5; index++) {
            const counts = Array(5).fill(0)
            counts[index] = 50
            const segments = createPieSegments(counts)
            assert.equal(segments.filter((segment) => segment.path).length, 1)
            assert.equal((segments[index].path.match(/ A /g) ?? []).length, 2)
            assert.equal(segments[index].share, 1)
            assert.ok(segments[index].path.endsWith('Z'))
        }
    })

    it('keeps an empty dataset empty rather than fabricating equal slices', () => {
        for (const segment of createPieSegments([0, 0, 0, 0, 0])) {
            assert.equal(segment.path, '')
            assert.equal(segment.share, 0)
            assert.ok(Number.isFinite(segment.anchor.x))
        }
        assert.deepEqual(createPieSegments([]), [])
    })

    it('does not create broken SVG paths from invalid counts', () => {
        const segments = createPieSegments([NaN, Infinity, -1, 2, 0])
        assert.deepEqual(
            segments.map((segment) => segment.count),
            [0, 0, 0, 2, 0]
        )
        assert.equal(segments[3].share, 1)
    })
})
