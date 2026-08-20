package pkg

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func DecodeAndValidateBody[T any](w http.ResponseWriter, r *http.Request, logger *slog.Logger) (T, int, error, bool) {
	var req T

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	
	if err := decoder.Decode(&req); err != nil {
		slog.Error("decode body request error", "error", err)
		return req, http.StatusBadRequest, errors.New("invalid body request"), false
	}

	if err := validate.Struct(req); err != nil {
		slog.Error("validation body requst error", "error", err)
		return req, http.StatusBadRequest, err, false
	}

	return req, 0, nil, true
}
