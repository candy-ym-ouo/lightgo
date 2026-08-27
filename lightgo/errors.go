package lightgo

import (
	"errors"
	"fmt"
	"net/http"
)

type HTTPError struct {
	Code    int
	Message string
	Err     error
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%d %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%d %s", e.Code, e.Message)
}
func (e *HTTPError) Unwrap() error { return e.Err }
func NewHTTPError(code int, message string, err ...error) *HTTPError {
	e := &HTTPError{Code: code, Message: message}
	if len(err) > 0 {
		e.Err = err[0]
	}
	return e
}

type ErrorHandler func(*Context, error)

func defaultErrorHandler(c *Context, err error) {
	if c.Written() {
		return
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		httpErr = NewHTTPError(http.StatusInternalServerError, "服务器内部错误", err)
	}
	_ = c.JSON(httpErr.Code, map[string]any{"code": httpErr.Code, "message": httpErr.Message, "data": nil})
}
