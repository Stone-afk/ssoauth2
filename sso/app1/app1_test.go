package app1

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"io"
	"net/http"
	"net/url"
	"ssoauth2/web"
	"ssoauth2/web/context"
	webHandler "ssoauth2/web/handler"
	"testing"
	"time"
)

var sessCache = cache.New(time.Minute*15, time.Minute)

func TestApp1Server(t *testing.T) {
	server := web.NewHTTPServer()
	server.Use(app1LoginMiddleware)
	server.Get("/profile", func(ctx *context.Context) {
		_ = ctx.RespString(http.StatusOK, "这是 App1 平台")
	})
	// 就是处理从 SSO 跳回来的逻辑，也就是说，要在这里设置登录态
	// 可以直接设置吗？
	// 自己设置一个登录态
	// 第一个问题：你怎么知道，这个地方就是从 SSO 过来的？
	// 解析 token
	// 调用 SSO 的另外一个接口，去解析 token
	// 到 19:35
	server.Get("/token", func(ctx *context.Context) {
		token, _ := ctx.QueryValue("token").String()
		// 要去解析 token
		// 怎么发起调用
		// 调用sso
		resp, err := http.Post("http://localhost:8083/token/validate?token="+token,
			"application/json", nil)
		if err != nil {
			_ = ctx.RespString(http.StatusInternalServerError, "服务器故障")
			return
		}
		body, _ := io.ReadAll(resp.Body)
		// 获得 token 解析结果
		// 假设 123 是token
		if string(body) != "123" {
			_ = ctx.RespString(http.StatusForbidden, "非法访问")
			return
		}
		// 种下 session 和 cookie
		ssid := uuid.New().String()
		sessCache.Set(ssid, Session{Uid: 123}, time.Minute*15)
		ctx.SetCookie(&http.Cookie{
			Name:     "app1_ssid",
			Value:    ssid,
			Domain:   "app1.com",
			Expires:  time.Now().Add(time.Minute * 15),
			HttpOnly: true,
		})
		path, err := ctx.QueryValue("redirect_uri").String()
		if err != nil {
			_ = ctx.RespString(http.StatusInternalServerError, "服务器故障")
			return
		}
		ctx.Redirect(path)
	})

	_ = server.Start(":8081")
}

func app1LoginMiddleware(next webHandler.HandleFunc) webHandler.HandleFunc {
	return func(ctx *context.Context) {
		if ctx.Request.URL.Path == "/login" || ctx.Request.URL.Path == "/health" {
			next(ctx)
			return
		}
		path := ctx.Request.URL.String()
		// sso 链接
		const pattern = "http://localhost:8083/check_login?redirect_uri=%sapp_id=%s"
		// URL 编码
		path = fmt.Sprintf(pattern, url.PathEscape(path), "app1")
		ck, err := ctx.Request.Cookie("app1_ssid")
		if err != nil {
			// 这个地方要考虑跳转，跳过去 SSO 里面
			ctx.Redirect(path)
			//ctx.RespString(http.StatusUnauthorized, "请登录")
			return
		}
		ssid := ck.Value
		sess, ok := sessCache.Get(ssid)
		if !ok {
			ctx.Redirect(path)
			//ctx.RespString(http.StatusUnauthorized, "请登录")
			return
		}
		ctx.UserValues["sess"] = sess
		next(ctx)
	}
}

type Session struct {
	// session 里面放的内容，就是 UID，你有需要你可以继续加
	Uid uint64
}
