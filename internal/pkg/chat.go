package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OllamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Think    bool          `json:"think"`
	Options  struct {
		NumCtx     int `json:"num_ctx"`
		NumPredict int `json:"num_predict"`
	} `json:"options"`
}

type OllamaChatResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

type OpenAIChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Stream    bool          `json:"stream,omitempty"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func ChatAPIKey() string {
	if v := viper.GetString("llm.chat.api_key"); v != "" {
		return v
	}
	if v := viper.GetString("llm.api_key"); v != "" {
		return v
	}
	return ""
}

func isOpenAIChatEndpoint(endpoint, apiKey string) bool {
	if apiKey != "" {
		return true
	}
	if v := viper.GetString("llm.chat.provider"); strings.EqualFold(v, "openai") {
		return true
	}
	if strings.Contains(endpoint, "/v1/chat/completions") || strings.Contains(endpoint, "/v1/") && strings.Contains(endpoint, "chat") {
		return true
	}
	if strings.Contains(endpoint, "openai") {
		return true
	}
	return false
}

func Chat(ctx context.Context, endpoint, model string, messages []ChatMessage, numCtx, maxTokens int) (string, error) {
	apiKey := ChatAPIKey()
	return ChatWithAPIKey(ctx, endpoint, model, apiKey, messages, numCtx, maxTokens)
}

// ChatWithAPIKey is like Chat but allows the caller to supply an explicit API key.
// If apiKey == "" it falls back to ChatAPIKey() (viper llm.chat.api_key / llm.api_key).
// When an API key is present or endpoint looks like an OpenAI-compatible URL, the
// OpenAI scheme (POST /v1/chat/completions with Bearer auth, response {choices:[{message}]})
// is used; otherwise Ollama scheme (POST /api/chat with options, response {message,done}) is used.
func ChatWithAPIKey(ctx context.Context, endpoint, model, apiKey string, messages []ChatMessage, numCtx, maxTokens int) (string, error) {
	if apiKey == "" {
		apiKey = ChatAPIKey()
	}
	useOpenAI := isOpenAIChatEndpoint(endpoint, apiKey)

	var body []byte
	var err error
	if useOpenAI {
		reqBody := OpenAIChatRequest{
			Model:     model,
			Messages:  messages,
			Stream:    false,
			MaxTokens: maxTokens,
		}
		body, err = json.Marshal(reqBody)
	} else {
		reqBody := OllamaChatRequest{
			Model:    model,
			Messages: messages,
			Stream:   false,
		}
		reqBody.Options.NumCtx = numCtx
		reqBody.Options.NumPredict = maxTokens
		body, err = json.Marshal(reqBody)
	}
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("chat request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}

	if useOpenAI {
		var result OpenAIChatResponse
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return "", err
		}
		if len(result.Choices) == 0 {
			return "", fmt.Errorf("chat response contains no choices")
		}
		return result.Choices[0].Message.Content, nil
	}

	var result OllamaChatResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Message.Content, nil
}
