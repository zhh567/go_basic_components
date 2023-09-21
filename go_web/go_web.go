package go_web

import (
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"text/template"
)

type Engine struct {
	router *router

	// group
	*RouterGroup                // Engine is the top level group. It has all RouterGroup's ability.
	groups       []*RouterGroup // store all groups

	// html template
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

// Group && Middleware
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup // suppert nesting
	engine      *Engine      // all groups share a Engine instance
}

func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = append(engine.groups, engine.RouterGroup)
	return engine
}

// struct `Engine` has field `*RouterGroup`
// and it inherits all methods

func (g *RouterGroup) Group(prefix string) *RouterGroup {
	engine := g.engine
	newGroup := &RouterGroup{
		prefix: g.prefix + prefix,
		parent: g,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (g *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	pattern := g.prefix + comp
	slog.Info(fmt.Sprintf("add route %4s - %s", method, pattern))
	g.engine.router.addRoute(method, pattern, handler)
}
func (g *RouterGroup) GET(pattern string, handler HandlerFunc) {
	g.addRoute("GET", pattern, handler)
}
func (g *RouterGroup) POST(pattern string, handler HandlerFunc) {
	g.addRoute("POST", pattern, handler)
}

// func (e *Engine) addRoute(method string, pattern string, handler HandlerFunc) {
// 	e.router.addRoute(method, pattern, handler)
// }
// func (e *Engine) GET(pattern string, handler HandlerFunc) {
// 	e.addRoute("GET", pattern, handler)
// }
// func (e *Engine) POST(pattern string, handler HandlerFunc) {
// 	e.addRoute("POST", pattern, handler)
// }

// file server
func (g *RouterGroup) Static(relativePath string, root string) {
	handler := g.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	g.GET(urlPattern, handler)
}
func (g *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(g.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Status(http.StatusOK)
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// add midle ware to the group
func (g *RouterGroup) Use(middlewares ...HandlerFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

// set HTML template analyze function
func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

// set HTML file direction
func (e *Engine) LoadHTMLGlob(pattern string) {
	e.htmlTemplates = template.Must(
		template.New("").Funcs(e.funcMap).ParseGlob(pattern))
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	// When a new request come, check which ones match
	for _, group := range e.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req, e)
	c.handlers = append(c.handlers, middlewares...)
	e.router.handle(c)
}

func (e *Engine) Run(add string) (err error) {
	return http.ListenAndServe(add, e)
}
