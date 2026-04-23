package es

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested document does not exist.
var ErrNotFound = errors.New("es: document not found")

// ResponseError represents a non-2xx response from Elasticsearch.
type ResponseError struct {
	StatusCode int
	RawBody    string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("es: response error status=%d body=%s", e.StatusCode, e.RawBody)
}
