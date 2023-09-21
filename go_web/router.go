package go_web

import (
	"fmt"
	"net/http"
	"strings"
)

type router struct {
	roots    map[string]*node       // roots["GET"], roots["POST"]
	handlers map[string]HandlerFunc // handlers["GET-/a/:b/*c"]
}

func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	r.handlers[key] = handler

	parts := parseParts(pattern)
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)
}

// Paraments:
//
//	method (string)
//	path (string)
//
// Returns:
//
//	node (node): matched node
//	params (map[string]string): path parameters
func (r *router) getRoute(
	method string,
	path string,
) (*node, map[string]string) {
	if _, ok := r.roots[method]; !ok {
		return nil, nil
	}
	root := r.roots[method]
	pathParts := parseParts(path)
	params := make(map[string]string, 0)

	searchedPart := root.search(pathParts, 0)

	if searchedPart != nil {
		parts := parseParts(searchedPart.pattern)
		for index, part := range parts {
			if part[0] == ':' && len(part) > 1 {
				params[part[1:]] = pathParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(pathParts[index:], "/")
				break // only allow one '*'
			}
		}
		return searchedPart, params
	}

	return nil, nil
}

func (r *router) handle(c *Context) {
	node, params := r.getRoute(c.Method, c.Path)

	if node != nil {
		key := c.Method + "-" + node.pattern
		if handler, ok := r.handlers[key]; ok {
			c.Params = params
			c.handlers = append(c.handlers, handler)
		} else {
			panic(fmt.Sprintf("there is `%s` in trie three but don't have handleFunc", key))
		}
	} else {
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 not found: %s", c.Path)
		})
	}
	c.Next()
}

func parseParts(pattern string) []string {
	slice := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, part := range slice {
		if part != "" {
			parts = append(parts, part)
			if part[0] == '*' {
				break
			}
		}
	}

	return parts
}
