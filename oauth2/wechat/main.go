package wechat

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"html/template"
	"net/http"
	"ssoauth2/web"
	"ssoauth2/web/context"
	webTlp "ssoauth2/web/template"
	"time"
)

var bizRedirectUrl = map[string]string{
	"app1":     "http://app1.com:8081/token?token=%s",
	"geekbang": "http://localhost:8080/authed?code=%s",
	"app2":     "http://app2.com:8082/token?token=%s",
}

func main() {
	tpls := template.New("geekbang")
	tpls, err := tpls.ParseGlob("./template/*")
	if err != nil {
		panic(err)
	}

	// 临时授权码存储
	tmpCodes := cache.New(time.Minute*3, time.Minute*3)
	engine := webTlp.GoTemplateEngine{
		T: tpls,
	}
	server := web.NewHTTPServer(
		web.ServerWithTemplateEngine(engine))

	server.Get("/login", func(ctx *context.Context) {
		appID, _ := ctx.QueryValue("appid").String()
		_ = ctx.Render("login.gohtml", map[string]string{
			"AppID": appID,
		})
	})

	// 这个接口要加限流，比如针对 IP 的限流
	server.Post("/login", func(ctx *context.Context) {
		appid, _ := ctx.QueryValue("appid").String()
		url := bizRedirectUrl[appid]
		code := uuid.New().String()
		tmpCodes.Set(code, appid, time.Minute)
		ctx.Redirect(fmt.Sprintf(url, code))
		return
	})
	server.Get("/access_token", func(ctx *context.Context) {
		appID, _ := ctx.QueryValue("appid").String()
		code, _ := ctx.QueryValue("code").String()
		val, _ := tmpCodes.Get(code)
		if appID == val {
			accessToken := uuid.New().String()
			_ = ctx.RespString(http.StatusOK, accessToken)
		} else {
			_ = ctx.RespString(http.StatusInternalServerError, "Error")
		}
	})

	if err := server.Start(":8081"); err != nil {
		panic(err)
	}
}
