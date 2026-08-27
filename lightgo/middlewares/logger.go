package middlewares

import (
	"lightgo/lightgo"
	"log"
	"time"
)

func Logger(logger *log.Logger) lightgo.Middleware {
	if logger == nil {
		logger = log.Default()
	}
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		started := time.Now()
		next()
		status := c.Status()
		if status == 0 {
			status = 200
		}
		requestID, _ := c.Get(RequestIDKey)
		logger.Printf("method=%s path=%s status=%d bytes=%d duration=%s request_id=%v remote=%s",
			c.Request.Method, c.Request.URL.RequestURI(), status, c.Size(), time.Since(started).Round(time.Microsecond), requestID, c.Request.RemoteAddr)
	}
}
