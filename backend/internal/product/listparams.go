package product

import (
	"fmt"
	"net/url"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

func ParseListParams(q url.Values) (ListParams, error) {
	p := ListParams{Search: q.Get("search"), Page: 1, Limit: defaultLimit}

	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return p, fmt.Errorf("page must be an integer greater than 0")
		}
		p.Page = n
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxLimit {
			return p, fmt.Errorf("limit must be an integer between 1 and %d", maxLimit)
		}
		p.Limit = n
	}
	return p, nil
}
