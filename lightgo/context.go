package lightgo

import (
	"lightgo/lightgo/binding"
	"lightgo/lightgo/render"
	"net/http"
	"sync"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *responseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += n
	return n, err
}
func (w *responseWriter) Status() int { return w.statusCode }
func (w *responseWriter) Size() int   { return w.bytesWritten }
func (w *responseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

type responseState interface {
	Status() int
	Size() int
}

type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request
	engine  *Engine
	params  map[string]string
	values  map[string]any
	mu      sync.RWMutex
	aborted bool
}

func newContext(engine *Engine, w http.ResponseWriter, r *http.Request) *Context {
	return &Context{Writer: &responseWriter{ResponseWriter: w}, Request: r, engine: engine, params: make(map[string]string), values: make(map[string]any)}
}
func (c *Context) Param(name string) string { return c.params[name] }
func (c *Context) Query(name string) string { return c.Request.URL.Query().Get(name) }
func (c *Context) QueryDefault(name, fallback string) string {
	if v := c.Query(name); v != "" {
		return v
	}
	return fallback
}
func (c *Context) FormValue(name string) string { return c.Request.FormValue(name) }
func (c *Context) Header(name string) string    { return c.Request.Header.Get(name) }
func (c *Context) SetHeader(name, value string) { c.Writer.Header().Set(name, value) }
func (c *Context) Status() int {
	if w, ok := c.Writer.(responseState); ok {
		return w.Status()
	}
	return 0
}
func (c *Context) Size() int {
	if w, ok := c.Writer.(responseState); ok {
		return w.Size()
	}
	return 0
}
func (c *Context) Written() bool   { return c.Status() != 0 }
func (c *Context) Abort()          { c.aborted = true }
func (c *Context) IsAborted() bool { return c.aborted }
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.values[key]
	return value, ok
}
func (c *Context) MustGet(key string) any {
	value, ok := c.Get(key)
	if !ok {
		panic("context value not found: " + key)
	}
	return value
}
func (c *Context) JSON(status int, value any) error    { return render.JSON(c.Writer, status, value) }
func (c *Context) XML(status int, value any) error     { return render.XML(c.Writer, status, value) }
func (c *Context) Text(status int, value string) error { return render.Text(c.Writer, status, value) }
func (c *Context) Blob(status int, contentType string, data []byte) error {
	return render.Blob(c.Writer, status, contentType, data)
}
func (c *Context) Redirect(status int, location string) error {
	return render.Redirect(c.Writer, c.Request, status, location)
}
func (c *Context) StatusOnly(status int) error { return render.Status(c.Writer, status) }
func (c *Context) File(path string, download bool) error {
	return render.File(c.Writer, c.Request, path, download)
}
func (c *Context) HTML(status int, name string, data any) error {
	if c.engine.templates == nil {
		return c.Error(NewHTTPError(http.StatusInternalServerError, "template engine is not configured"))
	}
	if err := c.engine.templates.Render(c.Writer, status, name, data); err != nil {
		return c.Error(NewHTTPError(http.StatusInternalServerError, "template render failed", err))
	}
	return nil
}
func (c *Context) Bind(dst any) error       { return binding.Bind(c.Request, dst, binding.SourceAuto, c) }
func (c *Context) BindJSON(dst any) error   { return binding.BindJSON(c.Request, dst) }
func (c *Context) BindForm(dst any) error   { return binding.BindForm(c.Request, dst) }
func (c *Context) BindQuery(dst any) error  { return binding.BindQuery(c.Request, dst) }
func (c *Context) BindParam(dst any) error  { return binding.BindParam(c.Request, dst, c) }
func (c *Context) BindHeader(dst any) error { return binding.BindHeader(c.Request, dst) }
func (c *Context) Error(err error) error {
	if err == nil {
		return nil
	}
	c.engine.errorHandler(c, err)
	return err
}
