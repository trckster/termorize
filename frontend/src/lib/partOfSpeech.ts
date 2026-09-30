export const partsOfSpeech = [
    'noun',
    'verb',
    'adjective',
    'adverb',
    'pronoun',
    'preposition',
    'conjunction',
    'numeral',
    'interjection',
    'phrase',
    'unknown',
] as const

export type PartOfSpeech = (typeof partsOfSpeech)[number]

const labels = {
    en: {
        noun: 'Noun',
        verb: 'Verb',
        adjective: 'Adjective',
        adverb: 'Adverb',
        pronoun: 'Pronoun',
        preposition: 'Preposition',
        conjunction: 'Conjunction',
        numeral: 'Numeral',
        interjection: 'Interjection',
        phrase: 'Phrase',
        unknown: 'Unknown',
        pending: 'Pending',
    },
    ru: {
        noun: 'Существительное',
        verb: 'Глагол',
        adjective: 'Прилагательное',
        adverb: 'Наречие',
        pronoun: 'Местоимение',
        preposition: 'Предлог',
        conjunction: 'Союз',
        numeral: 'Числительное',
        interjection: 'Междометие',
        phrase: 'Фраза',
        unknown: 'Не определено',
        pending: 'Ожидает определения',
    },
}

export function partOfSpeechLabel(value: PartOfSpeech | null, locale: 'en' | 'ru'): string {
    return labels[locale][value ?? 'pending']
}
