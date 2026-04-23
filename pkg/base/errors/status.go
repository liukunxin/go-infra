package errors

import "net/http"

// ---------------------------
// Status 封装
// ---------------------------

const (
	StatusOK                  = Status(http.StatusOK)
	StatusBadRequest          = Status(http.StatusBadRequest)
	StatusUnauthorized        = Status(http.StatusUnauthorized)
	StatusForbidden           = Status(http.StatusForbidden)
	StatusNotFound            = Status(http.StatusNotFound)
	StatusTooManyRequests     = Status(http.StatusTooManyRequests)
	StatusInternalServerError = Status(http.StatusInternalServerError)
)

type Status int

func (s Status) Error() string {
	return http.StatusText(int(s))
}
func (s Status) HTTPCode() int {
	return int(s)
}
