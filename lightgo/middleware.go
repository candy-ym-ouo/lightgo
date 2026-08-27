package lightgo

type HandlerFunc func(*Context)
type NextFunc func()
type Middleware func(*Context, NextFunc)

func runChain(c *Context, handler HandlerFunc, middleware []Middleware) {
	index := -1
	var next NextFunc
	next = func() {
		index++
		if index < len(middleware) {
			middleware[index](c, next)
			return
		}
		if index == len(middleware) && handler != nil && !c.aborted {
			handler(c)
		}
	}
	next()
}
func combineMiddleware(parts ...[]Middleware) []Middleware {
	total := 0
	for _, part := range parts {
		total += len(part)
	}
	out := make([]Middleware, 0, total)
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}
