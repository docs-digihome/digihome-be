package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

// Embed embeds text with the given model. If model == "" it resolves via
// viper key llm.embed.model (env LLM_EMBED_MODEL) falling back to DEFAULT_EMBED_MODEL.
// Endpoint is resolved via llm.embed.endpoint (env LLM_EMBED_ENDPOINT).
func Embed(ctx context.Context, model, text string) ([]float32, error) {
	if model == "" {
		model = EmbedModel()
	}
	endpoint := EmbedEndpoint()

	reqBody := OllamaEmbeddingRequest{
		Model: model,
		Input: text,
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
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding request failed with status %d", res.StatusCode)
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
