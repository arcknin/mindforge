package brain

// ErrorCode is intentionally small and stable. Error values never contain a
// principal, Brain ID, layer ID or display name.
type ErrorCode string

const (
	ErrorUnauthenticated ErrorCode = "unauthenticated"
	ErrorForbidden       ErrorCode = "forbidden"
	ErrorNotFound        ErrorCode = "not_found"
	ErrorInvalidName     ErrorCode = "invalid_name"
	ErrorConflict        ErrorCode = "conflict"
	ErrorInternal        ErrorCode = "internal"
)

// Error is an opaque typed domain error. Callers can use errors.Is or Code;
// no storage/provider detail is exposed by this layer.
type Error struct {
	code ErrorCode
}

func (e *Error) Error() string {
	if e == nil {
		return "brain: unknown error"
	}
	return "brain: " + string(e.code)
}

func (e *Error) Code() ErrorCode {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && e != nil && other != nil && e.code == other.code
}

var (
	ErrUnauthenticated = &Error{code: ErrorUnauthenticated}
	ErrForbidden       = &Error{code: ErrorForbidden}
	ErrNotFound        = &Error{code: ErrorNotFound}
	ErrInvalidName     = &Error{code: ErrorInvalidName}
	ErrConflict        = &Error{code: ErrorConflict}
	ErrInternal        = &Error{code: ErrorInternal}
)
