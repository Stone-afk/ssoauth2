package web

import (
	"log"
	"net"
	"net/http"
	"web/context"
	webHandler "web/handler"
	"web/middleware"
	"web/template"
)

func (s *HTTPServer) Delete(path string, handleFunc webHandler.HandleFunc, mdls ...middleware.Middleware) {
	s.addRoute(http.MethodDelete, path, handleFunc, mdls...)
}

func (s *HTTPServer) Post(path string, handleFunc webHandler.HandleFunc, mdls ...middleware.Middleware) {
	s.addRoute(http.MethodPost, path, handleFunc, mdls...)
}

func (s *HTTPServer) Get(path string, handleFunc webHandler.HandleFunc, mdls ...middleware.Middleware) {
	s.addRoute(http.MethodGet, path, handleFunc, mdls...)
}

// UseMdls 会执行路由匹配，只有匹配上了的 mdls 才会生效
// 这个只需要稍微改造一下路由树就可以实现
func (s *HTTPServer) UseMdls(method string, path string, mdls ...middleware.Middleware) {
	s.addRoute(method, path, nil, mdls...)
}

func (s *HTTPServer) Use(mdls ...middleware.Middleware) {
	if s.mdls == nil {
		s.mdls = mdls
		return
	}
	s.mdls = append(s.mdls, mdls...)
}

func (s *HTTPServer) Response(ctx *context.Context) {
	if ctx.RespStatusCode > 0 {
		ctx.Response.WriteHeader(ctx.RespStatusCode)
	}
	_, err := ctx.Response.Write(ctx.RespData)
	if err != nil {
		log.Fatalln("回写响应失败", err)
	}
}

func (s *HTTPServer) serve(ctx *context.Context) {
	mi, ok := s.findRoute(ctx.Request.Method, ctx.Request.URL.Path)
	if !ok || mi.n.handler == nil {
		// 没找到路由树 or 路由树未定义方法
		ctx.RespStatusCode = http.StatusNotFound
		return
	}
	ctx.PathParams = mi.pathParams
	// 命中的路由需要缓存起来
	ctx.MatchedRoute = mi.n.route

	var handler = mi.n.handler
	if mi.mdls != nil && len(mi.mdls) > 0 {
		for i := len(mi.mdls) - 1; i >= 0; i-- {
			handler = mi.mdls[i](handler)
		}
	}

	handler(ctx)
}

func (s *HTTPServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	ctx := &context.Context{
		Request:   req,
		Response:  writer,
		TplEngine: s.TplEngine,
	}

	// 最后一个应该是 HTTPServer 执行路由匹配，执行用户代码
	handler := s.serve
	// 从后往前组装
	for i := len(s.mdls) - 1; i >= 0; i-- {
		handler = s.mdls[i](handler)
	}

	// 第一个应该是回写响应的
	// 因为它在调用next之后才回写响应，
	// 所以实际上 Response 是最后一个步骤
	var m middleware.Middleware = func(next webHandler.HandleFunc) webHandler.HandleFunc {
		return func(ctx *context.Context) {
			next(ctx)
			s.Response(ctx)
		}
	}
	handler = m(handler)
	handler(ctx)

	// s.serve(ctx)
}

func (s *HTTPServer) Start(addr string) error {

	linstener, err := net.Listen("tcp", addr)

	if err != nil {
		return err
	}

	for _, root := range s.trees {
		s.findAndLoadMdls(root)
	}

	println("成功监听地址", addr)

	return http.Serve(linstener, s)

	// return http.ListenAndServe(addr, s)

}

// ServerWithTemplateEngine 因为渲染页面是一种个性需求，所以我们做成 Option 模式， 需要的用户自己注入 TemplateEngine。
func ServerWithTemplateEngine(engine template.TemplateEngine) ServerOption {
	return func(server *HTTPServer) {
		server.TplEngine = engine
	}
}

func NewHTTPServer(opts ...ServerOption) *HTTPServer {
	s := &HTTPServer{
		router: newRouter(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type HTTPServer struct {
	router
	mdls      []middleware.Middleware
	TplEngine template.TemplateEngine
}

var _ Server = &HTTPServer{}

type ServerOption func(server *HTTPServer)

type Server interface {
	http.Handler

	Start(addr string) error
	addRoute(method, path string, handleFunc webHandler.HandleFunc, mdls ...middleware.Middleware)
}
