package meeng

import (
	"fmt"
	"time"
)

// Error is a request the API refused. Code is the API's, such as NOT_UNIQUE,
// FORBIDDEN or NOT_FOUND; it is INTERNAL_SERVER_ERROR when the answer carried
// none. RetryAfter is how long to wait on a 429, when the API said.
type Error struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter time.Duration
}

// Error says the status, the code and why.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("meeng: %d %s: %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("meeng: %d %s", e.StatusCode, e.Code)
}
