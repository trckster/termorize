export type TranslationAutofillRequest = {
    text: string
    sourceLanguage: string
    targetLanguage: string
}

type TranslationAutofillOptions = {
    translate: (request: TranslationAutofillRequest) => Promise<string>
    onSuggestion: (suggestion: string) => void
    onLoadingChange?: (isLoading: boolean) => void
    onError?: (error: unknown) => void
    debounceMs?: number
}

export const createTranslationAutofill = ({
    translate,
    onSuggestion,
    onLoadingChange,
    onError,
    debounceMs = 500,
}: TranslationAutofillOptions) => {
    let debounceTimer: ReturnType<typeof setTimeout> | null = null
    let requestVersion = 0
    let isActive = false
    let isTargetEdited = false
    let isLoading = false

    const setLoading = (nextValue: boolean) => {
        if (isLoading === nextValue) return

        isLoading = nextValue
        onLoadingChange?.(nextValue)
    }

    const invalidatePendingRequest = () => {
        requestVersion += 1
        if (debounceTimer) {
            clearTimeout(debounceTimer)
            debounceTimer = null
        }
        setLoading(false)
    }

    const activate = () => {
        invalidatePendingRequest()
        isActive = true
        isTargetEdited = false
    }

    const deactivate = () => {
        isActive = false
        invalidatePendingRequest()
    }

    const markTargetEdited = () => {
        isTargetEdited = true
        invalidatePendingRequest()
    }

    const queue = (request: TranslationAutofillRequest) => {
        if (!isActive || isTargetEdited) return

        invalidatePendingRequest()
        onSuggestion('')

        const text = request.text.trim()
        if (!text) return

        const currentVersion = requestVersion
        debounceTimer = setTimeout(async () => {
            debounceTimer = null
            if (!isActive || isTargetEdited || currentVersion !== requestVersion) return

            setLoading(true)
            try {
                const suggestion = await translate({ ...request, text })
                if (!isActive || isTargetEdited || currentVersion !== requestVersion) return

                onSuggestion(suggestion)
            } catch (error) {
                if (isActive && !isTargetEdited && currentVersion === requestVersion) {
                    onError?.(error)
                }
            } finally {
                if (currentVersion === requestVersion) {
                    setLoading(false)
                }
            }
        }, debounceMs)
    }

    return {
        activate,
        deactivate,
        markTargetEdited,
        queue,
    }
}
