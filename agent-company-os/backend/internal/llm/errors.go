package llm

import "fmt"

const (
	ErrorUnavailable     = "llm_unavailable"
	ErrorAuthFailed      = "llm_auth_failed"
	ErrorRateLimited     = "llm_rate_limited"
	ErrorInvalidResponse = "llm_invalid_response"
)

type Error struct {
	Class  string
	Status int
	Err    error
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("%s: upstream status %d", e.Class, e.Status)
	}
	if e.Err != nil {
		return e.Class + ": " + e.Err.Error()
	}
	return e.Class
}

func (e *Error) Unwrap() error { return e.Err }

func ClassifyError(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*Error); ok {
		return e.Class
	}
	return ErrorUnavailable
}
