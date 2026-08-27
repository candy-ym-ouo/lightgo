package middlewares

import (
	"lightgo/lightgo"
	"net/http"
	"strconv"
	"strings"
)

type CORSConfig struct {
	AllowOrigin                               string
	AllowMethods, AllowHeaders, ExposeHeaders []string
	AllowCredentials                          bool
	MaxAge                                    int
}

func CORS(config CORSConfig) lightgo.Middleware {
	if config.AllowOrigin == "" {
		config.AllowOrigin = "*"
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	}
	return func(c *lightgo.Context, next lightgo.NextFunc) {
		origin := c.Header("Origin")
		if origin != "" {
			c.SetHeader("Access-Control-Allow-Origin", config.AllowOrigin)
			c.SetHeader("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			c.SetHeader("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			if len(config.ExposeHeaders) > 0 {
				c.SetHeader("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
			}
			if config.AllowCredentials {
				c.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if config.MaxAge > 0 {
				c.SetHeader("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}
			c.Writer.Header().Add("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			_ = c.StatusOnly(http.StatusNoContent)
			c.Abort()
			return
		}
		next()
	}
}
