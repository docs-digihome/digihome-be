package schema

import "time"

type (
	ChatRequest struct {
		TextPrompt string `json:"prompt" validate:"required"`
	}
)

type (
	MessageResponse struct {
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		Documents []string  `json:"documents,omitempty"`
	}
)
type (
	ChatResponse struct {
		Reply     string   `json:"reply"`
		Documents []string `json:"documents,omitempty"`
	}
)
