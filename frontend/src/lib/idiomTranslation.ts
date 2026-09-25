export type ActiveIdiom = { id: string; word: string; language: string }

export function idiomLanguages(language: string, previousSource: string, previousTarget: string) {
    return { source: language, target: previousSource === language ? previousTarget : previousSource }
}

export function matchesIdiom(idiom: ActiveIdiom, text: string, language: string) {
    return idiom.language === language && idiom.word === text
}
