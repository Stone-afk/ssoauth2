package geekbang

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"ssoauth2/web"
	"ssoauth2/web/context"
	webTlp "ssoauth2/web/template"
)

func main() {
	tpls := template.New("geekbang")
	tpls, err := tpls.ParseGlob("./template/*")
	if err != nil {
		panic(err)
	}
	engine := webTlp.GoTemplateEngine{
		T: tpls,
	}
	server := web.NewHTTPServer(
		web.ServerWithTemplateEngine(engine))

	server.Get("/login", func(ctx *context.Context) {
		_ = ctx.Render("login_page.gohtml", nil)
	})

	server.Get("/wechat_login", func(ctx *context.Context) {
		ctx.Redirect("http://localhost:8081/login?appid=geekbang")
	})

	server.Get("/authed", func(ctx *context.Context) {
		// 这个就是临时授权码
		code, _ := ctx.QueryValue("code").String()
		resp, err := http.Get(fmt.Sprintf("http://localhost:8081/access_token?code=%s&appid=geekbang", code))
		// 不知道除了什么问题
		if err != nil {
			_ = ctx.RespString(http.StatusInternalServerError, "服务器故障")
			return
		}
		accessToken, _ := io.ReadAll(resp.Body)
		if len(accessToken) > 0 {
			// 拿到了 access token
			// 可以去访问资源了，就是个人信息
			// 然后设置 geekbang 的 登录态

			_ = ctx.RespString(http.StatusOK, "你已经得到授权了")
		}
	})

	if err := server.Start(":8080"); err != nil {
		panic(err)
	}
}
