export type TranslationField = 'source' | 'target'
export type TranslationDirection = 'source-to-target' | 'target-to-source'

export type EditableVocabularyPair = {
    original: string
    translation: string
}

export const getLanguageChangeDirection = (field: TranslationField): TranslationDirection =>
    field === 'source' ? 'target-to-source' : 'source-to-target'

export const hasMeaningfulVocabularyEdits = (initialPair: EditableVocabularyPair, editedPair: EditableVocabularyPair) =>
    initialPair.original.trim() !== editedPair.original.trim() ||
    initialPair.translation.trim() !== editedPair.translation.trim()

export const isEditableVocabularyShortcut = (
    event: Pick<KeyboardEvent, 'altKey' | 'code' | 'ctrlKey' | 'metaKey' | 'shiftKey'>
) => event.ctrlKey && !event.altKey && !event.metaKey && !event.shiftKey && event.code === 'KeyE'

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
