package apierr

import "fmt"

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]any
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func BadRequest(message string, fields map[string]any) *Error {
	return &Error{Status: 400, Code: "BAD_REQUEST", Message: message, Fields: fields}
}

func NotFound(code, message string) *Error {
	return &Error{Status: 404, Code: code, Message: message}
}

func Conflict(code, message string, fields map[string]any) *Error {
	return &Error{Status: 409, Code: code, Message: message, Fields: fields}
}

func Internal() *Error {
	return &Error{Status: 500, Code: "INTERNAL_ERROR", Message: "internal server error"}
}
