package middlewares

import (
	"lightgo/lightgo"
	"log"
	"net/http"
	"runtime/debug"
)

func Recovery(logger *log.Logger) lightgo.Middleware {
	if logger == nil {
		logger = log.Default()
	}
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get(RequestIDKey)
				logger.Printf("panic=%v request_id=%v\n%s", recovered, requestID, debug.Stack())
				if !c.Written() {
					_ = c.Error(lightgo.NewHTTPError(http.StatusInternalServerError, "服务器内部错误"))
				}
			}
		}()
		next()
	}
}
