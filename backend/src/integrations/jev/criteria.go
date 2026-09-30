package jev

var criteria = map[string]string{
	"noun":         "Single-word common or proper noun naming an entity, substance, place, person, or concept.",
	"verb":         "Single-word verb, auxiliary, or inflected verb form expressing an action, event, or state.",
	"adjective":    "Single-word adjective describing a noun or a property.",
	"adverb":       "Single-word adverb modifying a verb, adjective, adverb, or clause.",
	"pronoun":      "Single-word pronoun standing for a person, entity, or noun phrase.",
	"preposition":  "Single-word preposition introducing a following complement; excludes postpositions.",
	"conjunction":  "Single-word coordinating or subordinating conjunction connecting words or clauses.",
	"numeral":      "Single-word cardinal/ordinal numeral or numeric literal.",
	"interjection": "Single-word exclamation, greeting, or emotional utterance.",
	"phrase":       "Any meaningful term with multiple whitespace-separated words, regardless of grammatical role.",
	"unknown":      "No applicable supplied category, meaningless/nonlinguistic text, empty input, or insufficient evidence.",
}
