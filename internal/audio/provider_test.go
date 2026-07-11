package audio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/crimsab/oneday/internal/config"
)

func TestCloudProviderUsesSpeechBinaryContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/audio/speech" || request.Header.Get("Authorization") != "Bearer virtual-key" {
			t.Fatalf("request path/auth = %s/%s", request.URL.Path, request.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-4o-mini-tts" || body["voice"] != "coral" || body["response_format"] != "opus" || body["instructions"] == "" {
			t.Fatalf("speech payload=%+v", body)
		}
		_, _ = writer.Write([]byte("opus-audio"))
	}))
	defer server.Close()
	provider := NewCloudProvider(config.TTSEndpoint{Enabled: true, BaseURL: server.URL + "/v1", APIKey: "virtual-key", Model: "gpt-4o-mini-tts", Voice: "alloy"}, time.Second)
	result, err := provider.Synthesize(context.Background(), SynthesisRequest{Text: "Ciao", Voice: "coral", Format: "opus", Instructions: "Warm tone", Speed: 1.1})
	if err != nil || string(result.Audio) != "opus-audio" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestLocalProviderUsesPersistentPiperHTTPContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/voices":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"it_IT-paola-medium":{"name":"Paola"}}`)
		case "/synthesize":
			var body map[string]any
			_ = json.NewDecoder(request.Body).Decode(&body)
			if body["text"] != "Lyanna" || body["voice"] != "it_IT-paola-medium" {
				t.Fatalf("piper payload=%+v", body)
			}
			_, _ = writer.Write([]byte("RIFF-wave"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	provider := NewLocalProvider(config.TTSEndpoint{Enabled: true, BaseURL: server.URL, Voice: "it_IT-paola-medium"}, time.Second)
	if status := provider.Status(context.Background()); !status.Available {
		t.Fatalf("status=%+v", status)
	}
	voices, err := provider.Voices(context.Background(), "it-IT")
	if err != nil || len(voices) != 1 || voices[0].Name != "Paola" {
		t.Fatalf("voices=%+v err=%v", voices, err)
	}
	result, err := provider.Synthesize(context.Background(), SynthesisRequest{Text: "Lyanna", Voice: "it_IT-paola-medium", Speed: 1})
	if err != nil || result.Format != "wav" || string(result.Audio) != "RIFF-wave" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
