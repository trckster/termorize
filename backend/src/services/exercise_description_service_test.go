package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescriptionMentionsAnswerUsesWordBoundariesAndIgnoresArticles(t *testing.T) {
	assert.False(t, descriptionMentionsAnswer("The object belongs to someone.", "he"))
	assert.False(t, descriptionMentionsAnswer("A thin material used for printing.", "paper"))
	assert.True(t, descriptionMentionsAnswer("You write on paper.", "paper"))
	assert.True(t, descriptionMentionsAnswer("You write words by hand.", "to write"))
	assert.True(t, descriptionMentionsAnswer("This clue accidentally says carta.", "la carta"))
	assert.True(t, descriptionMentionsAnswer("On les utilise pour couper : ciseaux.", "les ciseaux"))
	assert.True(t, descriptionMentionsAnswer("Ils sont encore jeunes : enfants.", "des enfants"))
	assert.True(t, descriptionMentionsAnswer("Игрушка висит на ёлке.", "ёлка"))
}
