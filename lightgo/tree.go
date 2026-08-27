package lightgo

import (
	"fmt"
	"strings"
)

type nodeKind uint8

const (
	staticNode nodeKind = iota
	paramNode
	wildcardNode
)

type routeNode struct {
	segment  string
	kind     nodeKind
	children map[string]*routeNode
	param    *routeNode
	wildcard *routeNode
	handler  HandlerFunc
	route    *Route
}

func newRouteNode(segment string, kind nodeKind) *routeNode {
	return &routeNode{segment: segment, kind: kind, children: make(map[string]*routeNode)}
}
func splitPath(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return nil
	}
	return strings.Split(strings.Trim(path, "/"), "/")
}
func (n *routeNode) insert(path string, route *Route) error {
	parts := splitPath(path)
	current := n
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid empty path segment in %q", path)
		}
		switch part[0] {
		case ':':
			name := strings.TrimPrefix(part, ":")
			if name == "" {
				return fmt.Errorf("empty parameter in %q", path)
			}
			if current.param == nil {
				current.param = newRouteNode(name, paramNode)
			}
			current = current.param
		case '*':
			name := strings.TrimPrefix(part, "*")
			if name == "" {
				return fmt.Errorf("empty wildcard in %q", path)
			}
			if i != len(parts)-1 {
				return fmt.Errorf("wildcard must be final in %q", path)
			}
			if current.wildcard == nil {
				current.wildcard = newRouteNode(name, wildcardNode)
			}
			current = current.wildcard
		default:
			child := current.children[part]
			if child == nil {
				child = newRouteNode(part, staticNode)
				current.children[part] = child
			}
			current = child
		}
	}
	if current.handler != nil {
		return fmt.Errorf("route already registered: %s", path)
	}
	current.handler, current.route = route.handler, route
	return nil
}
func (n *routeNode) search(path string) (HandlerFunc, *Route, map[string]string, bool) {
	parts := splitPath(path)
	params := make(map[string]string)
	h, route, ok := searchParts(n, parts, 0, params)
	return h, route, params, ok
}
func searchParts(n *routeNode, parts []string, index int, params map[string]string) (HandlerFunc, *Route, bool) {
	if index == len(parts) {
		if n.handler != nil {
			return n.handler, n.route, true
		}
		if n.wildcard != nil && n.wildcard.handler != nil {
			params[n.wildcard.segment] = ""
			return n.wildcard.handler, n.wildcard.route, true
		}
		return nil, nil, false
	}
	part := parts[index]
	if child := n.children[part]; child != nil {
		if h, route, ok := searchParts(child, parts, index+1, params); ok {
			return h, route, true
		}
	}
	if n.param != nil {
		params[n.param.segment] = part
		if h, route, ok := searchParts(n.param, parts, index+1, params); ok {
			return h, route, true
		}
		delete(params, n.param.segment)
	}
	if n.wildcard != nil && n.wildcard.handler != nil {
		params[n.wildcard.segment] = strings.Join(parts[index:], "/")
		return n.wildcard.handler, n.wildcard.route, true
	}
	return nil, nil, false
}
