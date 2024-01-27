package app2

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

func TestApp2Server(t *testing.T) {
	server := web.NewHTTPServer()
	server.Use(app2LoginMiddleware)

	server.Get("/profile", func(ctx *context.Context) {
		_ = ctx.RespString(http.StatusOK, "这是 App2 平台")
	})

	server.Get("/token", func(ctx *context.Context) {
		token, _ := ctx.QueryValue("token").String()
		resp, err := http.Post("http://localhost:8083/token/validate?token="+token,
			"application/json", nil)
		if err != nil {
			_ = ctx.RespString(http.StatusInternalServerError, "服务器故障")
			return
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "123" {
			_ = ctx.RespString(http.StatusForbidden, "非法访问")
			return
		}
		ssid := uuid.New().String()
		sessCache.Set(ssid, Session{Uid: 123}, time.Minute*15)
		ctx.SetCookie(&http.Cookie{
			Name:     "app2_ssid",
			Value:    ssid,
			Domain:   "app2.com",
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

	_ = server.Start(":8082")
}

func app2LoginMiddleware(next webHandler.HandleFunc) webHandler.HandleFunc {
	return func(ctx *context.Context) {
		if ctx.Request.URL.Path == "/login" || ctx.Request.URL.Path == "/health" {
			next(ctx)
			return
		}
		path := ctx.Request.URL.String()
		// sso 链接
		const pattern = "http://localhost:8083/check_login?redirect_uri=%sapp_id=%s"
		// URL 编码
		path = fmt.Sprintf(pattern, url.PathEscape(path), "app2")
		ck, err := ctx.Request.Cookie("app2_ssid")
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