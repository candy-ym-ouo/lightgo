package lightgo

import (
	"fmt"
	"lightgo/lightgo/render"
	"log"
	"net/http"
	"strings"
)

type Engine struct {
	router       *Router
	middleware   []Middleware
	templates    *render.TemplateEngine
	errorHandler ErrorHandler
	NotFound     HandlerFunc
	MethodDenied HandlerFunc
	logger       *log.Logger
}

func New() *Engine {
	e := &Engine{router: NewRouter(), errorHandler: defaultErrorHandler, logger: log.Default()}
	e.NotFound = func(c *Context) {
		_ = c.JSON(http.StatusNotFound, map[string]any{"code": 404, "message": "资源不存在", "data": nil})
	}
	e.MethodDenied = func(c *Context) {
		_ = c.JSON(http.StatusMethodNotAllowed, map[string]any{"code": 405, "message": "请求方法不允许", "data": nil})
	}
	return e
}
func (e *Engine) Use(middleware ...Middleware)                  { e.middleware = append(e.middleware, middleware...) }
func (e *Engine) SetTemplates(templates *render.TemplateEngine) { e.templates = templates }
func (e *Engine) SetErrorHandler(handler ErrorHandler) {
	if handler != nil {
		e.errorHandler = handler
	}
}
func (e *Engine) Router() *Router { return e.router }
func (e *Engine) Handle(method, path string, handler HandlerFunc, middleware ...Middleware) *Route {
	route, err := e.router.Add(method, path, handler, middleware...)
	if err != nil {
		panic(err)
	}
	return route
}
func (e *Engine) GET(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodGet, path, handler, middleware...)
}
func (e *Engine) POST(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodPost, path, handler, middleware...)
}
func (e *Engine) PUT(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodPut, path, handler, middleware...)
}
func (e *Engine) PATCH(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodPatch, path, handler, middleware...)
}
func (e *Engine) DELETE(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodDelete, path, handler, middleware...)
}
func (e *Engine) OPTIONS(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle(http.MethodOptions, path, handler, middleware...)
}
func (e *Engine) ANY(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return e.Handle("ANY", path, handler, middleware...)
}
func (e *Engine) Group(prefix string, middleware ...Middleware) *Group {
	return &Group{engine: e, prefix: normalizePath(prefix), middleware: append([]Middleware(nil), middleware...)}
}
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newContext(e, w, r)
	handler, route, params, allowed := e.router.Match(r.Method, r.URL.Path)
	if handler == nil {
		if len(allowed) > 0 {
			c.Writer.Header().Set("Allow", strings.Join(allowed, ", "))
			runChain(c, e.MethodDenied, e.middleware)
		} else {
			runChain(c, e.NotFound, e.middleware)
		}
		return
	}
	c.params = params
	runChain(c, handler, combineMiddleware(e.middleware, route.middleware))
}
func (e *Engine) PrintRoutes() {
	for _, route := range e.router.Routes() {
		e.logger.Printf("%-7s %s", route.Method, route.Path)
	}
}

type Group struct {
	engine     *Engine
	prefix     string
	middleware []Middleware
}

func (g *Group) Use(middleware ...Middleware) { g.middleware = append(g.middleware, middleware...) }
func (g *Group) Group(prefix string, middleware ...Middleware) *Group {
	return &Group{engine: g.engine, prefix: joinPath(g.prefix, prefix), middleware: combineMiddleware(g.middleware, middleware)}
}
func (g *Group) Handle(method, path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.engine.Handle(method, joinPath(g.prefix, path), handler, combineMiddleware(g.middleware, middleware)...)
}
func (g *Group) GET(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle(http.MethodGet, path, handler, middleware...)
}
func (g *Group) POST(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle(http.MethodPost, path, handler, middleware...)
}
func (g *Group) PUT(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle(http.MethodPut, path, handler, middleware...)
}
func (g *Group) PATCH(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle(http.MethodPatch, path, handler, middleware...)
}
func (g *Group) DELETE(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle(http.MethodDelete, path, handler, middleware...)
}
func (g *Group) ANY(path string, handler HandlerFunc, middleware ...Middleware) *Route {
	return g.Handle("ANY", path, handler, middleware...)
}
func joinPath(prefix, path string) string {
	if path == "" || path == "/" {
		return normalizePath(prefix)
	}
	return normalizePath(strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(path, "/"))
}
func (e *Engine) String() string { return fmt.Sprintf("LightGo(%d routes)", len(e.router.Routes())) }
