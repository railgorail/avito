package sl

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"railgorail/avito/internal/transport/http/dto"

	"github.com/stretchr/testify/assert"
)

func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func DecodeErrorResponse(t *testing.T, body *bytes.Buffer) dto.ErrorResponse {
	var resp dto.ErrorResponse
	err := json.NewDecoder(body).Decode(&resp)
	assert.NoError(t, err)
	return resp
}
