package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"railgorail/avito/internal/transport/http/dto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func DecodeErrorResponse(t *testing.T, body *bytes.Buffer) dto.ErrorResponse {
	var resp dto.ErrorResponse
	err := json.NewDecoder(body).Decode(&resp)
	assert.NoError(t, err)
	return resp
}
