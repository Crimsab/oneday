package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crimsab/oneday/internal/config"
)

type ProviderStatus struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type Voice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Languages []string `json:"languages,omitempty"`
}

type SynthesisRequest struct {
	Text         string
	Model        string
	Voice        string
	Instructions string
	Speed        float64
	Format       string
}

type SynthesisResult struct {
	Audio      []byte
	Format     string
	DurationMS int64
	Timings    json.RawMessage
}

type Provider interface {
	ID() string
	Status(context.Context) ProviderStatus
	Voices(context.Context, string) ([]Voice, error)
	Synthesize(context.Context, SynthesisRequest) (SynthesisResult, error)
}

type CloudProvider struct {
	cfg    config.TTSEndpoint
	client *http.Client
}

func NewCloudProvider(cfg config.TTSEndpoint, timeout time.Duration) *CloudProvider {
	return &CloudProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (provider *CloudProvider) ID() string { return "cloud" }

func (provider *CloudProvider) Status(context.Context) ProviderStatus {
	if provider == nil || !provider.cfg.Enabled {
		return ProviderStatus{ID: "cloud", Reason: "disabled"}
	}
	if strings.TrimSpace(provider.cfg.BaseURL) == "" || strings.TrimSpace(provider.cfg.Model) == "" || strings.TrimSpace(provider.cfg.Voice) == "" {
		return ProviderStatus{ID: "cloud", Reason: "base URL, model, or voice is not configured"}
	}
	return ProviderStatus{ID: "cloud", Available: true}
}

func (provider *CloudProvider) Voices(_ context.Context, language string) ([]Voice, error) {
	status := provider.Status(context.Background())
	if !status.Available {
		return nil, errors.New(status.Reason)
	}
	voices := []string{"alloy", "ash", "ballad", "coral", "echo", "fable", "onyx", "nova", "sage", "shimmer", "verse", "marin", "cedar"}
	result := make([]Voice, 0, len(voices))
	for _, id := range voices {
		result = append(result, Voice{ID: id, Name: strings.ToUpper(id[:1]) + id[1:], Languages: nonEmptySlice(normalizeLanguageTag(language))})
	}
	return result, nil
}

func (provider *CloudProvider) Synthesize(ctx context.Context, request SynthesisRequest) (SynthesisResult, error) {
	status := provider.Status(ctx)
	if !status.Available {
		return SynthesisResult{}, fmt.Errorf("cloud TTS unavailable: %s", status.Reason)
	}
	if request.Model == "" {
		request.Model = provider.cfg.Model
	}
	if request.Voice == "" {
		request.Voice = provider.cfg.Voice
	}
	if request.Speed == 0 {
		request.Speed = 1
	}
	if request.Format == "" {
		request.Format = "mp3"
	}
	payload := map[string]any{"model": request.Model, "input": request.Text, "voice": request.Voice, "speed": request.Speed, "response_format": request.Format}
	if request.Instructions != "" {
		payload["instructions"] = request.Instructions
	}
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(provider.cfg.BaseURL, "/") + "/audio/speech"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return SynthesisResult{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if provider.cfg.APIKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+provider.cfg.APIKey)
	}
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return SynthesisResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return SynthesisResult{}, httpStatusError("cloud TTS", response)
	}
	audio, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return SynthesisResult{}, err
	}
	if len(audio) == 0 {
		return SynthesisResult{}, errors.New("cloud TTS returned empty audio")
	}
	return SynthesisResult{Audio: audio, Format: request.Format, Timings: json.RawMessage(`[]`)}, nil
}

type LocalProvider struct {
	cfg    config.TTSEndpoint
	client *http.Client
}

func NewLocalProvider(cfg config.TTSEndpoint, timeout time.Duration) *LocalProvider {
	return &LocalProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (provider *LocalProvider) ID() string { return "local" }

func (provider *LocalProvider) Status(ctx context.Context) ProviderStatus {
	if provider == nil || !provider.cfg.Enabled {
		return ProviderStatus{ID: "local", Reason: "disabled"}
	}
	if _, err := url.ParseRequestURI(provider.cfg.BaseURL); err != nil || strings.TrimSpace(provider.cfg.BaseURL) == "" {
		return ProviderStatus{ID: "local", Reason: "invalid base URL"}
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(provider.cfg.BaseURL, "/")+"/voices", nil)
	response, err := provider.client.Do(request)
	if err != nil {
		return ProviderStatus{ID: "local", Reason: "Piper service unreachable"}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ProviderStatus{ID: "local", Reason: fmt.Sprintf("Piper voices returned HTTP %d", response.StatusCode)}
	}
	return ProviderStatus{ID: "local", Available: true}
}

func (provider *LocalProvider) Voices(ctx context.Context, language string) ([]Voice, error) {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(provider.cfg.BaseURL, "/")+"/voices", nil)
	response, err := provider.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, httpStatusError("Piper voices", response)
	}
	var raw any
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&raw); err != nil {
		return nil, err
	}
	return parsePiperVoices(raw, language), nil
}

func (provider *LocalProvider) Synthesize(ctx context.Context, request SynthesisRequest) (SynthesisResult, error) {
	if !provider.cfg.Enabled {
		return SynthesisResult{}, errors.New("local TTS unavailable: disabled")
	}
	if request.Voice == "" {
		request.Voice = provider.cfg.Voice
	}
	if request.Speed == 0 {
		request.Speed = 1
	}
	lengthScale := 1 / request.Speed
	payload := map[string]any{"text": request.Text, "length_scale": lengthScale}
	if request.Voice != "" {
		payload["voice"] = request.Voice
	}
	body, _ := json.Marshal(payload)
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.cfg.BaseURL, "/")+"/synthesize", bytes.NewReader(body))
	if err != nil {
		return SynthesisResult{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		return SynthesisResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return SynthesisResult{}, httpStatusError("Piper synthesis", response)
	}
	audio, err := io.ReadAll(io.LimitReader(response.Body, 64<<20))
	if err != nil {
		return SynthesisResult{}, err
	}
	if len(audio) == 0 {
		return SynthesisResult{}, errors.New("Piper returned empty audio")
	}
	return SynthesisResult{Audio: audio, Format: "wav", Timings: json.RawMessage(`[]`)}, nil
}

func parsePiperVoices(raw any, language string) []Voice {
	result := []Voice{}
	appendVoice := func(id, name string) {
		if strings.TrimSpace(id) != "" {
			result = append(result, Voice{ID: id, Name: firstNonEmpty(name, id), Languages: nonEmptySlice(normalizeLanguageTag(language))})
		}
	}
	switch values := raw.(type) {
	case []any:
		for _, value := range values {
			switch item := value.(type) {
			case string:
				appendVoice(item, item)
			case map[string]any:
				appendVoice(stringValue(item["id"], item["key"], item["name"]), stringValue(item["name"], item["id"], item["key"]))
			}
		}
	case map[string]any:
		for id, value := range values {
			name := id
			if item, ok := value.(map[string]any); ok {
				name = stringValue(item["name"], id)
			}
			appendVoice(id, name)
		}
	}
	return result
}

func httpStatusError(label string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return fmt.Errorf("%s failed with HTTP %d: %s", label, response.StatusCode, message)
}

func nonEmptySlice(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func stringValue(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
