package sso

import (
	"html/template"
	"net/http"
	"ssoauth2/web"
	"ssoauth2/web/context"
	webTlp "ssoauth2/web/template"
	"testing"
)

func TestSSOServer(t *testing.T) {
	tpls := template.New("test_server")
	tpls, err := tpls.ParseGlob("./template/*")
	if err != nil {
		t.Fatal(err)
	}
	engine := &webTlp.GoTemplateEngine{
		T: tpls,
	}
	server := web.NewHTTPServer(web.ServerWithTemplateEngine(engine))

	server.Post("/hello", func(ctx *context.Context) {
		_ = ctx.RespString(http.StatusOK, "欢迎来到 SSO")
	})

	server.Post("/logout", func(ctx *context.Context) {

	})

	server.Post("/login", func(ctx *context.Context) {

	})

	server.Post("/check_login", func(ctx *context.Context) {

	})

}
