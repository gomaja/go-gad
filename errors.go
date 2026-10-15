package gad

import (
	"errors"
	"fmt"
	"strings"
)

// Coding errors identify failures under TS 23.032 V19.0.0 §§6–8.
var (
	// ErrLength reports an incorrect octet count under TS 23.032 V19.0.0 §§7–8.
	ErrLength = errors.New("gad: invalid length")
	// ErrShapeTypeInvalid reports a reserved shape under TS 23.032 V19.0.0 §7.2.
	ErrShapeTypeInvalid = errors.New("gad: reserved type of shape")
	// ErrVelocityTypeInvalid reports a reserved velocity under TS 23.032 V19.0.0 §8.6.
	ErrVelocityTypeInvalid = errors.New("gad: reserved velocity type")
	// ErrSpareBit reports a nonzero spare bit under TS 23.032 V19.0.0 §§7–8.
	ErrSpareBit = errors.New("gad: spare bit set")
	// ErrOutOfRange reports an invalid field value under TS 23.032 V19.0.0 §§6–8.
	ErrOutOfRange = errors.New("gad: value out of range")
	// ErrNotFinite reports nonfinite physical units under TS 23.032 V19.0.0 §§6–8.
	ErrNotFinite = errors.New("gad: value not finite")
	// ErrUncertaintyRange reports differing horizontal ranges under TS 23.032 V19.0.0 §§7.3.3b, 7.3.6b.
	ErrUncertaintyRange = errors.New("gad: semi-major and semi-minor uncertainty ranges differ")
)

// FieldError locates a coding failure in TS 23.032 V19.0.0 §§6–8.
type FieldError struct {
	// Field is the figure label or the shape/velocity name for length errors.
	Field string
	// Octet is the first field octet, or the first missing/unexpected octet for length errors; 0 denotes a constructor.
	Octet int
	// Err is the coding sentinel, exposed by Unwrap; it may be nil.
	Err error
	// Detail describes the invalid value and its expected domain, when available.
	Detail string
}

// Error describes the field and octet under TS 23.032 V19.0.0 §§7–8.
func (e *FieldError) Error() string {
	where := e.Field
	if e.Octet > 0 {
		where = fmt.Sprintf("%s (octet %d)", where, e.Octet)
	}
	msg := "gad: " + where
	if e.Err != nil {
		msg += ": " + strings.TrimPrefix(e.Err.Error(), "gad: ")
	}
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

// Unwrap exposes the coding error under TS 23.032 V19.0.0 §§6–8.
func (e *FieldError) Unwrap() error { return e.Err }

func fieldError(field string, octet int, err error, detail string) error {
	return &FieldError{Field: field, Octet: octet, Err: err, Detail: detail}
}

func rangeError(field string, octet int, n, low, high int64) error {
	return fieldError(field, octet, ErrOutOfRange, fmt.Sprintf("%d, want %d..%d", n, low, high))
}

func lengthError(field string, got, want int) error {
	octet := min(got, want) + 1
	return fieldError(field, octet, ErrLength, fmt.Sprintf("%d octets, want %d", got, want))
}
