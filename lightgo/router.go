package lightgo

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

var standardMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions}

type Route struct {
	Method     string
	Path       string
	handler    HandlerFunc
	middleware []Middleware
}

func (r *Route) Use(middleware ...Middleware) *Route {
	r.middleware = append(r.middleware, middleware...)
	return r
}

type Router struct {
	mu     sync.RWMutex
	trees  map[string]*routeNode
	routes []*Route
}

func NewRouter() *Router { return &Router{trees: make(map[string]*routeNode)} }
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}
func (r *Router) Add(method, path string, handler HandlerFunc, middleware ...Middleware) (*Route, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	if handler == nil {
		return nil, fmt.Errorf("nil handler for %s %s", method, path)
	}
	if method == "ANY" {
		var first *Route
		for _, m := range standardMethods {
			route, err := r.Add(m, path, handler, middleware...)
			if err != nil {
				return nil, err
			}
			if first == nil {
				first = route
			}
		}
		return first, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	root := r.trees[method]
	if root == nil {
		root = newRouteNode("", staticNode)
		r.trees[method] = root
	}
	route := &Route{Method: method, Path: path, handler: handler, middleware: append([]Middleware(nil), middleware...)}
	if err := root.insert(path, route); err != nil {
		return nil, err
	}
	r.routes = append(r.routes, route)
	return route, nil
}
func (r *Router) Match(method, path string) (HandlerFunc, *Route, map[string]string, []string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	method = strings.ToUpper(method)
	path = normalizePath(path)
	if root := r.trees[method]; root != nil {
		if h, route, params, ok := root.search(path); ok {
			return h, route, params, nil
		}
	}
	allowed := make([]string, 0)
	for candidate, root := range r.trees {
		if candidate == method {
			continue
		}
		if _, _, _, ok := root.search(path); ok {
			allowed = append(allowed, candidate)
		}
	}
	sort.Strings(allowed)
	return nil, nil, nil, allowed
}
func (r *Router) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Route, len(r.routes))
	for i, route := range r.routes {
		out[i] = *route
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out
}
