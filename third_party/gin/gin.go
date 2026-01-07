package gin

import (
	"encoding/json"
	"net/http"
	"strings"
)

// H is a shortcut for JSON responses.
type H map[string]any

// HandlerFunc defines the handler used by middleware.
type HandlerFunc func(*Context)

// Context holds request context and response writer.
type Context struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	params   map[string]string
	handlers []HandlerFunc
	index    int
	aborted  bool
}

// Next executes the next handler in the chain.
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		c.handlers[c.index](c)
		if c.aborted {
			return
		}
		c.index++
	}
}

// Param retrieves a path parameter by name.
func (c *Context) Param(key string) string {
	return c.params[key]
}

// Query retrieves a query string value.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// GetHeader retrieves a header value.
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Header sets a response header.
func (c *Context) Header(key, value string) {
	c.Writer.Header().Set(key, value)
}

// JSON writes a JSON response.
func (c *Context) JSON(code int, obj any) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

// AbortWithStatusJSON stops handler chain and writes JSON response.
func (c *Context) AbortWithStatusJSON(code int, obj any) {
	c.aborted = true
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)
	_ = json.NewEncoder(c.Writer).Encode(obj)
}

// ShouldBindJSON binds request body JSON into the provided struct pointer.
func (c *Context) ShouldBindJSON(obj any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(obj)
}

// Engine is the main HTTP router.
type Engine struct {
	routes     map[string][]routeEntry
	middleware []HandlerFunc
}

// routeEntry represents a registered route.
type routeEntry struct {
	pattern  string
	handlers []HandlerFunc
}

// New creates a new Engine.
func New() *Engine {
	return &Engine{routes: make(map[string][]routeEntry)}
}

// Use registers global middleware.
func (e *Engine) Use(handlers ...HandlerFunc) {
	e.middleware = append(e.middleware, handlers...)
}

// Group creates a router group with shared prefix.
func (e *Engine) Group(prefix string) *RouterGroup {
	return &RouterGroup{engine: e, prefix: prefix}
}

// ServeHTTP implements http.Handler.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := &Context{Writer: w, Request: r, params: map[string]string{}, index: -1}
	handlers, params := e.match(r.Method, r.URL.Path)
	c.params = params
	c.handlers = append(c.handlers, e.middleware...)
	c.handlers = append(c.handlers, handlers...)
	if len(c.handlers) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	c.Next()
}

// GET registers a GET route.
func (e *Engine) GET(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodGet, path, handlers)
}

// POST registers a POST route.
func (e *Engine) POST(path string, handlers ...HandlerFunc) {
	e.addRoute(http.MethodPost, path, handlers)
}

func (e *Engine) addRoute(method, path string, handlers []HandlerFunc) {
	key := method
	e.routes[key] = append(e.routes[key], routeEntry{pattern: path, handlers: handlers})
}

func (e *Engine) match(method, path string) ([]HandlerFunc, map[string]string) {
	entries := e.routes[method]
	for _, entry := range entries {
		if params, ok := matchPattern(entry.pattern, path); ok {
			return entry.handlers, params
		}
	}
	return nil, map[string]string{}
}

// RouterGroup groups routes under a common prefix.
type RouterGroup struct {
	engine *Engine
	prefix string
}

// Use allows group-specific middleware.
func (g *RouterGroup) Use(handlers ...HandlerFunc) {
	g.engine.middleware = append(g.engine.middleware, handlers...)
}

// GET registers a GET route within the group.
func (g *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	g.engine.GET(g.combine(path), handlers...)
}

// POST registers a POST route within the group.
func (g *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	g.engine.POST(g.combine(path), handlers...)
}

func (g *RouterGroup) combine(path string) string {
	if g.prefix == "" {
		return path
	}
	return strings.TrimRight(g.prefix, "/") + path
}

// Recovery returns a middleware that recovers from panics.
func Recovery() HandlerFunc {
	return func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				http.Error(c.Writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				c.aborted = true
			}
		}()
		c.Next()
	}
}

func matchPattern(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return map[string]string{}, true
	}
	pSegs := strings.Split(strings.Trim(pattern, "/"), "/")
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")
	if len(pSegs) != len(pathSegs) {
		return nil, false
	}
	params := make(map[string]string)
	for i := range pSegs {
		if strings.HasPrefix(pSegs[i], ":") {
			params[pSegs[i][1:]] = pathSegs[i]
			continue
		}
		if pSegs[i] != pathSegs[i] {
			return nil, false
		}
	}
	return params, true
}
