package example

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

func Start() {
	var s Server = &HTTPServer{}
	var h1 HandleFunc = func(ctx *Context) {
		fmt.Println("步骤1")
		time.Sleep(time.Second)
	}

	var h2 HandleFunc = func(ctx *Context) {
		fmt.Println("步骤2")
		time.Sleep(time.Second)
	}

	s.AddRoute(http.MethodPost, "/user", func(ctx *Context) {
		// 循环调用多个 handlefunc
		h1(ctx)
		h2(ctx)
	})

	s.AddRoute(http.MethodPost, "/users", nil)
	// s.AddRoutes(http.MethodPost, "/user")
	// http.ListenAndServe(":8081", s)
	// http.ListenAndServeTLS("4000", "xxx", "aaa", s)
	s.Start("8081")

}

type Context struct {
	Request  *http.Request
	Response *http.Response
	Writer   http.ResponseWriter //  ResponseWriter 是一个接口
}

type Server interface {
	http.Handler
	Start(addr string) error

	// AddRoute 注册一个路由
	// method 是 HTTP 方法
	AddRoute(method, path string, handleFunc HandleFunc)

	// 我们并不采取这种设计方案
	// AddRoutes(method, path string, handlers ...HandleFunc)
}

type HandleFunc func(*Context)

//  @ HTTPServer  普通http服务
type HTTPServer struct {
}

// 确保 HTTPServer 肯定实现了 Server 接口
var _ Server = &HTTPServer{}

func (s *HTTPServer) AddRoute(method, path string, handleFunc HandleFunc) {
}

func (s *HTTPServer) Get(path string, handleFunc HandleFunc) {
	s.AddRoute(http.MethodGet, path, handleFunc)
}

func (s *HTTPServer) Post(path string, handleFunc HandleFunc) {
	s.AddRoute(http.MethodPost, path, handleFunc)
}

func (s *HTTPServer) Start(addr string) error {
	// 端口启动前
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	// 端口启动后
	// web 服务的服务发现
	// 注册本服务器到你的管理平台
	// 比如说你注册到 etcd，然后你打开管理界面，你就能看到这个实例
	// 10.0.0.1:8081
	println("成功监听端口 8081")
	// http.Serve 接收了一个 Listener
	return http.Serve(listener, s)

	// 这个是阻塞的
	// return http.ListenAndServe(addr, s)
	// 你没办法在这里做点什么
}

// ServeHTTP HTTPServer 处理请求的入口
func (s *HTTPServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := &Context{
		Request: request,
		Writer:  writer,
	}

	// 接下来就是
	// 查找路由
	// 执行业务逻辑
	s.serve(ctx)
}

func (s *HTTPServer) serve(ctx *Context) {

}

// HTTPSServer  https服务
type HTTPSServer struct {
	Server          // 接口 可以 组合 HTTPServer
	CertFile string // 证书
	KeyFile  string // 密钥
}

func (s *HTTPSServer) Start(addr string) error {
	return http.ListenAndServeTLS(addr, s.CertFile, s.KeyFile, s)
}
