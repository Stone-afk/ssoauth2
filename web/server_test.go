package web

import (
	"bytes"
	"fmt"
	"html/template"
	"testing"
	"web/context"
	"web/middleware/accesslog"
	"web/middleware/errhdl"
	"web/middleware/recovery"
	template2 "web/template"
)

func login(ctx *context.Context) {
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
	ctx.RespStatusCode = 200
	ctx.RespData = page.Bytes()
}

func TestServer(t *testing.T) {

	s := NewHTTPServer(ServerWithTemplateEngine(&template2.GoTemplateEngine{}))

	err := s.TplEngine.LoadFromGlob("testdata/tpls/*.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	bufByte, err := s.TplEngine.ExcuteTpl()
	if err != nil {
		t.Fatal(err)
	}

	logBd := accesslog.NewBuilder()
	erHdl := errhdl.NewBuilder().RegisterError(404, bufByte)
	//tracBd := &opentelemetry.MiddlewareBuilder{}
	pacnicBd := &recovery.MiddlewareBuilder{}

	s.Use(pacnicBd.Build())

	//s.Get("/", func(ctx *Context) {
	//	ctx.Response.Write([]byte("hello, world"))
	//})

	s.Get("/a/*/c", func(ctx *context.Context) {
		ctx.Response.Write([]byte("hello, a * c"))
	})
	s.UseMdls("GET", "/a/*/c", logBd.Build())

	s.Get("/a/b/c", func(ctx *context.Context) {
		ctx.Response.Write([]byte("hello, a b c"))
	})
	s.UseMdls("GET", "/a/b/c", erHdl.Build())

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
	s.Get("/sku/:id(^[0-9]+$)", func(ctx *context.Context) {
		ctx.Response.Write([]byte("hello,regx route"))
	})

	// 模板渲染
	s.Get("login", login)

	s.Start("127.0.0.1:8090")

}

func TestServerWithRenderEngine(t *testing.T) {

	s := NewHTTPServer(ServerWithTemplateEngine(&template2.GoTemplateEngine{}))

	err := s.TplEngine.LoadFromGlob("testdata/tpls/*.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	s.Get("/login", func(ctx *context.Context) {
		er := ctx.Render("login.gohtml", nil)
		if er != nil {
			t.Fatal(er)
		}
	})

	err = s.Start(":8081")

	if err != nil {
		t.Fatal(err)
	}
}
