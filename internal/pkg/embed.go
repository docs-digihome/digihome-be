package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/daffadon/digihome/internal/constant"
	"github.com/spf13/viper"
)

type OllamaEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type OllamaEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type OpenAIEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Object    string    `json:"object"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func EmbedEndpoint() string {
	if v := viper.GetString("llm.embed.endpoint"); v != "" {
		return v
	}
	return constant.DEFAULT_EMBED_ENDOPINT
}

func EmbedModel() string {
	if v := viper.GetString("llm.embed.model"); v != "" {
		return v
	}
	return constant.DEFAULT_EMBED_MODEL
}

func EmbedAPIKey() string {
	if v := viper.GetString("llm.embed.api_key"); v != "" {
		return v
	}
	if v := viper.GetString("llm.api_key"); v != "" {
		return v
	}
	return ""
}

func isOpenAIEmbedEndpoint(endpoint, apiKey string) bool {
	if apiKey != "" {
		return true
	}
	if v := viper.GetString("llm.embed.provider"); strings.EqualFold(v, "openai") {
		return true
	}
	if strings.Contains(endpoint, "/v1/embeddings") || strings.Contains(endpoint, "/v1/") && strings.Contains(endpoint, "embed") {
		return true
	}
	// generic OpenAI detection: /v1/ prefix with api_key-less hosted OpenAI-compatible gateways
	if strings.Contains(endpoint, "openai") {
		return true
	}
	return false
}

// Embed embeds text with the given model. If model == "" it resolves via
// viper key llm.embed.model (env LLM_EMBED_MODEL) falling back to DEFAULT_EMBED_MODEL.
// Endpoint is resolved via llm.embed.endpoint (env LLM_EMBED_ENDPOINT).
// API key is resolved via llm.embed.api_key (env LLM_EMBED_API_KEY) falling back to llm.api_key.
// When an API key is present or endpoint looks like an OpenAI-compatible URL, the OpenAI
// scheme (POST /v1/embeddings with Bearer auth, response {data:[{embedding}]}) is used;
// otherwise Ollama scheme (POST /api/embed, response {embeddings:[[float]]}) is used.
func Embed(ctx context.Context, model, text string) ([]float32, error) {
	if model == "" {
		model = EmbedModel()
	}
	endpoint := EmbedEndpoint()
	apiKey := EmbedAPIKey()

	useOpenAI := isOpenAIEmbedEndpoint(endpoint, apiKey)

	var reqBody any
	if useOpenAI {
		reqBody = OpenAIEmbeddingRequest{
			Model: model,
			Input: text,
		}
	} else {
		reqBody = OllamaEmbeddingRequest{
			Model: model,
			Input: text,
		}
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("embedding request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}

	if useOpenAI {
		var result OpenAIEmbeddingResponse
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return nil, err
		}
		if len(result.Data) == 0 {
			return nil, fmt.Errorf("embedding response contains no embeddings")
		}
		return result.Data[0].Embedding, nil
	}

	var result OllamaEmbeddingResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("embedding response contains no embeddings")
	}

	return result.Embeddings[0], nil
}
