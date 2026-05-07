package xpaymentsdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HTTPError struct {
	Code       int
	Message    string
	StatusCode int
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode > 0 && e.StatusCode != 200 {
		return fmt.Sprintf("xpayment http status %d: code=%d message=%s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("xpayment http error: code=%d message=%s", e.Code, e.Message)
}

func shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return true
		default:
			return false
		}
	}
	message := strings.ToLower(err.Error())
	for _, token := range []string{
		"connection refused",
		"connection reset",
		"connection closed",
		"transport is closing",
		"broken pipe",
		"eof",
		"no such host",
	} {
		if strings.Contains(message, token) {
			return true
		}
	}
	return false
}
