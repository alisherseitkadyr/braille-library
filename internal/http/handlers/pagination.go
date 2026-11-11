package handlers

import (
	"errors"
	"net/http"
	"strconv"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// parsePagination reads page and limit. Limit above 100 is capped at 100.
func parsePagination(r *http.Request) (page, limit int, err error) {
	page, limit = 1, defaultLimit
	q := r.URL.Query()

	if s := q.Get("page"); s != "" {
		page, err = strconv.Atoi(s)
		if err != nil || page < 1 {
			return 0, 0, errors.New("page must be a positive integer")
		}
	}
	if s := q.Get("limit"); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil || limit < 1 {
			return 0, 0, errors.New("limit must be a positive integer")
		}
		limit = min(limit, maxLimit)
	}
	return page, limit, nil
}
