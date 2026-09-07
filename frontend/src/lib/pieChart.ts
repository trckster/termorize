// Keep sectors proportional: no minimum angles or fixed padding that would
// exaggerate small groups. The full-circle case needs two SVG arcs.
export function createPieSegments(values: number[], radius = 124, center = 140) {
    const counts = values.map((value) => (Number.isFinite(value) ? Math.max(0, value) : 0))
    const total = counts.reduce((sum, count) => sum + count, 0)
    let cumulative = 0

    return counts.map((count) => {
        const startAngle = total ? (cumulative / total) * Math.PI * 2 - Math.PI / 2 : -Math.PI / 2
        cumulative += count
        const endAngle = total ? (cumulative / total) * Math.PI * 2 - Math.PI / 2 : startAngle
        const middle = (startAngle + endAngle) / 2
        const point = (angle: number, distance = radius) => ({
            x: center + distance * Math.cos(angle),
            y: center + distance * Math.sin(angle),
        })
        const start = point(startAngle)
        const end = point(endAngle)
        const path =
            count === 0
                ? ''
                : count === total
                  ? `M ${center} ${center - radius} A ${radius} ${radius} 0 1 1 ${center} ${center + radius} A ${radius} ${radius} 0 1 1 ${center} ${center - radius} Z`
                  : `M ${center} ${center} L ${start.x} ${start.y} A ${radius} ${radius} 0 ${endAngle - startAngle > Math.PI ? 1 : 0} 1 ${end.x} ${end.y} Z`
        return {
            count,
            share: total ? count / total : 0,
            startAngle,
            endAngle,
            path,
            anchor: point(middle, radius * 0.68),
        }
    })
}
