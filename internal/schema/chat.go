package schema

import "time"

type (
	ChatRequest struct {
		TextPrompt string `json:"prompt" validate:"required"`
	}
)

type ChatDocument struct {
	Name string `json:"document_name"`
	Link string `json:"link"`
}

type (
	MessageResponse struct {
		Role      string        `json:"role"`
		Content   string        `json:"content"`
		CreatedAt time.Time     `json:"created_at"`
		Documents []ChatDocument `json:"documents,omitempty"`
	}
)
type (
	ChatResponse struct {
		Reply     string        `json:"reply"`
		Documents []ChatDocument `json:"documents,omitempty"`
	}
)
