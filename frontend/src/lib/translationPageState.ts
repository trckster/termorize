export type TranslationField = 'source' | 'target'
export type TranslationDirection = 'source-to-target' | 'target-to-source'

export const getLanguageChangeDirection = (field: TranslationField): TranslationDirection =>
    field === 'source' ? 'target-to-source' : 'source-to-target'

export const createProgrammaticChangeGuard = <Field extends string>() => {
    const pendingChanges = new Map<Field, string>()

    return {
        mark(field: Field, value: string) {
            pendingChanges.set(field, value)
        },
        consume(field: Field, value: string) {
            const pendingValue = pendingChanges.get(field)
            pendingChanges.delete(field)

            return pendingValue === value
        },
    }
}
