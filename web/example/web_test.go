package example

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"testing"
	web "web/v9"
	"web/v9/middleware/accesslog"
	"web/v9/middleware/errhdl"
	"web/v9/middleware/opentelemetry"
	"web/v9/middleware/recovery"
)

const defualtTpl = "F:\\go语言学习资料\\极客时间\\go实战训练营\\web框架\\exercise_code\\src\\web\\testdata\\tpls\\login.gohtml"

//import "testing"
//
//func TestStart(t *testing.T) {
//	Start()
//}

func login(ctx *web.Context) {
	tpl := template.New("login")
	tpl, err := tpl.Parse(`
<html>
	<body>
		<form>
			// 在这里继续写页面
		<form>
	</body>
</html>
`)
	if err != nil {
		fmt.Println(err)
	}
	page := &bytes.Buffer{}
	err = tpl.Execute(page, nil)
	if err != nil {
		fmt.Println(err)
	}
	ctx.RespStatusCode = http.StatusOK
	ctx.RespData = page.Bytes()
}

func TestServer(t *testing.T) {

	s := web.NewHTTPServer(web.ServerWithTemplateEngine(&web.GoTemplateEngine{}))

	err := s.TplEngine.LoadFromGlob(defualtTpl)
	if err != nil {
		t.Fatal(err)
	}

	bufByte, err := s.TplEngine.ExcuteTpl()
	if err != nil {
		t.Fatal(err)
	}

	logBd := accesslog.NewBuilder()
	erHdl := errhdl.NewBuilder().RegisterError(404, bufByte)
	tracBd := &opentelemetry.MiddlewareBuilder{}
	pacnicBd := recovery.NewBuilder()

	s.Use(pacnicBd.Build())

	//s.Get("/", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, world"))
	//})

	s.Get("/a/*/c", func(ctx *web.Context) {
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("hello, a * c")
	}, logBd.Build())

	s.Get("/a/b/c", func(ctx *web.Context) {
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("hello, a b c")
	}, erHdl.Build())

	s.Post("/a/b/c", func(ctx *web.Context) {
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("hello, a b c")
	}, tracBd.Build())

	//s.Get("/user", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, user"))
	//})
	//
	//s.Get("/user/:id", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, user param"))
	//})
	//
	//s.Get("/a/b/*", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, a,b start"))
	//})
	//
	//s.Get("/order/*", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, order start"))
	//})

	// 正则匹配
	s.Get("/sku/:id(^[0-9]+$)", func(ctx *web.Context) {
		ctx.RespStatusCode = http.StatusOK
		ctx.RespData = []byte("hello,regx route")
	})

	// 模板渲染
	s.Get("/login", login)

	g := s.NewRouteGroup("/order")

	g.AddRoute(http.MethodGet, "/detail", func(ctx *web.Context) {
		ctx.Response.Write([]byte("hello, order detail"))
	})

	s.Start("127.0.0.1:8090")

}
