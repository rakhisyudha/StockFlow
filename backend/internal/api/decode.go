package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"stockflow/backend/internal/apierr"
)

const maxBodyBytes = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apierr.BadRequest("invalid JSON body: "+err.Error(), nil)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apierr.BadRequest("request body must contain a single JSON object", nil)
	}
	return nil
}
