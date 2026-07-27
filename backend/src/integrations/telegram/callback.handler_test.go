package telegram

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseCallbackData(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		handlerType string
		payload     []string
		ok          bool
	}{
		{name: "exercise payload", data: "exercise:answer:id:option", handlerType: "exercise", payload: []string{"answer", "id", "option"}, ok: true},
		{name: "empty payload value", data: "menu:", handlerType: "menu", payload: []string{""}, ok: true},
		{name: "missing payload", data: "menu", ok: false},
		{name: "missing handler", data: ":action", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerType, payload, ok := parseCallbackData(test.data)

			require.Equal(t, test.ok, ok)
			require.Equal(t, test.handlerType, handlerType)
			require.Equal(t, test.payload, payload)
		})
	}
}

func TestParseCallbackUUIDSupportsStandardAndCompactValues(t *testing.T) {
	expected := uuid.MustParse("52fdfc07-2182-454f-963f-5f0f9a621d72")

	for _, value := range []string{expected.String(), compactCallbackUUID(expected)} {
		actual, err := parseCallbackUUID(value)

		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func TestShouldDeferCallbackAnswerOnlyForMatchTaps(t *testing.T) {
	validCallback := callbackQuery{
		From:    &user{ID: 1},
		Message: &message{MessageID: 2},
	}

	matchTap := validCallback
	matchTap.Data = callbackTypeExercise + ":" + exerciseActionMatchTap + ":exercise:0"
	require.True(t, shouldDeferCallbackAnswer(&matchTap))

	characterTap := validCallback
	characterTap.Data = callbackTypeExercise + ":" + exerciseActionCharacterTap + ":exercise:0"
	require.False(t, shouldDeferCallbackAnswer(&characterTap))

	matchTap.Message = nil
	require.False(t, shouldDeferCallbackAnswer(&matchTap))
}
