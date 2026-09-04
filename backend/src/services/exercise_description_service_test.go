package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDescriptionMentionsAnswerUsesWordBoundariesAndIgnoresArticles(t *testing.T) {
	assert.False(t, descriptionMentionsAnswer("The object belongs to someone.", "he"))
	assert.True(t, descriptionMentionsAnswer("You write on paper.", "paper"))
	assert.True(t, descriptionMentionsAnswer("This clue accidentally says carta.", "la carta"))
	assert.True(t, descriptionMentionsAnswer("Эта подсказка называет ёлку.", "ёлку"))
}
