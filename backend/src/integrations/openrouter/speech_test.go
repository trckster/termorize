package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpeechClientRequestsPCMAndReturnsEncodedMP3(t *testing.T) {
	pcmAudio := []byte{0x01, 0x00, 0x02, 0x00}
	expectedMP3 := []byte{0xff, 0xfb, 0x90, 0x64, 0x00, 0x01}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "https://termorize.test", r.Header.Get("HTTP-Referer"))
		require.Equal(t, "Termorize", r.Header.Get("X-Title"))

		var request speechRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, speechRequest{
			Model:          "google/gemini-3.1-flash-tts-preview",
			Input:          "buongiorno",
			Voice:          "Kore",
			ResponseFormat: "pcm",
		}, request)

		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(pcmAudio)
	}))
	defer server.Close()

	restoreSpeechURL(t, server.URL)
	client := &speechClient{
		apiKey:  "secret",
		model:   "google/gemini-3.1-flash-tts-preview",
		voice:   "Kore",
		format:  "pcm",
		referer: "https://termorize.test",
		http:    server.Client(),
		encodePCM: func(actual []byte) ([]byte, error) {
			require.Equal(t, pcmAudio, actual)
			return expectedMP3, nil
		},
	}

	audio, err := client.GenerateSpeech("buongiorno")

	require.NoError(t, err)
	require.Equal(t, expectedMP3, audio)
}

func TestSpeechClientReturnsMP3WithoutEncoding(t *testing.T) {
	expectedMP3 := []byte{0xff, 0xfb, 0x90, 0x64}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request speechRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "mp3", request.ResponseFormat)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(expectedMP3)
	}))
	defer server.Close()

	restoreSpeechURL(t, server.URL)
	client := &speechClient{
		apiKey: "secret",
		model:  "microsoft/mai-voice-2",
		voice:  "it-IT-Rosa:MAI-Voice-2",
		format: "mp3",
		http:   server.Client(),
		encodePCM: func([]byte) ([]byte, error) {
			t.Fatal("MP3 responses must not be encoded as PCM")
			return nil, nil
		},
	}

	audio, err := client.GenerateSpeech("radio")

	require.NoError(t, err)
	require.Equal(t, expectedMP3, audio)
}

func TestSpeechClientReturnsPCMEncodingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte{0x01, 0x00})
	}))
	defer server.Close()

	restoreSpeechURL(t, server.URL)
	client := &speechClient{
		apiKey: "secret",
		model:  "model",
		voice:  "voice",
		http:   server.Client(),
		encodePCM: func([]byte) ([]byte, error) {
			return nil, errors.New("encoder unavailable")
		},
	}

	_, err := client.GenerateSpeech("word")

	require.ErrorContains(t, err, "failed to encode")
	require.ErrorContains(t, err, "encoder unavailable")
}

func TestSpeechClientReturnsErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"unsupported voice"}}`)
	}))
	defer server.Close()

	restoreSpeechURL(t, server.URL)
	client := &speechClient{apiKey: "secret", model: "model", voice: "voice", http: server.Client()}

	_, err := client.GenerateSpeech("word")

	require.ErrorContains(t, err, "status 400")
	require.ErrorContains(t, err, "unsupported voice")
}

func TestSpeechClientReturnsTransportTimeout(t *testing.T) {
	client := &speechClient{
		apiKey: "secret",
		model:  "model",
		voice:  "voice",
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		})},
	}

	_, err := client.GenerateSpeech("word")

	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestSpeechClientRequiresAPIKey(t *testing.T) {
	client := &speechClient{http: http.DefaultClient}

	_, err := client.GenerateSpeech("word")

	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestSpeechClientRejectsEmptyAudio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	restoreSpeechURL(t, server.URL)
	client := &speechClient{apiKey: "secret", model: "model", voice: "voice", http: server.Client()}

	_, err := client.GenerateSpeech("word")

	require.ErrorContains(t, err, "empty audio")
}

func TestEncodePCMToMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed in the test environment")
	}

	// 100 ms of silent 24 kHz, 16-bit, mono PCM.
	pcm := make([]byte, 2400*2)
	mp3, err := encodePCMToMP3(pcm)

	require.NoError(t, err)
	require.NotEmpty(t, mp3)
	require.True(t,
		bytes.HasPrefix(mp3, []byte("ID3")) || (len(mp3) >= 2 && mp3[0] == 0xff && mp3[1]&0xe0 == 0xe0),
		"expected an ID3 tag or MPEG frame sync",
	)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func restoreSpeechURL(t *testing.T, url string) {
	t.Helper()
	previous := speechAPIURL
	speechAPIURL = url
	t.Cleanup(func() { speechAPIURL = previous })
}
