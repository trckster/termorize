package jev

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"termorize/src/enums"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fakeClient(body string) *Client {
	c := NewClient("test-key")
	c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	return c
}

func TestRequestMatchesResearchedChoiceContract(t *testing.T) {
	golden, err := os.ReadFile("testdata/request.json")
	require.NoError(t, err)
	c := NewClient("test-key")
	c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, Endpoint, r.URL.String())
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, string(golden), string(body))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"answers":{"category":{"choice":"pronoun","confidence":0.99}}}`))}, nil
	})
	category, err := c.Classify(context.Background(), "io", enums.LanguageIt)
	require.NoError(t, err)
	assert.Equal(t, enums.PartOfSpeechPronoun, category)
}

func TestConfidenceAndMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       enums.PartOfSpeech
		invalid    bool
	}{
		{"boundary", `{"answers":{"category":{"choice":"noun","confidence":0.90}}}`, enums.PartOfSpeechNoun, false},
		{"below", `{"answers":{"category":{"choice":"noun","confidence":0.899999}}}`, enums.PartOfSpeechUnknown, false},
		{"zero", `{"answers":{"category":{"choice":"noun","confidence":0}}}`, enums.PartOfSpeechUnknown, false},
		{"unknown", `{"answers":{"category":{"choice":"unknown","confidence":1}}}`, enums.PartOfSpeechUnknown, false},
		{"missing answer", `{}`, "", true},
		{"null answer", `{"answers":{"category":null}}`, "", true},
		{"invalid choice", `{"answers":{"category":{"choice":"article","confidence":1}}}`, "", true},
		{"missing confidence", `{"answers":{"category":{"choice":"noun"}}}`, "", true},
		{"null confidence", `{"answers":{"category":{"choice":"noun","confidence":null}}}`, "", true},
		{"string confidence", `{"answers":{"category":{"choice":"noun","confidence":"0.99"}}}`, "", true},
		{"negative confidence", `{"answers":{"category":{"choice":"noun","confidence":-0.1}}}`, "", true},
		{"oversized confidence", `{"answers":{"category":{"choice":"noun","confidence":1.1}}}`, "", true},
		{"wrong confidence location", `{"confidence":1,"answers":{"category":{"choice":"noun"}}}`, "", true},
		{"trailing data", `{"answers":{"category":{"choice":"noun","confidence":1}}} {}`, "", true},
		{"invalid JSON", `not json`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fakeClient(tc.body).Classify(context.Background(), "book", enums.LanguageEn)
			if tc.invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRecordedPhraseAndSingleWordDecisions(t *testing.T) {
	data, err := os.ReadFile("testdata/recorded.json")
	require.NoError(t, err)
	var records []struct {
		Term, Language string
		Response       json.RawMessage
	}
	require.NoError(t, json.Unmarshal(data, &records))
	wants := map[string]enums.PartOfSpeech{
		"io": enums.PartOfSpeechPronoun, "spill the beans": enums.PartOfSpeechPhrase,
		"a piece of cake": enums.PartOfSpeechPhrase, "under the weather": enums.PartOfSpeechPhrase,
		"don't": enums.PartOfSpeechVerb, "well-known": enums.PartOfSpeechAdjective,
		"бы": enums.PartOfSpeechUnknown, "the": enums.PartOfSpeechUnknown,
	}
	for _, record := range records {
		t.Run(record.Term, func(t *testing.T) {
			var language enums.Language
			for _, candidate := range enums.AllLanguageValues() {
				if candidate.DisplayName() == record.Language {
					language = candidate
				}
			}
			got, err := fakeClient(string(record.Response)).Classify(context.Background(), record.Term, language)
			require.NoError(t, err)
			assert.Equal(t, wants[record.Term], got)
		})
	}
	assert.Len(t, records, len(wants))
}

func TestProviderAndContextFailures(t *testing.T) {
	for _, status := range []int{401, 429, 500, 529} {
		c := NewClient("test-key")
		c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("secret provider details"))}, nil
		})
		_, err := c.Classify(context.Background(), "book", enums.LanguageEn)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "secret")
	}
	c := NewClient("test-key")
	c.HTTP.Transport = transport(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Classify(ctx, "book", enums.LanguageEn)
	assert.True(t, errors.Is(err, context.Canceled))
	_, err = fakeClient(strings.Repeat("x", (1<<20)+1)).Classify(context.Background(), "book", enums.LanguageEn)
	require.ErrorContains(t, err, "size limit")
}
