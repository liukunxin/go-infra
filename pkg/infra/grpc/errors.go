package grpc

import (
	"net/http"

	kerr "github.com/liukunxin/go-infra/pkg/base/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HTTPStatusFromGRPCCode converts gRPC code to HTTP status.
func HTTPStatusFromGRPCCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// GRPCCodeFromHTTPStatus converts HTTP status to gRPC code.
func GRPCCodeFromHTTPStatus(statusCode int) codes.Code {
	switch statusCode {
	case http.StatusOK:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.Aborted
	case http.StatusPreconditionFailed:
		return codes.FailedPrecondition
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		if statusCode >= 500 {
			return codes.Internal
		}
		return codes.Unknown
	}
}

func GRPCErrorFromHTTPStatus(statusCode int, message string) error {
	return status.Error(GRPCCodeFromHTTPStatus(statusCode), message)
}

func GRPCErrorFromStatus(s kerr.Status, message string) error {
	return status.Error(GRPCCodeFromHTTPStatus(int(s)), message)
}

func httpStatusToCode(statusCode int) codes.Code {
	return GRPCCodeFromHTTPStatus(statusCode)
}
