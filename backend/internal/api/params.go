package api

import (
	"net/http"
	"strconv"

	"stockflow/backend/internal/apierr"
)

func ParseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, apierr.BadRequest("id must be a positive integer", nil)
	}
	return id, nil
}
