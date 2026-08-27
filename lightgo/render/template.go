package render

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TemplateEngine struct {
	mu        sync.RWMutex
	templates map[string]*template.Template
	funcs     template.FuncMap
}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{templates: make(map[string]*template.Template), funcs: template.FuncMap{
		"date": func(v time.Time) string {
			if v.IsZero() {
				return "-"
			}
			return v.Format("2006-01-02 15:04")
		},
		"truncate": func(s string, n int) string {
			r := []rune(s)
			if len(r) <= n {
				return s
			}
			return string(r[:n]) + "…"
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"default": func(fallback string, value any) any {
			if value == nil || fmt.Sprint(value) == "" {
				return fallback
			}
			return value
		},
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
	}}
}
func (e *TemplateEngine) Funcs(funcs template.FuncMap) *TemplateEngine {
	e.mu.Lock()
	defer e.mu.Unlock()
	for name, fn := range funcs {
		e.funcs[name] = fn
	}
	return e
}
func (e *TemplateEngine) Load(dir string) error {
	layout := filepath.Join(dir, "layout.html")
	if _, err := os.Stat(layout); err != nil {
		return fmt.Errorf("template layout: %w", err)
	}
	found := make(map[string]*template.Template)
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".html" || filepath.Base(path) == "layout.html" {
			return nil
		}
		name := filepath.Base(path)
		t, err := template.New("layout.html").Funcs(e.funcs).ParseFiles(layout, path)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		found[name] = t
		return nil
	})
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return errors.New("no page templates found")
	}
	e.mu.Lock()
	e.templates = found
	e.mu.Unlock()
	return nil
}
func (e *TemplateEngine) Render(w http.ResponseWriter, status int, name string, data any) error {
	if filepath.Ext(name) == "" {
		name += ".html"
	}
	e.mu.RLock()
	t, ok := e.templates[name]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("template %q not loaded", name)
	}
	var body strings.Builder
	if err := t.ExecuteTemplate(&body, "layout.html", data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(body.String()))
	return err
}
func (e *TemplateEngine) Names() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]string, 0, len(e.templates))
	for name := range e.templates {
		out = append(out, name)
	}
	return out
}
