package middlewares

import (
	"crypto/rand"
	"fmt"
	"lightgo/lightgo"
)

const RequestIDKey = "requestID"

func RequestID() lightgo.Middleware {
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		id := c.Header("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.SetHeader("X-Request-ID", id)
		next()
	}
}
func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "lightgo-request"
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
}
